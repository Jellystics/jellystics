package handler_test

import (
	"testing"
)

func TestV2PlaybackMethodsClients(t *testing.T) {
	db := v2Seeded(t)

	var m struct {
		Totals   struct{ Direct, Stream, Transcode int }
		Previous *struct{ Direct, Stream, Transcode int }
		Daily    []struct {
			Date                      string
			Direct, Stream, Transcode int
		}
	}
	v5JSON(t, db, "/api/v2/playback/methods?compare=1", &m)
	if m.Totals.Direct != 2 || m.Totals.Stream != 3 || m.Totals.Transcode != 1 || m.Previous == nil || m.Previous.Direct != 0 || len(m.Daily) != 30 {
		t.Fatalf("methods = %+v", m)
	}
	day3 := v2Day3.Format("2006-01-02")
	for _, d := range m.Daily {
		if d.Date == day3 && (d.Direct != 1 || d.Transcode != 1) {
			t.Errorf("day3 = %+v", d)
		}
	}

	var clients []struct {
		Client                                 string
		Clicks, Views, Transcodes, FalseStarts int
		TranscodeRatio, FalseStartRatio        *float64
	}
	v5JSON(t, db, "/api/v2/playback/clients", &clients)
	if len(clients) != 2 {
		t.Fatalf("clients = %+v", clients)
	}
	a, w := clients[0], clients[1]
	if a.Client != "Android" || a.Clicks != 4 || a.Views != 2 || a.Transcodes != 1 || a.FalseStarts != 2 ||
		!ptrNear(a.TranscodeRatio, 0.5) || !ptrNear(a.FalseStartRatio, 0.5) {
		t.Errorf("android = %+v", a)
	}
	if w.Client != "Web" || w.Clicks != 4 || w.Views != 4 || !ptrNear(w.TranscodeRatio, 0) || !ptrNear(w.FalseStartRatio, 0) {
		t.Errorf("web = %+v", w)
	}
}

type qualityJSON struct {
	CollectedSince *string
	Captured       int
	Views          int
	Transcodes     int
	Resolution     struct {
		Served                   map[string]int
		SourceUhd, DownscaledUhd int
	}
	Hdr     struct{ Native, ToneMapped, Sdr int }
	Reasons []struct {
		Label string
		Views int
	}
	Conversions []struct {
		VideoFrom, VideoTo, AudioFrom, AudioTo string
		Views                                  int
	}
	CodecPairs []struct {
		VideoFrom, VideoTo string
		Views              int
	}
	Streams struct{ AudioOnly, VideoOnly, Both int }
	HwAccel []struct {
		Label string
		Views int
	}
	Bitrate struct {
		Samples  int
		Avg, P95 *float64
		Buckets  []int
	}
}

