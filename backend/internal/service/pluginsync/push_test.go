package pluginsync

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func histRow(id, source string, start time.Time, durSec int64) models.JFPlaybackActivity {
	item, user, mt := "movie-"+id, "user-1", "Movie"
	end := start.Add(time.Duration(durSec) * time.Second)
	return models.JFPlaybackActivity{
		Id: id, UserId: &user, UserName: sptr("alice"), ItemId: &item, NowPlayingItemId: &item,
		NowPlayingItemName: sptr("Film " + id), MediaType: &mt, PlaybackDuration: &durSec,
		StartedAt: &start, EndedAt: &end, IsView: durSec >= 30, Source: source,
		Imported: source != "watchdog" && source != "jellystics-plugin",
	}
}

// seedHistory: 1203 tracker rows + 2 legacy Playback Reporting rows to push,
// and rows that must not be pushed (plugin plays, plugin imports, untracked
// type, undated row).
func seedHistory(t *testing.T, db *gorm.DB) {
	t.Helper()
	var rows []models.JFPlaybackActivity
	for i := 0; i < 1203; i++ {
		rows = append(rows, histRow(fmt.Sprintf("t%05d", i), "watchdog", t0.Add(time.Duration(i)*time.Hour), 600))
	}
	rows = append(rows, histRow("legacy-1", "plugin", t0, 100), histRow("legacy-2", "plugin", t0, 100),
		histRow("p1", "jellystics-plugin", t0, 100), histRow("pbr-9", "import-pbr", t0, 100),
		histRow("ij-1", "import-jellystics", t0, 100))
	other := histRow("other-type", "watchdog", t0, 100)
	other.MediaType = sptr("Other")
	undated := histRow("undated", "watchdog", t0, 100)
	undated.StartedAt = nil
	rows = append(rows, other, undated)
	if err := db.CreateInBatches(rows, 500).Error; err != nil {
		t.Fatal(err)
	}
}

func pushStatus(t *testing.T, s *Service) PushStatus {
	t.Helper()
	st, err := s.PushStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestPush_SendsHistoryInBatchesAndResumes(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	seedHistory(t, db)
	f.set(func(f *fakePlugin) { f.failOn["/Jellystics/Import/Plays"] = 2 })

	st, err := s.StartPush(ctx)
	if err != nil || st.State != PushRunning || st.Total != 1205 {
		t.Fatalf("start = %+v, %v", st, err)
	}
	s.WaitPush()
	st = pushStatus(t, s)
	if st.State != PushError || st.Sent != 500 || st.Imported != 500 || st.Error == nil || !st.Resumable || st.FinishedAt == nil {
		t.Fatalf("after failure = %+v", st)
	}

	// Resume: continues after the acknowledged batch.
	if _, err := s.StartPush(ctx); err != nil {
		t.Fatal(err)
	}
	s.WaitPush()
	st = pushStatus(t, s)
	if st.State != PushDone || st.Total != 1205 || st.Sent != 1205 || st.Imported != 1205 || st.Skipped != 0 || st.Rejected != 0 || st.Resumable {
		t.Fatalf("after resume = %+v", st)
	}
	var sizes []int
	seen := map[string]int{}
	for _, b := range f.pushBatches {
		sizes = append(sizes, len(b))
		for _, p := range b {
			seen[p.Id]++
			if p.Source != "import-jellystics" || p.State != "closed" {
				t.Fatalf("play %s: source %q state %q", p.Id, p.Source, p.State)
			}
		}
	}
	// Accepted batches: 500, then (the failed second batch resent) 500, 205.
	if fmt.Sprint(sizes) != "[500 500 205]" || len(seen) != 1205 {
		t.Fatalf("batches = %v, distinct ids = %d", sizes, len(seen))
	}
	for _, id := range []string{"p1", "pbr-9", "ij-1", "other-type", "undated"} {
		if seen[id] != 0 {
			t.Errorf("%s must not be pushed", id)
		}
	}
	if seen["legacy-1"] != 1 || seen["t00000"] != 1 || seen["t01202"] != 1 {
		t.Errorf("legacy-1 %d, t00000 %d, t01202 %d", seen["legacy-1"], seen["t00000"], seen["t01202"])
	}

	// A new push after a finished one restarts from the beginning; the
	// plugin skips what it already has.
	if _, err := s.StartPush(ctx); err != nil {
		t.Fatal(err)
	}
	s.WaitPush()
	st = pushStatus(t, s)
	if st.State != PushDone || st.Sent != 1205 || st.Imported != 0 || st.Skipped != 1205 {
		t.Fatalf("second push = %+v", st)
	}
}

func TestPush_RefusedWithoutPlugin(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.absent = true })
	s, db := newTestService(t, f)
	if _, err := s.StartPush(ctx); !errors.Is(err, ErrPluginAbsent) {
		t.Fatalf("absent: %v", err)
	}
	f.set(func(f *fakePlugin) { f.absent, f.schemaVersion = false, 2 })
	if _, err := s.StartPush(ctx); !errors.Is(err, ErrPluginIncompatible) {
		t.Fatalf("incompatible: %v", err)
	}
	db.Exec(`UPDATE app_config SET "JF_HOST" = NULL`)
	if _, err := s.StartPush(ctx); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("not configured: %v", err)
	}
	if st := pushStatus(t, s); st.State != PushIdle || st.Sent != 0 {
		t.Fatalf("status = %+v", st)
	}
}

