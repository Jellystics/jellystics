package sync

import (
	"encoding/json"
	"testing"

	"github.com/Jellystics/Jellystics/internal/jellyfin"
)

func TestIsRemoteEndpoint(t *testing.T) {
	cases := []struct {
		in   string
		want *bool // nil = unknown
	}{
		{"192.168.1.20", ptr(false)},
		{"10.0.0.5", ptr(false)},
		{"172.16.4.1", ptr(false)},
		{"172.32.0.1", ptr(true)}, // just outside 172.16/12
		{"127.0.0.1", ptr(false)},
		{"169.254.10.2", ptr(false)},
		{"::1", ptr(false)},
		{"fe80::1", ptr(false)},
		{"fd12:3456::7", ptr(false)}, // ULA fc00::/7
		{"::ffff:192.168.1.5", ptr(false)},
		{"[::1]:8096", ptr(false)},
		{"192.168.1.20:5555", ptr(false)},
		{"203.0.113.9", ptr(true)},
		{"100.64.1.2", ptr(true)}, // CGNAT / Tailscale counts as remote
		{"203.0.113.9:443", ptr(true)},
		{"2a01:e0a::1", ptr(true)},
		{"::ffff:8.8.8.8", ptr(true)},
		{"", nil},
		{"not-an-ip", nil},
	}
	for _, c := range cases {
		got := isRemoteEndpoint(c.in)
		if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
			t.Errorf("isRemoteEndpoint(%q) = %v, want %v", c.in, deref0(got), deref0(c.want))
		}
	}
}

func deref0(b *bool) any {
	if b == nil {
		return nil
	}
	return *b
}

func captureMovie(sess, item string, posMs int64, paused bool) jellyfin.SessionInfo {
	s := movieAt(sess, item, posMs, paused)
	s.RemoteEndPoint = sp("203.0.113.9")
	s.NowPlayingItem.MediaStreams = []jellyfin.MediaStream{
		{Index: 0, Type: "Video", VideoRange: "HDR", VideoRangeType: "HDR10", BitRate: ptr(int64(8_000_000))},
		{Index: 1, Type: "Audio", Language: sp("eng")},
		{Index: 2, Type: "Audio", Language: sp("fre"), BitRate: ptr(int64(640_000))},
		{Index: 3, Type: "Subtitle", Language: sp("eng")},
	}
	s.PlayState.AudioStreamIndex = ptr(2)
	s.PlayState.SubtitleStreamIndex = ptr(3)
	return s
}

func TestTracker_CapturesAudioSubtitleRangeAndLocality(t *testing.T) {
	d := newDriver(t)
	d.tick(0, captureMovie("s1", "m1", 0, false))
	d.tick(5, captureMovie("s1", "m1", 5000, false))
	d.tick(5) // stopped
	p := d.plays()
	if len(p) != 1 {
		t.Fatalf("plays = %d", len(p))
	}
	r := p[0]
	if r.AudioStreamIndex == nil || *r.AudioStreamIndex != 2 || r.AudioLanguage == nil || *r.AudioLanguage != "fre" {
		t.Errorf("audio = %v %v", r.AudioStreamIndex, r.AudioLanguage)
	}
	if r.SubtitleStreamIndex == nil || *r.SubtitleStreamIndex != 3 || r.SubtitleLanguage == nil || *r.SubtitleLanguage != "eng" {
		t.Errorf("subtitle = %v %v", r.SubtitleStreamIndex, r.SubtitleLanguage)
	}
	if r.VideoRange == nil || *r.VideoRange != "HDR10" {
		t.Errorf("range = %v", r.VideoRange)
	}
	if r.IsRemote == nil || !*r.IsRemote {
		t.Errorf("IsRemote = %v", r.IsRemote)
	}
	if r.PauseCount != 0 || r.TranscodeReasons != nil || r.HwAccel != nil {
		t.Errorf("direct play should have no transcode data: %+v", r)
	}
	// No MediaSource: video 8 Mbps + selected audio (index 2) 640 kbps.
	if r.Bitrate == nil || *r.Bitrate != 8_640_000 {
		t.Errorf("direct-play bitrate = %v", r.Bitrate)
	}
}

