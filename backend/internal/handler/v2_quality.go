package handler

import (
	"time"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
)

// Data quality: where the plays of the period come from and how precise
// they are. Source keys, from most to least precise:
//
//	plugin             Jellystics plugin: exact Jellyfin playback events
//	tracker            Jellystics tracker: /Sessions polled every 5 s
//	playbackReporting  Playback Reporting imports (legacy sync, TSV backup,
//	                   plugin import-pbr): estimated durations, some repaired
//	otherImports       other imports (Jellystics history coming back from the plugin...)
//	restore            restored backup rows without a source
//	other              anything else
var dataQualityKeys = []string{"plugin", "tracker", "playbackReporting", "otherImports", "restore", "other"}

func dataQualityKey(source string, imported bool) string {
	switch source {
	case repository.SourceJellysticsPlugin:
		return "plugin"
	case repository.SourceTracker:
		return "tracker"
	case repository.SourceLegacyPBR, repository.SourceImportPBR, "import":
		return "playbackReporting"
	case "restore":
		return "restore"
	}
	if imported {
		return "otherImports"
	}
	return "other"
}

// DataQualityCounts are the counters of one source (or of the period).
type DataQualityCounts struct {
	Plays        int   `json:"plays"`
	Views        int   `json:"views"`
	WatchSeconds int64 `json:"watchSeconds"`
	// Repaired: imported plays whose impossible duration Jellystics replaced.
	Repaired int `json:"repaired"`
	// Captured: plays with playback details (audio / subtitle choice,
	// bitrate, local / remote, video range, pauses: see capturedCond).
	Captured int `json:"captured"`
	// Positions: plays with a known stop position (completion is exact).
	Positions int `json:"positions"`
	// Streams: plays with the audio / subtitle track played.
	Streams int `json:"streams"`
	// Transcodes / TranscodeDetailed: transcoded plays, and those whose
	// transcode details (codecs, reasons) are known.
	Transcodes        int `json:"transcodes"`
	TranscodeDetailed int `json:"transcodeDetailed"`
}

func (a *DataQualityCounts) add(b DataQualityCounts) {
	a.Plays += b.Plays
	a.Views += b.Views
	a.WatchSeconds += b.WatchSeconds
	a.Repaired += b.Repaired
	a.Captured += b.Captured
	a.Positions += b.Positions
	a.Streams += b.Streams
	a.Transcodes += b.Transcodes
	a.TranscodeDetailed += b.TranscodeDetailed
}

// DataQualitySource is one source of the breakdown. Shares are 0..1 of the
// period totals (0 when the period is empty).
type DataQualitySource struct {
	Key string `json:"key"`
	DataQualityCounts
	PlayShare  float64 `json:"playShare"`
	WatchShare float64 `json:"watchShare"`
}

// DataQuality is GET /api/v2/data-quality.
type DataQuality struct {
	Totals  DataQualityCounts   `json:"totals"`
	Sources []DataQualitySource `json:"sources"` // every key, fixed order
	// PluginCollectedSince: local date of the first play recorded by the
	// Jellystics plugin (all time); null without plugin plays.
	PluginCollectedSince *string `json:"pluginCollectedSince"`
	// CollectedSince: local date of the first play with playback details.
	CollectedSince *string `json:"collectedSince"`
	// Undated: plays (all time) without a usable start date, excluded from
	// every statistic.
	Undated int `json:"undated"`
}

// DataQuality serves GET /api/v2/data-quality (period, userId, libraryId,
// type filters of the v2 contract).
func (h *V2Handler) DataQuality(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	var rows []struct {
		Source   string
		Imported bool
		DataQualityCounts
	}
	q, args := p.filter(p.Range).sql(`
		SELECT pa.source AS source, COALESCE(pa.imported, false) AS imported,
		       COUNT(*)::int AS plays,
		       COUNT(*) FILTER (WHERE acts.is_view)::int AS views,
		       COALESCE(SUM(acts.duration), 0)::bigint AS watch_seconds,
		       COUNT(*) FILTER (WHERE pa."DurationRepaired")::int AS repaired,
		       COUNT(*) FILTER (WHERE ` + capturedCond("pa") + `)::int AS captured,
		       COUNT(*) FILTER (WHERE pa."PositionTicksEnd" IS NOT NULL)::int AS positions,
		       COUNT(*) FILTER (WHERE pa."AudioStreamIndex" IS NOT NULL OR pa."AudioLanguage" IS NOT NULL
		                           OR pa."SubtitleStreamIndex" IS NOT NULL OR pa."SubtitleLanguage" IS NOT NULL)::int AS streams,
		       COUNT(*) FILTER (WHERE acts.play_method = 'Transcode')::int AS transcodes,
		       COUNT(*) FILTER (WHERE acts.play_method = 'Transcode'
		                          AND (jsonb_typeof(pa."TranscodingInfo") = 'object' OR pa."TranscodeReasons" IS NOT NULL))::int AS transcode_detailed
		FROM acts
		JOIN jf_playback_activity pa ON pa."Id" = acts.id
		GROUP BY 1, 2`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}

	out := DataQuality{Sources: make([]DataQualitySource, len(dataQualityKeys))}
	idx := map[string]int{}
	for i, k := range dataQualityKeys {
		out.Sources[i].Key = k
		idx[k] = i
	}
	for _, r := range rows {
		out.Sources[idx[dataQualityKey(r.Source, r.Imported)]].add(r.DataQualityCounts)
		out.Totals.add(r.DataQualityCounts)
	}
	for i := range out.Sources {
		s := &out.Sources[i]
		if out.Totals.Plays > 0 {
			s.PlayShare = float64(s.Plays) / float64(out.Totals.Plays)
		}
		if out.Totals.WatchSeconds > 0 {
			s.WatchShare = float64(s.WatchSeconds) / float64(out.Totals.WatchSeconds)
		}
	}

	var meta struct {
		PluginFirst *time.Time
		Undated     int
	}
	if v2Fail(c, h.raw(c, &meta, `
		SELECT (SELECT MIN("StartedAt") FROM jf_playback_activity WHERE source = ?) AS plugin_first,
		       (SELECT COUNT(*) FROM jf_playback_activity WHERE "StartedAt" IS NULL)::int AS undated`,
		repository.SourceJellysticsPlugin)) {
		return
	}
	if meta.PluginFirst != nil && !meta.PluginFirst.IsZero() {
		d := localDate(*meta.PluginFirst)
		out.PluginCollectedSince = &d
	}
	out.Undated = meta.Undated
	since, err := h.collectedSince(c)
	if v2Fail(c, err) {
		return
	}
	out.CollectedSince = since
	c.JSON(200, out)
}