// A push left "running" by a restart is reported as interrupted and resumes.
func TestPush_InterruptedResumes(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	seedHistory(t, db)
	db.Exec(`UPDATE plugin_sync_state SET push_state = 'running', push_cursor = 't01000', push_cursor_at = ?, push_sent = 1003, push_imported = 1003, push_total = 1205`,
		t0.Add(1000*time.Hour))
	if st := pushStatus(t, s); st.State != PushInterrupted || !st.Resumable {
		t.Fatalf("status = %+v", st)
	}
	if _, err := s.StartPush(ctx); err != nil {
		t.Fatal(err)
	}
	s.WaitPush()
	st := pushStatus(t, s)
	// Ids after t01000: t01001..t01202 (202 rows).
	if st.State != PushDone || st.Sent != 1205 || st.Imported != 1205 || len(f.pushBatches) != 1 || len(f.pushBatches[0]) != 202 {
		t.Fatalf("status = %+v, batches = %d", st, len(f.pushBatches))
	}
}

// The sync never overwrites the push progress and vice versa.
func TestPush_StateIsolatedFromSync(t *testing.T) {
	f := newFakePlugin(t)
	f.putPlay(moviePlay("p1", t0, 60_000))
	s, db := newTestService(t, f)
	db.Exec(`UPDATE plugin_sync_state SET push_state = 'done', push_sent = 7`)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if st := pushStatus(t, s); st.State != PushDone || st.Sent != 7 {
		t.Fatalf("push state overwritten by the sync: %+v", st)
	}
	if st := state(t, s); st.CursorPlays != 1 || st.PlaysSynced != 1 || st.SupportedSchemaVersion != 1 {
		t.Fatalf("sync state = %+v", st)
	}
}

func TestResync_ResetsCursorsAndRuns(t *testing.T) {
	f := newFakePlugin(t)
	f.putPlay(moviePlay("p1", t0, 60_000))
	f.putPlay(moviePlay("p2", t0.Add(time.Hour), 60_000))
	s, _ := newTestService(t, f)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.Resync(ctx); err != nil {
		t.Fatal(err)
	}
	s.WaitResync()
	sinces := f.sinces["/Jellystics/Plays"]
	if len(sinces) != 2 || sinces[0] != 0 || sinces[1] != 0 {
		t.Fatalf("plays read since %v, want a second read from 0", sinces)
	}
	if st := state(t, s); st.CursorPlays != 2 || st.LastError != nil || !st.Healthy {
		t.Fatalf("state = %+v", st)
	}
}

