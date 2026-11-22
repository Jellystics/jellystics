package pluginsync

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
)

func bptr(b bool) *bool { return &b }
func iptr(i int) *int   { return &i }

// --- C1: plugin cursor reset -------------------------------------------------

func TestRun_PluginResetIsDetectedAndResynced(t *testing.T) {
	f := newFakePlugin(t)
	for i := 0; i < 3; i++ {
		f.putPlay(moviePlay(fmt.Sprintf("old%d", i), t0.Add(time.Duration(i)*time.Hour), 60_000))
	}
	s, db := newTestService(t, f)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}

	f.wipe() // reinstall: versions restart at 1
	f.putPlay(moviePlay("new0", t0.Add(10*time.Hour), 60_000))
	f.putPlay(moviePlay("new1", t0.Add(11*time.Hour), 60_000))
	res, err := s.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	st := state(t, s)
	if !res.CursorReset || res.Plays != 2 || st.CursorPlays != 2 || !st.Healthy || len(plays(t, db)) != 5 {
		t.Fatalf("res %+v, state %+v, rows %d", res, st, len(plays(t, db)))
	}
}

// A reinstalled plugin can be ahead of our cursors before the next sync (e.g.
// a Playback Reporting import right after the reinstall): only the database
// id tells the reset apart.
func TestRun_PluginResetAheadOfOurCursorIsDetectedByDatabaseId(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.databaseId = "db-a" })
	for i := 0; i < 3; i++ {
		f.putPlay(moviePlay(fmt.Sprintf("old%d", i), t0.Add(time.Duration(i)*time.Hour), 60_000))
	}
	s, db := newTestService(t, f)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if st := state(t, s); st.PluginDatabaseId == nil || *st.PluginDatabaseId != "db-a" {
		t.Fatalf("database id not stored: %+v", st)
	}

	f.wipe()
	f.set(func(f *fakePlugin) { f.databaseId = "db-b" })
	for i := 0; i < 5; i++ { // versions 1..5: the cursor (5) is past ours (3)
		f.putPlay(moviePlay(fmt.Sprintf("new%d", i), t0.Add(time.Duration(10+i)*time.Hour), 60_000))
	}
	res, err := s.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	st := state(t, s)
	if !res.CursorReset || res.Plays != 5 || st.CursorPlays != 5 || *st.PluginDatabaseId != "db-b" || len(plays(t, db)) != 8 {
		t.Fatalf("res %+v, state %+v, rows %d", res, st, len(plays(t, db)))
	}

	// Same database: no reset.
	if res, err = s.Run(ctx); err != nil || res.CursorReset {
		t.Fatalf("res %+v, err %v", res, err)
	}
}

func TestRun_CursorBackwardsIsBlindNotHealthy(t *testing.T) {
	f := newFakePlugin(t)
	f.putPlay(moviePlay("p0", t0, 60_000))
	f.putPlay(moviePlay("p1", t0.Add(time.Hour), 60_000))
	s, db := newTestService(t, f)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !state(t, s).Healthy {
		t.Fatal("healthy after first sync")
	}
	// Plugin data lost but /Info still reports a high cursor, and empty pages
	// answer next = 0: Jellystics cannot see new plays.
	f.wipe()
	high := int64(100)
	f.set(func(f *fakePlugin) { f.infoCursor = &high; f.emptyNextZero = true })
	if _, err := s.Run(ctx); err == nil {
		t.Fatal("want cursor error")
	}
	st := state(t, s)
	if st.Healthy || st.LastSuccessAt != nil || st.LastError == nil || !strings.Contains(*st.LastError, "backwards") {
		t.Fatalf("state = %+v: the tracker must not stay paused while blind", st)
	}
	// The tracker resumes writing: plugin is not healthy.
	repos := repository.New(db)
	got, _ := repos.PluginSync.GetState(ctx)
	if got.Healthy(time.Now()) {
		t.Fatal("tracker would still be paused")
	}
}

// --- I1: no backfill, sanitize only for imports ------------------------------

func TestRun_PluginPlaysDoNotTriggerBackfill(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	ts := "2026-11-01 20:00:00"
	db.Exec(`INSERT INTO jf_playback_activity ("Id", "UserId", "NowPlayingItemId", "ActivityDateInserted", "PlaybackDuration", source)
		VALUES ('legacy', 'user-1', 'movie-9', ?, 600, 'watchdog')`, ts)
	f.putPlay(moviePlay("p1", t0, 60_000))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	var legacy models.JFPlaybackActivity
	db.First(&legacy, `"Id" = 'legacy'`)
	if legacy.ItemId != nil || legacy.StartedAt != nil {
		t.Fatalf("plugin sync must not run BackfillTracking: %+v", legacy)
	}
}

