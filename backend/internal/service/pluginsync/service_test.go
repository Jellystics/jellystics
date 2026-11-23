package pluginsync

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"gorm.io/gorm"
)

var ctx = context.Background()

func plays(t *testing.T, db *gorm.DB) []models.JFPlaybackActivity {
	t.Helper()
	var out []models.JFPlaybackActivity
	if err := db.Order(`"StartedAt", "Id"`).Find(&out).Error; err != nil {
		t.Fatal(err)
	}
	return out
}

func state(t *testing.T, s *Service) *Status {
	t.Helper()
	st, err := s.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func TestRun_PluginAbsentIsANoOp(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.absent = true })
	s, db := newTestService(t, f)

	res, err := s.Run(ctx)
	if err != nil {
		t.Fatalf("absent plugin must not be an error: %v", err)
	}
	if res.Detected {
		t.Fatal("detected = true")
	}
	st := state(t, s)
	if st.Detected || st.Healthy || st.LastSuccessAt != nil || st.LastError != nil || st.LastCheckedAt == nil {
		t.Fatalf("state = %+v", st)
	}
	if n := len(plays(t, db)); n != 0 {
		t.Fatalf("plays = %d", n)
	}
	if f.calls["/Jellystics/Plays"] != 0 {
		t.Fatal("tables must not be read without the plugin")
	}
}

func TestRun_NoJellyfinConfigured(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	db.Exec(`UPDATE app_config SET "JF_HOST" = NULL`)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if f.calls["/Jellystics/Info"] != 0 {
		t.Fatal("no request expected")
	}
}

func TestRun_IncompatibleSchemaStaysIdle(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.schemaVersion = 2 })
	f.putPlay(moviePlay("p1", t0, 60_000))
	s, db := newTestService(t, f)

	res, _ := s.Run(ctx)
	if !res.Detected || res.Compatible {
		t.Fatalf("res = %+v", res)
	}
	st := state(t, s)
	if !st.Detected || st.Healthy || st.LastError == nil || !strings.Contains(*st.LastError, "schema") || *st.SchemaVersion != 2 {
		t.Fatalf("state = %+v", st)
	}
	if len(plays(t, db)) != 0 || f.calls["/Jellystics/Plays"] != 0 {
		t.Fatal("incompatible plugin must not be synced")
	}
}

func TestRun_UnreachableRecordsError(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.failOn["/Jellystics/Info"] = 1 })
	s, _ := newTestService(t, f)
	if _, err := s.Run(ctx); err == nil {
		t.Fatal("want error")
	}
	st := state(t, s)
	if st.LastError == nil || st.Healthy {
		t.Fatalf("state = %+v", st)
	}
}

func TestRun_PaginatesAndResumesFromCursor(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.pageSize = 2 })
	for i := 0; i < 5; i++ {
		f.putPlay(moviePlay(fmt.Sprintf("p%d", i), t0.Add(time.Duration(i)*time.Hour), 60_000))
	}
	s, db := newTestService(t, f)

	res, err := s.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Plays != 5 || len(plays(t, db)) != 5 {
		t.Fatalf("plays = %d / %d", res.Plays, len(plays(t, db)))
	}
	if got := f.sinces["/Jellystics/Plays"]; fmt.Sprint(got) != "[0 2 4]" {
		t.Fatalf("since sequence = %v, want [0 2 4]", got)
	}
	st := state(t, s)
	if st.CursorPlays != 5 || !st.Healthy || st.LastError != nil || !st.Detected || *st.PluginVersion != "1.0.0.0" {
		t.Fatalf("state = %+v", st)
	}

	// A new play: the next run starts from the saved cursor.
	f.putPlay(moviePlay("p5", t0.Add(5*time.Hour), 60_000))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	got := f.sinces["/Jellystics/Plays"]
	if got[len(got)-1] != 5 || len(plays(t, db)) != 6 || state(t, s).CursorPlays != 6 {
		t.Fatalf("resume: sinces %v, plays %d", got, len(plays(t, db)))
	}
}