func TestTracker_DirectPlayBitratePrefersMediaSource(t *testing.T) {
	d := newDriver(t)
	s := captureMovie("s1", "m1", 0, false)
	s.NowPlayingItem.MediaSources = []jellyfin.MediaSource{{Bitrate: ptr(int64(12_000_000))}}
	d.tick(0, s)
	if b := d.watchdog("s1").Bitrate; b == nil || *b != 12_000_000 {
		t.Fatalf("bitrate = %v", b)
	}
}

func TestTracker_TranscodeBitrateOverridesAndSurvivesMediaFallback(t *testing.T) {
	d := newDriver(t)
	s := captureMovie("s1", "m1", 0, false)
	d.tick(0, s) // no TranscodingInfo yet: media fallback recorded
	s = captureMovie("s1", "m1", 5000, false)
	s.TranscodingInfo = &jellyfin.TranscodingInfo{Bitrate: ptr(int64(3_000_000))}
	d.tick(5, s)
	d.tick(5, captureMovie("s1", "m1", 5000, true)) // TranscodingInfo gone
	if b := d.watchdog("s1").Bitrate; b == nil || *b != 3_000_000 {
		t.Fatalf("bitrate = %v, want transcode value", b)
	}
}

func TestTracker_AudioFallsBackToDefaultThenFirst(t *testing.T) {
	d := newDriver(t)
	s := captureMovie("s1", "m1", 0, false)
	s.PlayState.AudioStreamIndex = nil
	s.NowPlayingItem.MediaStreams[2].IsDefault = true
	d.tick(0, s)
	wd := d.watchdog("s1")
	if wd.AudioStreamIndex == nil || *wd.AudioStreamIndex != 2 || *wd.AudioLanguage != "fre" {
		t.Fatalf("default audio: %v %v", wd.AudioStreamIndex, wd.AudioLanguage)
	}
	d2 := newDriver(t)
	s = captureMovie("s1", "m1", 0, false)
	s.PlayState.AudioStreamIndex = nil
	d2.tick(0, s)
	wd = d2.watchdog("s1")
	if wd.AudioStreamIndex == nil || *wd.AudioStreamIndex != 1 || *wd.AudioLanguage != "eng" {
		t.Fatalf("first audio: %v %v", wd.AudioStreamIndex, wd.AudioLanguage)
	}
	// Subtitles never fall back.
	d3 := newDriver(t)
	s = captureMovie("s1", "m1", 0, false)
	s.PlayState.SubtitleStreamIndex = nil
	d3.tick(0, s)
	if wd := d3.watchdog("s1"); wd.SubtitleStreamIndex != nil || wd.SubtitleLanguage != nil {
		t.Fatalf("subtitle must not fall back")
	}
}

func TestTracker_AudioKeepsLastKnownValue(t *testing.T) {
	d := newDriver(t)
	d.tick(0, captureMovie("s1", "m1", 0, false))
	s := captureMovie("s1", "m1", 5000, false)
	s.PlayState.AudioStreamIndex = nil
	s.NowPlayingItem.MediaStreams = nil // streams vanished: no fallback possible
	d.tick(5, s)
	wd := d.watchdog("s1")
	if wd.AudioStreamIndex == nil || *wd.AudioStreamIndex != 2 || *wd.AudioLanguage != "fre" {
		t.Fatalf("audio = %v %v", wd.AudioStreamIndex, wd.AudioLanguage)
	}
}

func TestPickVideo_PrefersEmbeddedDefaultLowestIndex(t *testing.T) {
	streams := []jellyfin.MediaStream{
		{Index: 0, Type: "Video", IsExternal: true},
		{Index: 3, Type: "Video"},
		{Index: 2, Type: "Video", IsDefault: true},
		{Index: 1, Type: "Video"},
	}
	if v := pickVideo(streams); v.Index != 2 {
		t.Fatalf("picked %d, want default 2", v.Index)
	}
	streams[2].IsDefault = false
	if v := pickVideo(streams); v.Index != 1 {
		t.Fatalf("picked %d, want lowest embedded 1", v.Index)
	}
}