func TestRun_ImportedDurationsAreSanitized(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	p := moviePlay("pbr-1", t0, 55*3600*1000) // overlong PBR row
	rt := int64(3600) * 10_000_000
	p.RunTimeTicks = &rt
	p.Source = "import-pbr"
	f.putPlay(p)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if r := plays(t, db); len(r) != 1 || *r[0].PlaybackDuration != 3600 {
		t.Fatalf("rows = %+v", r)
	}
}

// --- I3: direct flags and source video ---------------------------------------

func TestMapPlay_DirectFlagsNeverGuessed(t *testing.T) {
	p := moviePlay("p", t0, 60_000)
	p.PlayMethod = sptr("Transcode")
	p.VideoCodecFrom, p.VideoCodecTo = sptr("h264"), sptr("h264") // same-codec transcode (e.g. bitrate)
	p.AudioCodecFrom, p.AudioCodecTo = sptr("aac"), sptr("aac")
	p.IsVideoDirect, p.IsAudioDirect = bptr(false), bptr(true)
	a, _ := MapPlay(p)
	var ti map[string]any
	_ = json.Unmarshal(a.TranscodingInfo, &ti)
	if ti["IsVideoDirect"] != false || ti["IsAudioDirect"] != true {
		t.Fatalf("TranscodingInfo = %s", a.TranscodingInfo)
	}

	p.IsVideoDirect, p.IsAudioDirect = nil, nil
	a, _ = MapPlay(p)
	ti = nil
	_ = json.Unmarshal(a.TranscodingInfo, &ti)
	if _, ok := ti["IsVideoDirect"]; ok {
		t.Fatalf("absent flag must stay absent: %s", a.TranscodingInfo)
	}
	if _, ok := ti["IsAudioDirect"]; ok {
		t.Fatalf("absent flag must stay absent: %s", a.TranscodingInfo)
	}
}

func videoStream(t *testing.T, a models.JFPlaybackActivity) map[string]any {
	t.Helper()
	var streams []map[string]any
	_ = json.Unmarshal(a.MediaStreams, &streams)
	for _, s := range streams {
		if s["Type"] == "Video" {
			return s
		}
	}
	t.Fatalf("no video stream in %s", a.MediaStreams)
	return nil
}

func TestMapPlay_SourceVideo(t *testing.T) {
	p := moviePlay("p", t0, 60_000)
	p.VideoCodecFrom = sptr("hevc")
	p.SourceWidth, p.SourceHeight = iptr(3840), iptr(2160)
	p.VideoRange, p.VideoRangeType = sptr("HDR"), sptr("DOVI")
	fb := &repository.SourceVideo{Width: iptr(1920), VideoRangeType: sptr("SDR")}
	a, _ := MapPlayWith(p, fb)
	v := videoStream(t, a)
	if v["Width"] != float64(3840) || v["Height"] != float64(2160) || v["VideoRange"] != "HDR" || v["VideoRangeType"] != "DOVI" ||
		*a.VideoRange != "DOVI" {
		t.Fatalf("plugin values must win: %v, column %v", v, *a.VideoRange)
	}

	p.SourceWidth, p.SourceHeight, p.VideoRange, p.VideoRangeType = nil, nil, nil, nil
	a, _ = MapPlayWith(p, &repository.SourceVideo{Width: iptr(1920), Height: iptr(1080), VideoRange: sptr("HDR"), VideoRangeType: sptr("HDR10")})
	v = videoStream(t, a)
	if v["Width"] != float64(1920) || v["Height"] != float64(1080) || v["VideoRangeType"] != "HDR10" || *a.VideoRange != "HDR10" {
		t.Fatalf("catalog fallback: %v", v)
	}
}

func TestRun_SourceVideoFallsBackToCatalog(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	db.Exec(`INSERT INTO jf_item_info ("Id", "MediaStreams") VALUES ('movie-1', ?::jsonb)`,
		`[{"Type":"Audio","Index":1},{"Type":"Video","Index":0,"Width":3840,"Height":2160,"VideoRange":"HDR","VideoRangeType":"HDR10"}]`)
	f.putPlay(moviePlay("p1", t0, 60_000))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	r := plays(t, db)
	if len(r) != 1 || r[0].VideoRange == nil || *r[0].VideoRange != "HDR10" || videoStream(t, r[0])["Width"] != float64(3840) {
		t.Fatalf("row = %+v", r)
	}
}

func TestMapPlay_EmptyUserIdIsNull(t *testing.T) {
	p := moviePlay("p", t0, 1000)
	p.UserId, p.UserName = "", nil
	a, _ := MapPlay(p)
	if a.UserId != nil || a.UserName != nil {
		t.Fatalf("user = %v / %v", a.UserId, a.UserName)
	}
}