func TestRun_FailureMidwayKeepsCursorAndResumes(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) {
		f.pageSize = 2
		f.failOn["/Jellystics/Plays"] = 2 // second page fails
	})
	for i := 0; i < 4; i++ {
		f.putPlay(moviePlay(fmt.Sprintf("p%d", i), t0.Add(time.Duration(i)*time.Hour), 60_000))
	}
	s, db := newTestService(t, f)
	if _, err := s.Run(ctx); err == nil {
		t.Fatal("want error")
	}
	st := state(t, s)
	if st.CursorPlays != 2 || st.LastSuccessAt != nil || st.LastError == nil {
		t.Fatalf("state after failure = %+v", st)
	}
	if len(plays(t, db)) != 2 {
		t.Fatalf("plays = %d, want first page only", len(plays(t, db)))
	}
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if st := state(t, s); st.CursorPlays != 4 || st.LastError != nil || !st.Healthy || len(plays(t, db)) != 4 {
		t.Fatalf("after resume: state %+v plays %d", st, len(plays(t, db)))
	}
}

func TestRun_OpenPlayIsUpdatedWhenClosed(t *testing.T) {
	f := newFakePlugin(t)
	open := moviePlay("p1", t0, 40_000)
	open.EndedAt = nil
	open.State = "open"
	open.LastEventAt = jellyfin.FlexTime{Time: t0.Add(45 * time.Second)}
	f.putPlay(open)
	s, db := newTestService(t, f)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	p := plays(t, db)
	if len(p) != 1 || !p[0].EndedAt.Equal(t0.Add(45*time.Second)) || *p[0].PlaybackDuration != 40 || !p[0].IsView {
		t.Fatalf("open play row = %+v", p)
	}
	if playStateOf(p[0]) != "open" {
		t.Fatalf("PlayState = %s", p[0].PlayState)
	}

	closed := moviePlay("p1", t0, 600_000)
	closed.EndedAt = &jellyfin.FlexTime{Time: t0.Add(11 * time.Minute)}
	closed.PlayedToCompletion = new(bool)
	*closed.PlayedToCompletion = true
	f.putPlay(closed)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	p = plays(t, db)
	if len(p) != 1 || *p[0].PlaybackDuration != 600 || !p[0].EndedAt.Equal(t0.Add(11*time.Minute)) ||
		p[0].PlayedToCompletion == nil || !*p[0].PlayedToCompletion || playStateOf(p[0]) != "closed" {
		t.Fatalf("closed play row = %+v", p[0])
	}
}

func TestRun_AnnexTables(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) {
		f.library = []jellyfin.JellysticsLibraryEvent{{Id: "l1", Kind: "added", At: jellyfin.FlexTime{Time: t0},
			ItemId: sptr("0123456789ABCDEF0123456789ABCDEF"), Version: 1}}
		f.userData = []jellyfin.JellysticsUserDataEvent{{Id: "d1", Kind: "played", At: jellyfin.FlexTime{Time: t0},
			Reason: sptr("TogglePlayed"), Version: 2}}
		f.logins = []jellyfin.JellysticsLogin{{Id: "g1", At: jellyfin.FlexTime{Time: t0}, Success: true, UserName: sptr("bob"), Version: 3},
			{Id: "g2", At: jellyfin.FlexTime{Time: t0}, Success: false, Version: 4}}
		f.seq = 4
	})
	s, db := newTestService(t, f)
	res, err := s.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Library != 1 || res.UserData != 1 || res.Logins != 2 {
		t.Fatalf("res = %+v", res)
	}
	var lib models.JFLibraryEvent
	db.First(&lib, `"Id" = 'l1'`)
	if lib.Kind != "added" || *lib.ItemId != "0123456789abcdef0123456789abcdef" || !lib.At.Equal(t0) {
		t.Fatalf("library event = %+v", lib)
	}
	var n int64
	db.Model(&models.JFUserLogin{}).Where(`"Success"`).Count(&n)
	if n != 1 {
		t.Fatalf("successful logins = %d", n)
	}
	st := state(t, s)
	if st.CursorLibraryEvents != 1 || st.CursorUserDataEvents != 2 || st.CursorLogins != 4 {
		t.Fatalf("cursors = %+v", st)
	}
	// Replay is idempotent.
	db.Exec(`UPDATE plugin_sync_state SET cursor_logins = 0, cursor_library_events = 0, cursor_user_data_events = 0`)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	db.Model(&models.JFUserLogin{}).Count(&n)
	if n != 2 {
		t.Fatalf("logins after replay = %d", n)
	}
}