func TestToPlay(t *testing.T) {
	start := time.Date(2026, 3, 1, 20, 0, 0, 0, time.FixedZone("CET", 3600))
	end := start.Add(50 * time.Minute)
	dur, paused, rt, pos0, pos1 := int64(2700), int64(120), int64(3000*secTicks), int64(0), int64(2800*secTicks)
	seeks, skipped, audio, sub := 2, 30, 1, 3
	a := models.JFPlaybackActivity{
		Id: "act-1", UserId: sptr("u1"), UserName: sptr("alice"), ItemId: sptr("ep1"), NowPlayingItemId: sptr("series1"),
		EpisodeId: sptr("ep1"), SeasonId: sptr("season1"), SeriesName: sptr("Show"), NowPlayingItemName: sptr("Pilot"),
		MediaType: sptr("Episode"), PlaybackDuration: &dur, PausedSeconds: &paused, PauseCount: 3, SeekCount: &seeks,
		SkippedSeconds: &skipped, RunTimeTicks: &rt, PositionTicksStart: &pos0, PositionTicksEnd: &pos1,
		StartedAt: &start, EndedAt: &end, DeviceId: sptr("dev"), DeviceName: sptr("TV"), Client: sptr("Web"),
		PlayMethod: sptr("Transcode"), AudioStreamIndex: &audio, AudioLanguage: sptr("fre"), SubtitleStreamIndex: &sub,
		VideoRange: sptr("HDR10"), TranscodeReasons: datatypes.JSON(`["AudioCodecNotSupported"]`),
		TranscodingInfo: datatypes.JSON(`{"VideoCodec":"h264","AudioCodec":"aac","IsVideoDirect":false,"IsAudioDirect":false,"Width":1920,"Height":1080,"Framerate":23.976,"Bitrate":8000000,"HardwareAccelerationType":"vaapi"}`),
		MediaStreams: datatypes.JSON(`[{"Type":"Video","Codec":"hevc","Width":3840,"Height":2160,"VideoRange":"HDR","VideoRangeType":"HDR10","IsDefault":true,"Index":0},
			{"Type":"Audio","Codec":"eac3","Index":2,"IsDefault":true},{"Type":"Audio","Codec":"truehd","Index":1}]`),
		PlayState: datatypes.JSON(`{"StreamChanges":[{"at":"2026-03-01T19:05:00Z","kind":"audio","index":1}]}`),
	}
	p := ToPlay(a)
	ms := func(v int64) int64 { return v * 1000 }
	if p.Id != "act-1" || p.Source != "import-jellystics" || p.State != "closed" || p.UserId != "u1" || p.ItemId != "ep1" ||
		*p.ItemType != "Episode" || *p.ItemName != "Pilot" || *p.SeriesId != "series1" || *p.SeriesName != "Show" ||
		*p.SeasonId != "season1" || p.AlbumId != nil {
		t.Fatalf("identity = %+v", p)
	}
	if p.WatchedMs != ms(2700) || p.PausedMs != ms(120) || p.PauseCount != 3 || p.SeekCount != 2 || p.SkippedMs != ms(30) ||
		!p.StartedAt.Equal(start) || !p.EndedAt.Equal(end) || !p.LastEventAt.Equal(end) || *p.PositionEndTicks != pos1 || *p.RunTimeTicks != rt {
		t.Fatalf("times = %+v", p)
	}
	if *p.VideoCodecFrom != "hevc" || *p.VideoCodecTo != "h264" || *p.AudioCodecFrom != "truehd" || *p.AudioCodec != "truehd" ||
		*p.AudioCodecTo != "aac" || *p.Width != 1920 || *p.Height != 1080 || *p.SourceWidth != 3840 || *p.SourceHeight != 2160 ||
		*p.VideoRangeType != "HDR10" || *p.VideoRange != "HDR10" || *p.IsVideoDirect || *p.IsAudioDirect ||
		*p.Bitrate != 8000000 || *p.HwAccel != "vaapi" || *p.Framerate != 23.976 ||
		string(p.TranscodeReasons) != `["AudioCodecNotSupported"]` || string(p.StreamChanges) == "" {
		t.Fatalf("streams = %+v", p)
	}
	// Round trip through the plugin mapping keeps the play identical.
	back, ok := MapPlay(p)
	if !ok || back.Id != "act-1" || *back.PlaybackDuration != 2700 || *back.NowPlayingItemId != "series1" || *back.EpisodeId != "ep1" ||
		back.Source != "import-jellystics" || !back.StartedAt.Equal(start) || !back.EndedAt.Equal(end) {
		t.Fatalf("round trip = %+v", back)
	}

	// Audio: the album is the NowPlayingItemId; no ended time → start + duration.
	track := models.JFPlaybackActivity{Id: "a2", UserId: sptr("u1"), ItemId: sptr("t1"), NowPlayingItemId: sptr("al1"),
		SeriesName: sptr("Album"), MediaType: sptr("Audio"), PlaybackDuration: &dur, StartedAt: &start}
	p = ToPlay(track)
	if *p.AlbumId != "al1" || *p.AlbumName != "Album" || p.SeriesId != nil || !p.EndedAt.Equal(start.Add(45*time.Minute)) {
		t.Fatalf("track = %+v", p)
	}
}

const secTicks = 10_000_000