func TestTracker_SubtitleOffAndMissingIndexAreNull(t *testing.T) {
	d := newDriver(t)
	s := captureMovie("s1", "m1", 0, false)
	s.PlayState.SubtitleStreamIndex = ptr(-1)
	s.PlayState.AudioStreamIndex = nil
	s.RemoteEndPoint = nil
	d.tick(0, s)
	wd := d.watchdog("s1")
	if wd.SubtitleStreamIndex != nil || wd.SubtitleLanguage != nil {
		t.Errorf("subtitle should be null: %v %v", wd.SubtitleStreamIndex, wd.SubtitleLanguage)
	}
	if wd.AudioStreamIndex == nil || *wd.AudioStreamIndex != 1 {
		t.Errorf("audio should fall back to first stream: %v", wd.AudioStreamIndex)
	}
	if wd.IsRemote != nil {
		t.Errorf("IsRemote should be null without endpoint")
	}
}

func TestTracker_RangeFallsBackToVideoRange(t *testing.T) {
	d := newDriver(t)
	s := captureMovie("s1", "m1", 0, false)
	s.NowPlayingItem.MediaStreams[0].VideoRangeType = ""
	s.NowPlayingItem.MediaStreams[0].VideoRange = "SDR"
	d.tick(0, s)
	if v := d.watchdog("s1").VideoRange; v == nil || *v != "SDR" {
		t.Fatalf("range = %v", v)
	}
}

func TestTracker_TranscodeInfoKeepsLastNonEmpty(t *testing.T) {
	d := newDriver(t)
	s := captureMovie("s1", "m1", 0, false)
	s.TranscodingInfo = &jellyfin.TranscodingInfo{
		TranscodeReasons:         jellyfin.FlexStringList{"VideoCodecNotSupported", "AudioCodecNotSupported"},
		Bitrate:                  ptr(int64(6_000_000)),
		HardwareAccelerationType: "vaapi",
	}
	d.tick(0, s)
	// Paused: Jellyfin drops TranscodingInfo; the captured values must stay.
	d.tick(5, captureMovie("s1", "m1", 0, true))
	d.tick(5) // stopped
	p := d.plays()
	if len(p) != 1 {
		t.Fatalf("plays = %d", len(p))
	}
	var reasons []string
	if err := json.Unmarshal(p[0].TranscodeReasons, &reasons); err != nil || len(reasons) != 2 || reasons[0] != "VideoCodecNotSupported" {
		t.Fatalf("reasons = %s (%v)", p[0].TranscodeReasons, err)
	}
	if p[0].Bitrate == nil || *p[0].Bitrate != 6_000_000 {
		t.Errorf("bitrate = %v", p[0].Bitrate)
	}
	if p[0].HwAccel == nil || *p[0].HwAccel != "vaapi" {
		t.Errorf("hwaccel = %v", p[0].HwAccel)
	}
}

func TestTracker_PauseCountCountsTransitions(t *testing.T) {
	d := newDriver(t)
	d.tick(0, movieAt("s1", "m1", 0, false))
	d.tick(5, movieAt("s1", "m1", 5000, false))
	d.tick(5, movieAt("s1", "m1", 5000, true)) // pause 1
	d.tick(5, movieAt("s1", "m1", 5000, true)) // still paused: no new transition
	d.tick(5, movieAt("s1", "m1", 10000, false))
	d.tick(5, movieAt("s1", "m1", 15000, false))
	d.tick(5, movieAt("s1", "m1", 15000, true)) // pause 2
	if got := d.watchdog("s1").PauseCount; got != 2 {
		t.Fatalf("watchdog PauseCount = %d, want 2", got)
	}
	d.tick(5)
	p := d.plays()
	if len(p) != 1 || p[0].PauseCount != 2 {
		t.Fatalf("plays = %+v", p)
	}
}

func TestTracker_PlayStartedPausedIsNotAPauseTransition(t *testing.T) {
	d := newDriver(t)
	d.tick(0, movieAt("s1", "m1", 0, true))
	d.tick(5, movieAt("s1", "m1", 0, true))
	if got := d.watchdog("s1").PauseCount; got != 0 {
		t.Fatalf("PauseCount = %d, want 0", got)
	}
}

func TestTracker_PauseCountResetsOnNewPlay(t *testing.T) {
	d := newDriver(t)
	d.tick(0, movieAt("s1", "m1", 0, false))
	d.tick(5, movieAt("s1", "m1", 5000, true))
	d.tick(5, movieAt("s1", "m2", 0, false)) // another item: new play
	if got := d.watchdog("s1").PauseCount; got != 0 {
		t.Fatalf("new play PauseCount = %d", got)
	}
	p := d.plays()
	if len(p) != 1 || p[0].PauseCount != 1 {
		t.Fatalf("closed play = %+v", p)
	}
}