func insertTrackerRow(t *testing.T, db *gorm.DB, id, device string, start time.Time, secs int64) {
	t.Helper()
	end := start.Add(time.Duration(secs) * time.Second)
	a := models.JFPlaybackActivity{Id: id, UserId: sptr("user-1"), ItemId: sptr("movie-1"), NowPlayingItemId: sptr("movie-1"),
		MediaType: sptr("Movie"), DeviceId: sptr(device), StartedAt: &start, EndedAt: &end, PlaybackDuration: &secs,
		Source: "watchdog", IsView: secs >= 30}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
}

func TestRun_PluginPlaysReplaceOverlappingTrackerRows(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	insertTrackerRow(t, db, "tr-overlap", "dev-1", t0.Add(3*time.Second), 590)   // same play seen by the tracker
	insertTrackerRow(t, db, "tr-other-dev", "dev-2", t0.Add(3*time.Second), 590) // another device
	insertTrackerRow(t, db, "tr-later", "dev-1", t0.Add(2*time.Hour), 600)       // another play

	f.putPlay(moviePlay("p1", t0, 600_000))
	res, err := s.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Deduped != 1 {
		t.Fatalf("deduped = %d, want 1", res.Deduped)
	}
	var ids []string
	db.Raw(`SELECT "Id" FROM jf_playback_activity ORDER BY "Id"`).Scan(&ids)
	if fmt.Sprint(ids) != "[p1 tr-later tr-other-dev]" {
		t.Fatalf("rows = %v", ids)
	}
}

func TestRun_ImportSources(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)

	// Existing Jellystics row pushed to the plugin and coming back: kept as is.
	insertTrackerRow(t, db, "js-1", "dev-1", t0, 100)
	back := moviePlay("js-1", t0, 999_000)
	back.Source = "import-jellystics"
	f.putPlay(back)
	// Jellystics history lost locally: restored from the plugin.
	restored := moviePlay("js-2", t0.Add(-48*time.Hour), 120_000)
	restored.Source = "import-jellystics"
	f.putPlay(restored)

	// Legacy Jellystics import of PBR rowid 77 is replaced by the plugin's pbr-77.
	legacyStart := t0.Add(-72 * time.Hour)
	legacy := models.JFPlaybackActivity{Id: "77", UserId: sptr("user-1"), ItemId: sptr("movie-2"), MediaType: sptr("Movie"),
		StartedAt: &legacyStart, Source: "plugin", Imported: true}
	db.Create(&legacy)
	pbr := moviePlay("pbr-77", legacyStart, 300_000)
	pbr.ItemId = "movie-2"
	pbr.Source = "import-pbr"
	f.putPlay(pbr)
	// A PBR import covered by a tracker row is skipped (the tracker is more precise).
	covered := moviePlay("pbr-78", t0.Add(30*time.Second), 90_000)
	covered.Source = "import-pbr"
	f.putPlay(covered)

	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	byID := map[string]models.JFPlaybackActivity{}
	for _, p := range plays(t, db) {
		byID[p.Id] = p
	}
	if fmt.Sprint(len(byID)) != "3" {
		t.Fatalf("rows = %v", byID)
	}
	if r := byID["js-1"]; r.Source != "watchdog" || *r.PlaybackDuration != 100 {
		t.Fatalf("js-1 overwritten: %+v", r)
	}
	if r, ok := byID["js-2"]; !ok || r.Source != "import-jellystics" || !r.Imported {
		t.Fatalf("js-2 = %+v", r)
	}
	if r, ok := byID["pbr-77"]; !ok || r.Source != "import-pbr" || !r.Imported || *r.PlaybackDuration != 300 {
		t.Fatalf("pbr-77 = %+v", r)
	}
}

