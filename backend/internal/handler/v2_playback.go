package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Jellystics/Jellystics/internal/netutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// capturedCond is true for playback rows recorded with the migration-011
// capture (audio/subtitle choice, bitrate, local/remote, video range…).
// Keep it identical to the predicate of idx_playback_captured_started_at
// (migration 012) so collectedSince can use that partial index.
func capturedCond(alias string) string {
	return `(` + alias + `."AudioStreamIndex" IS NOT NULL OR ` + alias + `."AudioLanguage" IS NOT NULL OR ` +
		alias + `."Bitrate" IS NOT NULL OR ` + alias + `."IsRemote" IS NOT NULL OR ` + alias + `."VideoRange" IS NOT NULL OR ` +
		alias + `."TranscodeReasons" IS NOT NULL OR ` + alias + `."PauseCount" > 0)`
}

// collectedSinceCache keeps, per database, the first capture date once it is
// known: it can only move if captured rows are deleted, which never happens
// in normal operation. "Nothing captured yet" is not cached.
var collectedSinceCache = struct {
	sync.Mutex
	m map[*gorm.DB]string
}{m: map[*gorm.DB]string{}}

// collectedSince is the local date of the first captured playback row (all
// users, all time), or nil when nothing has been captured yet.
func (h *V2Handler) collectedSince(c *gin.Context) (*string, error) {
	collectedSinceCache.Lock()
	cached, ok := collectedSinceCache.m[h.db]
	collectedSinceCache.Unlock()
	if ok {
		return &cached, nil
	}
	var row struct{ First *time.Time }
	if err := h.raw(c, &row, `SELECT MIN("StartedAt") AS first FROM jf_playback_activity a WHERE `+capturedCond("a")); err != nil {
		return nil, err
	}
	if row.First == nil || row.First.IsZero() {
		return nil, nil
	}
	d := localDate(*row.First)
	collectedSinceCache.Lock()
	collectedSinceCache.m[h.db] = d
	collectedSinceCache.Unlock()
	return &d, nil
}

// streamsOf is the SQL array of the MediaStreams of jf_playback_activity pa.
const streamsOf = `jsonb_array_elements(CASE WHEN jsonb_typeof(pa."MediaStreams") = 'array' THEN pa."MediaStreams" ELSE '[]'::jsonb END)`

// ---------------------------------------------------------------------------
// GET /api/v2/playback/methods
// ---------------------------------------------------------------------------

type methodDay struct {
	Date      string `json:"date"`
	Direct    int    `json:"direct"`
	Stream    int    `json:"stream"`
	Transcode int    `json:"transcode"`
}

const methodCounts = `COUNT(*) FILTER (WHERE play_method = 'DirectPlay')::int AS direct,
	COUNT(*) FILTER (WHERE play_method = 'DirectStream')::int AS stream,
	COUNT(*) FILTER (WHERE play_method = 'Transcode')::int AS transcode`