// --- M2 / M3: tracker reconciliation ------------------------------------------

func TestFilterTrackerPlays_KeepsLongerTrackerPlay(t *testing.T) {
	f := newFakePlugin(t)
	s, _ := newTestService(t, f)
	f.putPlay(moviePlay("p1", t0, 60_000)) // closed at t0+60s
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	mk := func(id string, secs int64) models.JFPlaybackActivity {
		start, end := t0.Add(2*time.Second), t0.Add(time.Duration(secs)*time.Second)
		return models.JFPlaybackActivity{Id: id, UserId: sptr("user-1"), ItemId: sptr("movie-1"), DeviceId: sptr("dev-1"),
			StartedAt: &start, EndedAt: &end, Source: "watchdog"}
	}
	kept, err := s.repos.PluginSync.FilterTrackerPlays(ctx, []models.JFPlaybackActivity{mk("covered", 65), mk("longer", 600)})
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 1 || kept[0].Id != "longer" {
		t.Fatalf("kept = %v", kept)
	}
}

func TestRun_RecentTrackerRowsDedupedEveryRun(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	start := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	f.putPlay(moviePlay("p1", start, 600_000))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	// A tracker write racing the sync lands after the plugin play.
	insertTrackerRow(t, db, "late-tracker", "dev-1", start.Add(3*time.Second), 590)
	res, err := s.Run(ctx) // no new plays
	if err != nil {
		t.Fatal(err)
	}
	if res.Deduped != 1 || len(plays(t, db)) != 1 {
		t.Fatalf("deduped %d, rows %d", res.Deduped, len(plays(t, db)))
	}
}

// --- M4: legacy PBR rows replaced by overlap ------------------------------------

func TestRun_ImportPBRReplacesLegacyByOverlap(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	legacyStart := t0.Add(-72*time.Hour + 20*time.Second)
	legacyEnd := legacyStart.Add(5 * time.Minute)
	db.Create(&models.JFPlaybackActivity{Id: "9001", UserId: sptr("user-1"), ItemId: sptr("movie-1"), MediaType: sptr("Movie"),
		StartedAt: &legacyStart, EndedAt: &legacyEnd, Source: "plugin", Imported: true})
	p := moviePlay("pbr-42", t0.Add(-72*time.Hour), 300_000) // different rowid, same play
	p.Source = "import-pbr"
	f.putPlay(p)
	res, err := s.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	r := plays(t, db)
	if res.Imported != 1 || len(r) != 1 || r[0].Id != "pbr-42" {
		t.Fatalf("imported %d, rows %+v", res.Imported, r)
	}
}

// --- M5: PluginOpen and stale plays -------------------------------------------

func TestRun_StaleOpenPluginPlaysAreClosed(t *testing.T) {
	f := newFakePlugin(t)
	open := moviePlay("p1", t0, 40_000)
	open.EndedAt, open.State = nil, "open"
	f.putPlay(open)
	s, db := newTestService(t, f)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if r := plays(t, db); r[0].PluginOpen == nil || !*r[0].PluginOpen {
		t.Fatalf("PluginOpen = %v", r[0].PluginOpen)
	}
	// Plugin gone for 2 h.
	f.set(func(f *fakePlugin) { f.absent = true })
	db.Exec(`UPDATE plugin_sync_state SET last_success_at = now() - interval '2 hours'`)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if r := plays(t, db); r[0].PluginOpen == nil || *r[0].PluginOpen {
		t.Fatalf("stale open play not closed: %v", r[0].PluginOpen)
	}
}

// --- M8: annex failures do not affect health ------------------------------------

func TestRun_AnnexFailureKeepsPluginHealthy(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.failOn["/Jellystics/LibraryEvents"] = 1 })
	f.putPlay(moviePlay("p1", t0, 60_000))
	s, _ := newTestService(t, f)
	if _, err := s.Run(ctx); err == nil {
		t.Fatal("want the annex error reported")
	}
	st := state(t, s)
	if !st.Healthy || st.CursorPlays != 1 || st.LastError == nil || !strings.Contains(*st.LastError, "LibraryEvents") {
		t.Fatalf("state = %+v", st)
	}
}

// --- M7: stats caches reset -----------------------------------------------------

func TestRun_NotifiesDataChanges(t *testing.T) {
	f := newFakePlugin(t)
	s, _ := newTestService(t, f)
	calls := 0
	s.OnDataChanged = func() { calls++ }
	if _, err := s.Run(ctx); err != nil || calls != 0 {
		t.Fatalf("no data: calls %d err %v", calls, err)
	}
	f.putPlay(moviePlay("p1", t0, 60_000))
	if _, err := s.Run(ctx); err != nil || calls != 1 {
		t.Fatalf("calls = %d err %v", calls, err)
	}
}