func TestMapPlay(t *testing.T) {
	ep := moviePlay("p1", t0, 29_400)
	ep.ItemType = sptr("Episode")
	ep.ItemId = "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"
	ep.SeriesId = sptr("series-1")
	ep.SeasonId = sptr("season-1")
	ep.SeriesName = sptr("Show")
	ep.PausedMs = 61_000
	ep.PauseCount = 2
	ep.SeekCount = 3
	ep.SkippedMs = 90_000
	ep.PlayMethod = sptr("Transcode")
	ep.VideoCodecFrom, ep.VideoCodecTo = sptr("hevc"), sptr("h264")
	ep.IsVideoDirect = new(bool) // explicit false
	ep.AudioCodecFrom, ep.AudioCodecTo = sptr("eac3"), sptr("eac3")
	ep.TranscodeReasons = jellyfin.EmbeddedJSON(`["VideoCodecNotSupported"]`)
	ep.HwAccel = sptr("None")
	ep.SubtitleStreamIndex = new(int)
	*ep.SubtitleStreamIndex = -1
	ep.IsRemote = new(bool)
	a, ok := MapPlay(ep)
	if !ok {
		t.Fatal("episode not mapped")
	}
	if *a.ItemId != "aaaaaaaabbbbccccddddeeeeeeeeeeee" || *a.EpisodeId != *a.ItemId || *a.NowPlayingItemId != "series-1" ||
		*a.SeasonId != "season-1" || *a.SeriesName != "Show" || *a.MediaType != "Episode" {
		t.Fatalf("ids = %+v", a)
	}
	if *a.PlaybackDuration != 29 || a.IsView || *a.PausedSeconds != 61 || a.PauseCount != 2 || *a.SeekCount != 3 ||
		*a.SkippedSeconds != 90 || a.Source != "jellystics-plugin" || a.Imported || a.HwAccel != nil || a.SubtitleStreamIndex != nil ||
		a.IsRemote == nil || *a.IsRemote {
		t.Fatalf("fields = %+v", a)
	}
	var ti map[string]any
	_ = json.Unmarshal(a.TranscodingInfo, &ti)
	if _, guessed := ti["IsAudioDirect"]; ti["VideoCodec"] != "h264" || ti["IsVideoDirect"] != false || guessed {
		t.Fatalf("TranscodingInfo = %s", a.TranscodingInfo)
	}
	if string(a.TranscodeReasons) != `["VideoCodecNotSupported"]` {
		t.Fatalf("reasons = %s", a.TranscodeReasons)
	}
	if *a.ActivityDateInserted != "2026-11-20 20:00:00.000+00:00" {
		t.Fatalf("ActivityDateInserted = %s", *a.ActivityDateInserted)
	}

	au := moviePlay("p2", t0, 30_000)
	au.ItemType, au.ItemId, au.AlbumId, au.AlbumName = sptr("Audio"), "track-1", sptr("album-1"), sptr("Album")
	au.Source = "import-pbr"
	a, _ = MapPlay(au)
	if *a.NowPlayingItemId != "album-1" || *a.EpisodeId != "track-1" || *a.SeriesName != "Album" || a.SeasonId != nil ||
		!a.IsView || a.Source != "import-pbr" || !a.Imported {
		t.Fatalf("audio = %+v", a)
	}

	mv, _ := MapPlay(moviePlay("p3", t0, 1000))
	if *mv.NowPlayingItemId != "movie-1" || mv.EpisodeId != nil || *mv.DeviceId != "dev-1" {
		t.Fatalf("movie = %+v", mv)
	}

	other := moviePlay("p4", t0, 1000)
	other.ItemType = sptr("TvChannel")
	if _, ok := MapPlay(other); ok {
		t.Fatal("untracked type must be skipped")
	}
}

func playStateOf(a models.JFPlaybackActivity) string {
	switch {
	case a.PluginOpen == nil:
		return ""
	case *a.PluginOpen:
		return "open"
	}
	return "closed"
}