func TestV2PlaybackQualityNetworkLanguages(t *testing.T) {
	db := v2Seeded(t)

	var q qualityJSON
	v5JSON(t, db, "/api/v2/playback/quality", &q)
	if q.CollectedSince != nil || q.Captured != 0 || q.Views != 6 || q.Transcodes != 1 || q.Bitrate.Samples != 0 || q.Bitrate.Avg != nil {
		t.Errorf("quality before capture = %+v", q)
	}

	// r1: direct 1080p SDR, local. r2: 4K HDR10 transcoded to 1080p H.264, remote.
	mustExec(t, db, `UPDATE jf_playback_activity SET
		"MediaStreams" = '[{"Type":"Video","Codec":"h264","Width":1920,"VideoRange":"SDR","Index":0},{"Type":"Audio","Codec":"aac","Index":1}]',
		"Bitrate" = 25000000, "IsRemote" = false, "VideoRange" = 'SDR', "RemoteEndPoint" = '192.168.1.10'
		WHERE "Id" = 'r1'`)
	mustExec(t, db, `UPDATE jf_playback_activity SET
		"MediaStreams" = '[{"Type":"Video","Codec":"hevc","Width":3840,"VideoRangeType":"HDR10","Index":0},{"Type":"Audio","Codec":"truehd","Index":1,"IsDefault":true}]',
		"TranscodingInfo" = '{"VideoCodec":"h264","AudioCodec":"aac","IsVideoDirect":false,"IsAudioDirect":false,"Width":1920,"TranscodeReasons":["ContainerNotSupported"]}',
		"TranscodeReasons" = '["VideoRangeTypeNotSupported","AudioCodecNotSupported"]',
		"Bitrate" = 8000000, "HwAccel" = 'vaapi', "IsRemote" = true, "RemoteEndPoint" = '203.0.113.5:8096', "VideoRange" = 'HDR10'
		WHERE "Id" = 'r2'`)

	v5JSON(t, db, "/api/v2/playback/quality", &q)
	if q.CollectedSince == nil || *q.CollectedSince != v2Day3.Format("2006-01-02") || q.Captured != 2 || q.Views != 6 || q.Transcodes != 1 {
		t.Errorf("quality header = %+v", q)
	}
	if q.Resolution.Served["1080p"] != 2 || q.Resolution.Served["4K"] != 0 || q.Resolution.SourceUhd != 1 || q.Resolution.DownscaledUhd != 1 {
		t.Errorf("resolution = %+v", q.Resolution)
	}
	if q.Hdr.ToneMapped != 1 || q.Hdr.Native != 0 || q.Hdr.Sdr != 1 {
		t.Errorf("hdr = %+v", q.Hdr)
	}
	// The captured column wins over the legacy TranscodingInfo reasons.
	if len(q.Reasons) != 2 || q.Reasons[0].Label != "AudioCodecNotSupported" || q.Reasons[1].Label != "VideoRangeTypeNotSupported" || q.Reasons[0].Views != 1 {
		t.Errorf("reasons = %+v", q.Reasons)
	}
	if len(q.Conversions) != 1 || q.Conversions[0].VideoFrom != "HEVC" || q.Conversions[0].VideoTo != "H264" ||
		q.Conversions[0].AudioFrom != "TRUEHD" || q.Conversions[0].AudioTo != "AAC" || q.Conversions[0].Views != 1 {
		t.Errorf("conversions = %+v", q.Conversions)
	}
	if len(q.CodecPairs) != 1 || q.CodecPairs[0].VideoFrom != "HEVC" || q.CodecPairs[0].VideoTo != "H264" || q.CodecPairs[0].Views != 1 {
		t.Errorf("codec pairs = %+v", q.CodecPairs)
	}
	if q.Streams.Both != 1 || q.Streams.AudioOnly != 0 || len(q.HwAccel) != 1 || q.HwAccel[0].Label != "vaapi" {
		t.Errorf("streams = %+v, hw = %+v", q.Streams, q.HwAccel)
	}
	if q.Bitrate.Samples != 2 || !ptrNear(q.Bitrate.Avg, 16_500_000) || !ptrNear(q.Bitrate.P95, 24_150_000) ||
		q.Bitrate.Buckets[2] != 1 || q.Bitrate.Buckets[4] != 1 {
		t.Errorf("bitrate = %+v", q.Bitrate)
	}

	var net struct {
		CollectedSince                   *string
		Captured, Local, Remote, Unknown int
		Places                           []struct {
			Country string
			City    *string
			Views   int
		}
	}
	v5JSON(t, db, "/api/v2/playback/network", &net)
	if net.Local != 1 || net.Remote != 1 || net.Unknown != 4 || net.Captured != 2 || len(net.Places) != 0 {
		t.Errorf("network before geo = %+v", net)
	}
	mustExec(t, db, `INSERT INTO ip_geo_cache (ip, country, city) VALUES ('203.0.113.5', 'FR', 'Lyon')`)
	v5JSON(t, db, "/api/v2/playback/network", &net)
	if len(net.Places) != 1 || net.Places[0].Country != "FR" || net.Places[0].City == nil || *net.Places[0].City != "Lyon" || net.Places[0].Views != 1 {
		t.Errorf("places = %+v", net.Places)
	}

	// Languages (Alice): e1 English (first track) with French subtitles,
	// e2 French dub (second track), e3 English with no stream list.
	streams := `[{"Type":"Audio","Index":1,"Language":"eng"},{"Type":"Audio","Index":2,"Language":"fre"},{"Type":"Subtitle","Index":3,"Language":"fre"}]`
	mustExec(t, db, `UPDATE jf_playback_activity SET "MediaStreams" = ?, "AudioStreamIndex" = 1, "AudioLanguage" = 'eng',
		"SubtitleStreamIndex" = 3, "SubtitleLanguage" = 'fre' WHERE "Id" = 'r4'`, streams)
	mustExec(t, db, `UPDATE jf_playback_activity SET "MediaStreams" = ?, "AudioStreamIndex" = 2, "AudioLanguage" = 'fre' WHERE "Id" = 'r5'`, streams)
	mustExec(t, db, `UPDATE jf_playback_activity SET "AudioLanguage" = 'eng' WHERE "Id" = 'r6'`)
	var lang struct {
		CollectedSince *string
		Captured       int
		Split          struct{ OriginalSubtitled, Original, Dubbed, Unknown int }
		Audio          []struct {
			Label string
			Views int
		}
		Subtitles []struct {
			Label string
			Views int
		}
		SubtitleOff int
	}
	v5JSON(t, db, "/api/v2/users/u1/languages", &lang)
	// r1 (captured movie, no audio track reported) and r6 count as unknown.
	if lang.Captured != 4 || lang.Split.OriginalSubtitled != 1 || lang.Split.Dubbed != 1 || lang.Split.Unknown != 2 || lang.Split.Original != 0 {
		t.Errorf("languages = %+v", lang)
	}
	if len(lang.Audio) != 3 || lang.Audio[0].Label != "eng" || lang.Audio[0].Views != 2 || len(lang.Subtitles) != 1 ||
		lang.Subtitles[0].Label != "fre" || lang.SubtitleOff != 3 || lang.CollectedSince == nil {
		t.Errorf("languages lists = %+v", lang)
	}
	v5JSON(t, db, "/api/v2/users/u2/languages", &lang)
	if lang.Captured != 1 || lang.Split.Unknown != 1 || lang.Split.Dubbed+lang.Split.Original != 0 { // r2: captured, no audio track
		t.Errorf("languages u2 = %+v", lang)
	}
	// The user detail route is still served next to /languages.
	if w := v5Get(t, db, "/api/v2/users/u1"); w.Code != 200 {
		t.Errorf("/users/u1 status %d", w.Code)
	}
}