// PlaybackMethods → {totals, previous?, daily}: views per play method.
func (h *V2Handler) PlaybackMethods(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	f := p.filter(p.Range)
	var totals methodSplit
	q, args := f.sql(`SELECT ` + methodCounts + ` FROM acts WHERE is_view`)
	if v2Fail(c, h.raw(c, &totals, q, args...)) {
		return
	}
	// Bucketed like the overview: by day, by ISO week (labelled by its Monday)
	// on ranges over 400 days, where leading empty weeks are trimmed.
	week := useWeeks(p.Range)
	var rows []methodDay
	q, args = f.sql(`SELECT ` + bucketExpr(week) + ` AS date, ` + methodCounts + ` FROM acts WHERE is_view GROUP BY 1`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	by := map[string]methodDay{}
	for _, r := range rows {
		by[r.Date] = r
	}
	daily := []methodDay{}
	eachBucket(p.Range, week, func(d string) {
		r := by[d]
		r.Date = d
		daily = append(daily, r)
	})
	bucket := "day"
	if week {
		bucket = "week"
		daily = daily[leadingEmpty(len(daily), func(i int) bool { return daily[i].Direct+daily[i].Stream+daily[i].Transcode == 0 }):]
	}
	out := gin.H{"bucket": bucket, "totals": totals, "daily": daily}
	if p.Prev != nil {
		var prev methodSplit
		q, args = p.filter(*p.Prev).sql(`SELECT ` + methodCounts + ` FROM acts WHERE is_view`)
		if v2Fail(c, h.raw(c, &prev, q, args...)) {
			return
		}
		out["previous"] = prev
	}
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/v2/playback/clients
// ---------------------------------------------------------------------------

type clientRow struct {
	Client          string   `json:"client"`
	Clicks          int      `json:"clicks"`
	Views           int      `json:"views"`
	Transcodes      int      `json:"transcodes"`
	TranscodeRatio  *float64 `json:"transcodeRatio"` // transcoded views / views
	FalseStarts     int      `json:"falseStarts"`    // clicks that never became a view (< 30 s)
	FalseStartRatio *float64 `json:"falseStartRatio"`
}

func ratioPtr(n, d int) *float64 {
	if d == 0 {
		return nil
	}
	v := float64(n) / float64(d)
	return &v
}

// PlaybackClients → the 20 clients with the most plays.
func (h *V2Handler) PlaybackClients(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	var rows []clientRow
	q, args := p.filter(p.Range).sql(`
		SELECT COALESCE(NULLIF(client, ''), 'Unknown') AS client, ` + aggClicks + ` AS clicks, ` + aggPlays + ` AS views,
		       COUNT(*) FILTER (WHERE is_view AND play_method = 'Transcode')::int AS transcodes,
		       COUNT(*) FILTER (WHERE NOT is_view)::int AS false_starts
		FROM acts GROUP BY 1 ORDER BY clicks DESC, client LIMIT 20`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	out := make([]clientRow, len(rows))
	for i, r := range rows {
		r.TranscodeRatio = ratioPtr(r.Transcodes, r.Views)
		r.FalseStartRatio = ratioPtr(r.FalseStarts, r.Clicks)
		out[i] = r
	}
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/v2/playback/quality
// ---------------------------------------------------------------------------

type labelCount struct {
	Label string `json:"label"`
	Views int    `json:"views"`
}

type conversionRow struct {
	VideoFrom string `json:"videoFrom"` // source codec ("" unknown)
	VideoTo   string `json:"videoTo"`   // output codec, "copy" when the video is not re-encoded
	AudioFrom string `json:"audioFrom"`
	AudioTo   string `json:"audioTo"`
	Views     int    `json:"views"`
}

type codecPair struct {
	VideoFrom string `json:"videoFrom"`
	VideoTo   string `json:"videoTo"`
	Views     int    `json:"views"`
}

type qualityResponse struct {
	CollectedSince *string `json:"collectedSince"`
	Captured       int     `json:"captured"` // views of the range with captured fields
	Views          int     `json:"views"`
	Transcodes     int     `json:"transcodes"`
	// Resolution actually served (video views): 4K / 1080p / 720p / SD.
	Resolution struct {
		Served        map[string]int `json:"served"`
		SourceUhd     int            `json:"sourceUhd"`     // views of 4K files
		DownscaledUhd int            `json:"downscaledUhd"` // … served below 4K
	} `json:"resolution"`
	Hdr struct {
		Native     int `json:"native"`
		ToneMapped int `json:"toneMapped"` // HDR source transcoded (tone mapping)
		Sdr        int `json:"sdr"`
	} `json:"hdr"`
	Reasons     []labelCount    `json:"reasons"`     // of transcoded views
	Conversions []conversionRow `json:"conversions"` // top 12 (video + audio) conversions
	// CodecPairs is every video source → output codec pair (not truncated).
	CodecPairs []codecPair `json:"codecPairs"`
	Streams    struct {
		AudioOnly int `json:"audioOnly"`
		VideoOnly int `json:"videoOnly"`
		Both      int `json:"both"`
	} `json:"streams"`
	HwAccel []labelCount `json:"hwAccel"`
	Bitrate struct {
		Samples int      `json:"samples"`
		Avg     *float64 `json:"avg"` // bit/s
		P95     *float64 `json:"p95"`
		Buckets []int    `json:"buckets"` // < 2, 2–5, 5–10, 10–20, 20–40, ≥ 40 Mb/s
	} `json:"bitrate"`
}

// bitrateBounds are the bucket upper bounds in bit/s.
var bitrateBounds = []int64{2_000_000, 5_000_000, 10_000_000, 20_000_000, 40_000_000}

// parseReasons decodes TranscodeReasons (JSON array, or a comma-separated
// string as Jellyfin sometimes sends [Flags] enums).
func parseReasons(raw *string) []string {
	if raw == nil || *raw == "" || *raw == "null" {
		return nil
	}
	var list []string
	if json.Unmarshal([]byte(*raw), &list) == nil {
		return list
	}
	var s string
	if json.Unmarshal([]byte(*raw), &s) == nil {
		out := []string{}
		for _, r := range strings.Split(s, ",") {
			if r = strings.TrimSpace(r); r != "" {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

func isHDR(r string) bool {
	switch strings.ToUpper(r) {
	case "", "SDR", "UNKNOWN":
		return false
	}
	return true
}

func upper(s *string) string {
	if s == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(*s))
}

func sortCounts(m map[string]int) []labelCount {
	out := make([]labelCount, 0, len(m))
	for k, v := range m {
		out = append(out, labelCount{Label: k, Views: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Views != out[j].Views {
			return out[i].Views > out[j].Views
		}
		return out[i].Label < out[j].Label
	})
	return out
}

// PlaybackQuality aggregates the views of the range: served resolution, HDR
// handling, transcode reasons, codec conversions, audio/video-only
// transcodes, hardware acceleration and bitrate.
func (h *V2Handler) PlaybackQuality(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	var out qualityResponse
	var err error
	if out.CollectedSince, err = h.collectedSince(c); v2Fail(c, err) {
		return
	}
	f := p.filter(p.Range)
	var rows []struct {
		MediaType, PlayMethod        string
		SrcVideo, SrcRange, SrcAudio *string
		SrcWidth, OutWidth           *int
		OutVideo, OutAudio           *string
		VideoDirect, AudioDirect     *bool
		VideoRange, HwAccel, Reasons *string
		Captured                     bool
		N                            int
	}
	q, args := f.sql(`
		SELECT media_type, play_method, src_video, src_range, src_audio, src_width, out_width, out_video, out_audio,
		       video_direct, audio_direct, video_range, hw_accel, reasons, captured, COUNT(*)::int AS n
		FROM (
		  SELECT acts.media_type, acts.play_method,
		         v.codec AS src_video, v.range AS src_range, v.width AS src_width, au.codec AS src_audio,
		         CASE WHEN jsonb_typeof(pa."TranscodingInfo") = 'object' THEN pa."TranscodingInfo"->>'VideoCodec' END AS out_video,
		         CASE WHEN jsonb_typeof(pa."TranscodingInfo") = 'object' THEN pa."TranscodingInfo"->>'AudioCodec' END AS out_audio,
		         CASE WHEN jsonb_typeof(pa."TranscodingInfo") = 'object' THEN (pa."TranscodingInfo"->>'IsVideoDirect')::boolean END AS video_direct,
		         CASE WHEN jsonb_typeof(pa."TranscodingInfo") = 'object' THEN (pa."TranscodingInfo"->>'IsAudioDirect')::boolean END AS audio_direct,
		         CASE WHEN jsonb_typeof(pa."TranscodingInfo") = 'object' THEN NULLIF(pa."TranscodingInfo"->>'Width', '')::numeric::int END AS out_width,
		         pa."VideoRange" AS video_range, pa."HwAccel" AS hw_accel,
		         COALESCE(pa."TranscodeReasons",
		                  CASE WHEN jsonb_typeof(pa."TranscodingInfo") = 'object' THEN pa."TranscodingInfo"->'TranscodeReasons' END)::text AS reasons,
		         ` + capturedCond("pa") + ` AS captured
		  FROM acts
		  JOIN jf_playback_activity pa ON pa."Id" = acts.id
		  LEFT JOIN LATERAL (
		    SELECT s->>'Codec' AS codec, NULLIF(s->>'Width', '')::numeric::int AS width,
		           COALESCE(NULLIF(NULLIF(s->>'VideoRangeType', ''), 'Unknown'), NULLIF(NULLIF(s->>'VideoRange', ''), 'Unknown')) AS range
		    FROM ` + streamsOf + ` s
		    WHERE s->>'Type' = 'Video'
		    ORDER BY COALESCE((s->>'IsExternal')::boolean, false), COALESCE((s->>'IsDefault')::boolean, false) DESC,
		             NULLIF(s->>'Index', '')::numeric NULLS LAST
		    LIMIT 1
		  ) v ON true
		  LEFT JOIN LATERAL (
		    SELECT s->>'Codec' AS codec
		    FROM ` + streamsOf + ` s
		    WHERE s->>'Type' = 'Audio'
		    ORDER BY (NULLIF(s->>'Index', '')::numeric = pa."AudioStreamIndex") DESC NULLS LAST,
		             COALESCE((s->>'IsDefault')::boolean, false) DESC, NULLIF(s->>'Index', '')::numeric NULLS LAST
		    LIMIT 1
		  ) au ON true
		  WHERE acts.is_view
		) x
		GROUP BY 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	out.Resolution.Served = map[string]int{"4K": 0, "1080p": 0, "720p": 0, "SD": 0}
	reasons, hw := map[string]int{}, map[string]int{}
	conv := map[conversionRow]int{}
	for _, r := range rows {
		out.Views += r.N
		if r.Captured {
			out.Captured += r.N
		}
		transcode := r.PlayMethod == "Transcode"
		if transcode {
			out.Transcodes += r.N
			for _, re := range parseReasons(r.Reasons) {
				reasons[re] += r.N
			}
			if r.HwAccel != nil && *r.HwAccel != "" {
				hw[*r.HwAccel] += r.N
			}
			if r.VideoDirect != nil && r.AudioDirect != nil {
				vd, ad := *r.VideoDirect, *r.AudioDirect
				switch {
				case vd && !ad:
					out.Streams.AudioOnly += r.N
				case !vd && ad:
					out.Streams.VideoOnly += r.N
				case !vd && !ad:
					out.Streams.Both += r.N
				}
			}
		}
		if r.MediaType == "Audio" {
			continue
		}
		videoTranscoded := transcode && (r.VideoDirect == nil || !*r.VideoDirect)
		if transcode {
			k := conversionRow{VideoFrom: upper(r.SrcVideo), AudioFrom: upper(r.SrcAudio), VideoTo: upper(r.OutVideo), AudioTo: upper(r.OutAudio)}
			if r.VideoDirect != nil && *r.VideoDirect {
				k.VideoTo = "copy"
			}
			if r.AudioDirect != nil && *r.AudioDirect {
				k.AudioTo = "copy"
			}
			conv[k] += r.N
		}
		served := r.SrcWidth
		if videoTranscoded && r.OutWidth != nil && *r.OutWidth > 0 {
			served = r.OutWidth
		}
		if l := resolutionLabel(served); l != nil {
			out.Resolution.Served[*l] += r.N
		}
		if src := resolutionLabel(r.SrcWidth); src != nil && *src == "4K" {
			out.Resolution.SourceUhd += r.N
			if l := resolutionLabel(served); l != nil && *l != "4K" {
				out.Resolution.DownscaledUhd += r.N
			}
		}
		rng := ""
		if r.VideoRange != nil {
			rng = *r.VideoRange
		} else if r.SrcRange != nil {
			rng = *r.SrcRange
		}
		switch {
		case isHDR(rng) && videoTranscoded:
			out.Hdr.ToneMapped += r.N
		case isHDR(rng):
			out.Hdr.Native += r.N
		case strings.EqualFold(rng, "SDR"):
			out.Hdr.Sdr += r.N
		}
	}
	out.Reasons = sortCounts(reasons)
	out.HwAccel = sortCounts(hw)
	out.Conversions = make([]conversionRow, 0, len(conv))
	for k, n := range conv {
		k.Views = n
		out.Conversions = append(out.Conversions, k)
	}
	sort.Slice(out.Conversions, func(i, j int) bool {
		a, b := out.Conversions[i], out.Conversions[j]
		if a.Views != b.Views {
			return a.Views > b.Views
		}
		return fmt.Sprint(a) < fmt.Sprint(b)
	})
	pairs := map[codecPair]int{}
	for _, cv := range out.Conversions {
		pairs[codecPair{VideoFrom: cv.VideoFrom, VideoTo: cv.VideoTo}] += cv.Views
	}
	out.CodecPairs = make([]codecPair, 0, len(pairs))
	for k, n := range pairs {
		k.Views = n
		out.CodecPairs = append(out.CodecPairs, k)
	}
	sort.Slice(out.CodecPairs, func(i, j int) bool {
		a, b := out.CodecPairs[i], out.CodecPairs[j]
		if a.Views != b.Views {
			return a.Views > b.Views
		}
		if a.VideoFrom != b.VideoFrom {
			return a.VideoFrom < b.VideoFrom
		}
		return a.VideoTo < b.VideoTo
	})
	if len(out.Conversions) > 12 {
		out.Conversions = out.Conversions[:12]
	}

	cases := "CASE"
	for i, b := range bitrateBounds {
		cases += fmt.Sprintf(" WHEN pa.\"Bitrate\" < %d THEN %d", b, i)
	}
	cases += fmt.Sprintf(" ELSE %d END", len(bitrateBounds))
	var buckets []struct{ Bucket, N int }
	q, args = f.sql(`SELECT ` + cases + ` AS bucket, COUNT(*)::int AS n
		FROM acts JOIN jf_playback_activity pa ON pa."Id" = acts.id
		WHERE acts.is_view AND pa."Bitrate" > 0 GROUP BY 1`)
	if v2Fail(c, h.raw(c, &buckets, q, args...)) {
		return
	}
	out.Bitrate.Buckets = make([]int, len(bitrateBounds)+1)
	for _, b := range buckets {
		out.Bitrate.Buckets[b.Bucket] = b.N
		out.Bitrate.Samples += b.N
	}
	var stats struct{ Avg, P95 *float64 }
	q, args = f.sql(`SELECT AVG(pa."Bitrate")::float8 AS avg,
		       (percentile_cont(0.95) WITHIN GROUP (ORDER BY pa."Bitrate"))::float8 AS p95
		FROM acts JOIN jf_playback_activity pa ON pa."Id" = acts.id
		WHERE acts.is_view AND pa."Bitrate" > 0`)
	if v2Fail(c, h.raw(c, &stats, q, args...)) {
		return
	}
	out.Bitrate.Avg, out.Bitrate.P95 = stats.Avg, stats.P95
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/v2/playback/network
// ---------------------------------------------------------------------------

type placeRow struct {
	Country string  `json:"country"`
	City    *string `json:"city"`
	Views   int     `json:"views"`
}

// PlaybackNetwork → {collectedSince, captured, local, remote, unknown,
// places}: views by network origin, and remote views per geolocated place
// (ip_geo_cache; IPs not geolocated yet are left out of places).
func (h *V2Handler) PlaybackNetwork(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	since, err := h.collectedSince(c)
	if v2Fail(c, err) {
		return
	}
	var rows []struct {
		Remote   *bool
		Endpoint *string
		N        int
	}
	q, args := p.filter(p.Range).sql(`
		SELECT pa."IsRemote" AS remote, CASE WHEN pa."IsRemote" THEN pa."RemoteEndPoint" END AS endpoint, COUNT(*)::int AS n
		FROM acts JOIN jf_playback_activity pa ON pa."Id" = acts.id
		WHERE acts.is_view GROUP BY 1, 2`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	var local, remote, unknown int
	perIP := map[string]int{}
	for _, r := range rows {
		switch {
		case r.Remote == nil:
			unknown += r.N
		case !*r.Remote:
			local += r.N
		default:
			remote += r.N
			if r.Endpoint != nil {
				if a, ok := netutil.Parse(*r.Endpoint); ok {
					perIP[a.String()] += r.N
				}
			}
		}
	}
	places := []placeRow{}
	if len(perIP) > 0 {
		ips := make([]string, 0, len(perIP))
		for ip := range perIP {
			ips = append(ips, ip)
		}
		var geo []struct {
			IP            string
			Country, City *string
		}
		if v2Fail(c, h.raw(c, &geo, `SELECT ip, country, city FROM ip_geo_cache WHERE ip IN ? AND country IS NOT NULL`, ips)) {
			return
		}
		type key struct{ country, city string }
		agg := map[key]int{}
		for _, g := range geo {
			k := key{country: *g.Country}
			if g.City != nil {
				k.city = *g.City
			}
			agg[k] += perIP[g.IP]
		}
		for k, n := range agg {
			pr := placeRow{Country: k.country, Views: n}
			if k.city != "" {
				city := k.city
				pr.City = &city
			}
			places = append(places, pr)
		}
		sort.Slice(places, func(i, j int) bool {
			a, b := places[i], places[j]
			if a.Views != b.Views {
				return a.Views > b.Views
			}
			if a.Country != b.Country {
				return a.Country < b.Country
			}
			return fmt.Sprint(a.City) < fmt.Sprint(b.City)
		})
		if len(places) > 10 {
			places = places[:10]
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"collectedSince": since, "captured": local + remote,
		"local": local, "remote": remote, "unknown": unknown, "places": places,
	})
}

// ---------------------------------------------------------------------------
// GET /api/v2/users/:id/languages — VO / VF and subtitles of one user
// ---------------------------------------------------------------------------

type languagesResponse struct {
	CollectedSince *string `json:"collectedSince"`
	Captured       int     `json:"captured"` // captured movie / episode views (migration 011)
	Split          struct {
		OriginalSubtitled int `json:"originalSubtitled"`
		Original          int `json:"original"`
		Dubbed            int `json:"dubbed"`
		Unknown           int `json:"unknown"`
	} `json:"split"`
	Audio       []labelCount `json:"audio"`     // audio language → views ("" unknown)
	Subtitles   []labelCount `json:"subtitles"` // subtitle language → views (subtitled views only)
	SubtitleOff int          `json:"subtitleOff"`
}

// mainAudio is true for an audio stream s that can be the original track:
// commentaries, audio descriptions and hearing-impaired mixes are skipped.
const mainAudio = `s->>'Type' = 'Audio'
		  AND NOT COALESCE(CASE WHEN jsonb_typeof(s->'IsHearingImpaired') = 'boolean' THEN (s->>'IsHearingImpaired')::boolean END, false)
		  AND lower(COALESCE(s->>'Title', '') || ' ' || COALESCE(s->>'DisplayTitle', '')) !~ '(comment|description|descriptive)'`

// knownLang is the lower-cased language expression e, NULL when empty or "und".
func knownLang(e string) string {
	return `NULLIF(NULLIF(lower(` + e + `), ''), 'und')`
}

// UserLanguages: "original" means the language played is the language of
// the first main audio track of the file — Jellyfin has no original-language
// field, and files usually list the original track first. Views stay
// unknown when the file has a single audio language (nothing to choose),
// when the first track or the played track has no language, and when the
// client did not report its audio track.
func (h *V2Handler) UserLanguages(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	p.UserID = c.Param("id")
	var out languagesResponse
	var err error
	if out.CollectedSince, err = h.collectedSince(c); v2Fail(c, err) {
		return
	}
	var rows []struct {
		Lang, SubLang *string
		Subtitled     bool
		Original      *bool
		N             int
	}
	q, args := p.filter(p.Range).sql(`
		SELECT lower(NULLIF(pa."AudioLanguage", '')) AS lang, lower(NULLIF(pa."SubtitleLanguage", '')) AS sub_lang,
		       (pa."SubtitleStreamIndex" IS NOT NULL) AS subtitled,
		       CASE WHEN fs.langs >= 2 AND fs.first_lang IS NOT NULL
		                 AND COALESCE(` + knownLang(`pa."AudioLanguage"`) + `, ps.lang) IS NOT NULL
		            THEN COALESCE(` + knownLang(`pa."AudioLanguage"`) + `, ps.lang) = fs.first_lang END AS original,
		       COUNT(*)::int AS n
		FROM acts
		JOIN jf_playback_activity pa ON pa."Id" = acts.id
		LEFT JOIN LATERAL (
		  SELECT (array_agg(` + knownLang(`s->>'Language'`) + ` ORDER BY NULLIF(s->>'Index', '')::numeric NULLS LAST))[1] AS first_lang,
		         COUNT(DISTINCT ` + knownLang(`s->>'Language'`) + `)::int AS langs
		  FROM ` + streamsOf + ` s WHERE ` + mainAudio + `
		) fs ON true
		LEFT JOIN LATERAL (
		  SELECT ` + knownLang(`s->>'Language'`) + ` AS lang FROM ` + streamsOf + ` s
		  WHERE s->>'Type' = 'Audio' AND pa."AudioStreamIndex" IS NOT NULL AND NULLIF(s->>'Index', '')::numeric = pa."AudioStreamIndex" LIMIT 1
		) ps ON true
		WHERE acts.is_view AND acts.media_type IN ('Movie', 'Episode') AND ` + capturedCond("pa") + `
		GROUP BY 1, 2, 3, 4`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	audio, subs := map[string]int{}, map[string]int{}
	for _, r := range rows {
		out.Captured += r.N
		l := ""
		if r.Lang != nil {
			l = *r.Lang
		}
		audio[l] += r.N
		if r.Subtitled {
			s := ""
			if r.SubLang != nil {
				s = *r.SubLang
			}
			subs[s] += r.N
		} else {
			out.SubtitleOff += r.N
		}
		switch {
		case r.Original == nil:
			out.Split.Unknown += r.N
		case *r.Original && r.Subtitled:
			out.Split.OriginalSubtitled += r.N
		case *r.Original:
			out.Split.Original += r.N
		default:
			out.Split.Dubbed += r.N
		}
	}
	out.Audio, out.Subtitles = sortCounts(audio), sortCounts(subs)
	c.JSON(http.StatusOK, out)
}