// VO / VF heuristic: a single-language file or an "und" first track stays
// unknown, commentaries never count as the original track.
func TestV2UserLanguagesHeuristic(t *testing.T) {
	db := v2Seeded(t)
	set := func(id, streams string, idx int, lang string) {
		t.Helper()
		mustExec(t, db, `UPDATE jf_playback_activity SET "MediaStreams" = ?, "AudioStreamIndex" = ?, "AudioLanguage" = NULLIF(?, '') WHERE "Id" = ?`,
			streams, idx, lang, id)
	}
	// r1: only English tracks (main + commentary) → unknown.
	set("r1", `[{"Type":"Audio","Index":1,"Language":"eng"},{"Type":"Audio","Index":2,"Language":"eng","Title":"Director's commentary"}]`, 1, "eng")
	// r4: first track "und" → unknown even though French is played.
	set("r4", `[{"Type":"Audio","Index":1,"Language":"und"},{"Type":"Audio","Index":2,"Language":"fre"}]`, 2, "fre")
	// r5: a French commentary listed first is skipped: the original is
	// English, the French dub is detected.
	set("r5", `[{"Type":"Audio","Index":1,"Language":"fre","Title":"Commentaire / Commentary"},{"Type":"Audio","Index":2,"Language":"eng"},`+
		`{"Type":"Audio","Index":3,"Language":"fre"},{"Type":"Audio","Index":4,"Language":"spa","IsHearingImpaired":true}]`, 3, "fre")
	// r6: English original played, language resolved from the stream index.
	set("r6", `[{"Type":"Audio","Index":1,"Language":"eng"},{"Type":"Audio","Index":2,"Language":"fre"}]`, 1, "")
	var lang struct {
		Captured int
		Split    struct{ OriginalSubtitled, Original, Dubbed, Unknown int }
	}
	v5JSON(t, db, "/api/v2/users/u1/languages", &lang)
	if lang.Captured != 4 || lang.Split.Unknown != 2 || lang.Split.Dubbed != 1 || lang.Split.Original != 1 || lang.Split.OriginalSubtitled != 0 {
		t.Errorf("languages = %+v", lang)
	}
}
