package sync

import (
	"context"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// helpers

// tickAt runs one SessionTick with the given live sessions, after pretending
// `secs` of wall time elapsed since the previous tick.
type driver struct {
	t    *testing.T
	svc  *Service
	db   *gorm.DB
	mock *mockJF
}

func newDriver(t *testing.T) *driver {
	mock := newMockJF(t)
	svc, db := newTestService(t, mock)
	return &driver{t: t, svc: svc, db: db, mock: mock}
}

func (d *driver) tick(elapsedSecs float64, sessions ...jellyfin.SessionInfo) {
	d.t.Helper()
	if elapsedSecs > 0 {
		var ids []string
		d.db.Raw(`SELECT "Id" FROM jf_activity_watchdog`).Scan(&ids)
		for _, id := range ids {
			rewindTick(d.t, d.db, id, elapsedSecs)
		}
	}
	d.mock.sessionScript = append(d.mock.sessionScript, sessions)
	d.svc.SessionTick(context.Background())
}

func (d *driver) plays() []models.JFPlaybackActivity {
	d.t.Helper()
	var out []models.JFPlaybackActivity
	if err := d.db.Order(`"StartedAt"`).Find(&out).Error; err != nil {
		d.t.Fatal(err)
	}
	return out
}

func (d *driver) watchdog(id string) models.JFActivityWatchdog {
	d.t.Helper()
	var wd models.JFActivityWatchdog
	if err := d.db.Where(`"Id" = ?`, id).First(&wd).Error; err != nil {
		d.t.Fatalf("watchdog %s: %v", id, err)
	}
	return wd
}

func movieAt(sess, item string, posMs int64, paused bool) jellyfin.SessionInfo {
	s := movieSession(sess, item, item, 0, paused)
	p := posMs * ticksPerMs
	s.PlayState.PositionTicks = &p
	rt := int64(7200) * ticksPerSecond
	s.NowPlayingItem.RunTimeTicks = &rt
	return s
}

func trackAt(sess, track string, posMs, runtimeSecs int64) jellyfin.SessionInfo {
	s := musicSession(sess, "album-1", track, track, "Album", 0)
	p := posMs * ticksPerMs
	s.PlayState.PositionTicks = &p
	rt := runtimeSecs * ticksPerSecond
	s.NowPlayingItem.RunTimeTicks = &rt
	return s
}

// ---------------------------------------------------------------------------
// precision

// The old tracker lost ~5% (per-step truncation to whole seconds) — the new
// one must match the real playback time to within a second over a long play,
// with jittery polls and 1 s position granularity like Jellyfin's extrapolation.
func TestTracker_LongPlayIsExact(t *testing.T) {
	d := newDriver(t)
	start := int64(600_000) // resumed at 10:00
	realMs := int64(0)
	d.tick(0, movieAt("s1", "m1", start, false))
	jitter := []float64{5.02, 4.97, 5.31, 5.004, 4.8, 5.6, 5.0}
	for i := 0; i < 120; i++ {
		el := jitter[i%len(jitter)]
		realMs += int64(el * 1000)
		pos := start + (realMs/1000)*1000 // server position advances in whole seconds
		d.tick(el, movieAt("s1", "m1", pos, false))
	}
	d.tick(5) // stopped
	p := d.plays()
	if len(p) != 1 {
		t.Fatalf("plays = %d, want 1", len(p))
	}
	got := *p[0].PlaybackDuration
	want := realMs / 1000
	if got < want-1 || got > want+1 {
		t.Fatalf("watched = %ds, real = %ds (must be within 1 s)", got, want)
	}
	if !p[0].IsView || *p[0].MediaType != "Movie" || *p[0].ItemId != "m1" {
		t.Errorf("row = view %v type %v item %v", p[0].IsView, *p[0].MediaType, *p[0].ItemId)
	}
	if *p[0].PositionTicksStart != start*ticksPerMs {
		t.Errorf("PositionTicksStart = %d", *p[0].PositionTicksStart)
	}
}

func TestTracker_PauseNotCountedButRecorded(t *testing.T) {
	d := newDriver(t)
	d.tick(0, movieAt("s1", "m1", 0, false))
	d.tick(5, movieAt("s1", "m1", 5000, false))
	for i := 0; i < 12; i++ { // 60 s paused, position frozen
		d.tick(5, movieAt("s1", "m1", 5000, true))
	}
	for i := 1; i <= 6; i++ {
		d.tick(5, movieAt("s1", "m1", 5000+int64(i)*5000, false))
	}
	d.tick(5)
	p := d.plays()[0]
	if *p.PlaybackDuration != 35 {
		t.Errorf("watched = %d, want 35", *p.PlaybackDuration)
	}
	if *p.PausedSeconds != 60 {
		t.Errorf("paused = %d, want 60", *p.PausedSeconds)
	}
}

func TestTracker_ForwardSeekCountsOnlyElapsed(t *testing.T) {
	d := newDriver(t)
	d.tick(0, movieAt("s1", "m1", 0, false))
	d.tick(5, movieAt("s1", "m1", 5000, false))
	d.tick(5, movieAt("s1", "m1", 1_800_000, false)) // jumped to 30:00
	d.tick(5, movieAt("s1", "m1", 1_805_000, false))
	d.tick(5)
	got := *d.plays()[0].PlaybackDuration
	if got > 22 { // 20 s real (+ at most the per-step slack)
		t.Errorf("watched = %d s after a 30 min forward seek, want ~20", got)
	}
}

func TestTracker_BackwardSeekIsSamePlay(t *testing.T) {
	d := newDriver(t)
	d.tick(0, movieAt("s1", "m1", 600_000, false))
	d.tick(5, movieAt("s1", "m1", 605_000, false))
	d.tick(5, movieAt("s1", "m1", 300_000, false)) // went back 5 min
	d.tick(5, movieAt("s1", "m1", 305_000, false))
	d.tick(5)
	if n := len(d.plays()); n != 1 {
		t.Fatalf("plays = %d, want 1 (a backward seek is not a new play)", n)
	}
}

// A client that reports every 10 s and then dies: Jellyfin keeps extrapolating
// the position for up to 10 min. That phantom time must not be counted.
func TestTracker_SilentClientNotCounted(t *testing.T) {
	d := newDriver(t)
	base := time.Date(2026, 11, 22, 20, 0, 0, 0, time.UTC)
	sess := func(posMs int64, checkIn, activity time.Time) jellyfin.SessionInfo {
		s := movieAt("s1", "m1", posMs, false)
		s.LastPlaybackCheckIn = jellyfin.FlexTime{Time: checkIn}
		s.LastActivityDate = jellyfin.FlexTime{Time: activity}
		return s
	}
	// Healthy: check-in every 10 s.
	pos := int64(0)
	for i := 0; i <= 12; i++ { // 60 s
		ts := base.Add(time.Duration(i*5) * time.Second)
		ci := base.Add(time.Duration(i/2*10) * time.Second)
		el := 5.0
		if i == 0 {
			el = 0
		}
		d.tick(el, sess(pos, ci, ts))
		pos += 5000
	}
	lastCheckIn := base.Add(60 * time.Second)
	// Client dies: position keeps moving (server extrapolation), no check-in.
	for i := 1; i <= 60; i++ { // 5 min
		ts := lastCheckIn.Add(time.Duration(i*5) * time.Second)
		d.tick(5, sess(pos, lastCheckIn, ts))
		pos += 5000
	}
	d.tick(5) // idle-stopped by Jellyfin
	got := *d.plays()[0].PlaybackDuration
	if got > 60+silentClientAfter.Milliseconds()/1000+5 {
		t.Errorf("watched = %d s, phantom extrapolated time was counted", got)
	}
}

// ---------------------------------------------------------------------------
// end of play

func TestTracker_TailRecoveredFromUserData(t *testing.T) {
	d := newDriver(t)
	d.tick(0, movieAt("s1", "m1", 0, false))
	d.tick(5, movieAt("s1", "m1", 5000, false))
	d.tick(5, movieAt("s1", "m1", 10000, false))
	// Stopped 4 s after the last poll: Jellyfin saved position 14 s.
	d.mock.userData["m1"] = jellyfin.UserItemData{PlaybackPositionTicks: 14_000 * ticksPerMs}
	d.tick(5)
	p := d.plays()[0]
	if *p.PlaybackDuration != 14 {
		t.Errorf("watched = %d, want 14 (tail from saved stop position)", *p.PlaybackDuration)
	}
	if *p.PositionTicksEnd != 14_000*ticksPerMs {
		t.Errorf("PositionTicksEnd = %d", *p.PositionTicksEnd)
	}
}

func TestTracker_TailIgnoredWhenInconsistent(t *testing.T) {
	d := newDriver(t)
	d.tick(0, movieAt("s1", "m1", 0, false))
	d.tick(5, movieAt("s1", "m1", 5000, false))
	d.tick(5, movieAt("s1", "m1", 40000, false))
	// Saved position from another device / older play: far away.
	d.mock.userData["m1"] = jellyfin.UserItemData{PlaybackPositionTicks: 3_000_000 * ticksPerMs}
	d.tick(5)
	if got := *d.plays()[0].PlaybackDuration; got > 12 {
		t.Errorf("watched = %d, inconsistent saved position must be ignored", got)
	}
}

func TestTracker_ShortClickRecordedNotView(t *testing.T) {
	d := newDriver(t)
	d.tick(0, movieAt("s1", "m1", 2000, false))
	d.tick(5, movieAt("s1", "m1", 7000, false))
	d.tick(5, movieAt("s1", "m1", 12000, false))
	d.tick(5)
	p := d.plays()
	if len(p) != 1 {
		t.Fatalf("a short play must still be recorded as a click, got %d rows", len(p))
	}
	if p[0].IsView {
		t.Errorf("%d s play flagged as a view", *p[0].PlaybackDuration)
	}
}

// ---------------------------------------------------------------------------
// item changes

func TestTracker_AutoplayNextEpisodeSplitsPrecisely(t *testing.T) {
	d := newDriver(t)
	d.tick(0, trackAt("s1", "t1", 170_000, 178))
	d.tick(5, trackAt("s1", "t1", 175_000, 178))
	// t1 ended at 178 s, t2 started; 5 s later t2 is at 2 s → 3 s tail + 2 s head.
	d.mock.userData["t1"] = jellyfin.UserItemData{Played: true}
	d.tick(5, trackAt("s1", "t2", 2000, 200))
	d.tick(5, trackAt("s1", "t2", 7000, 200))
	d.tick(5)
	var p []models.JFPlaybackActivity
	d.db.Order(`"ItemId"`).Find(&p)
	if len(p) != 2 {
		t.Fatalf("plays = %d, want 2", len(p))
	}
	if got := *p[0].PlaybackDuration; got != 8 { // 5 s seen + 3 s tail to the end
		t.Errorf("t1 watched = %d, want 8", got)
	}
	if got := *p[1].PlaybackDuration; got != 7 { // 2 s head + 5 s
		t.Errorf("t2 watched = %d, want 7", got)
	}
	if *p[1].NowPlayingItemId != "album-1" || *p[1].ItemId != "t2" || *p[1].MediaType != "Audio" {
		t.Errorf("music mapping: parent %v item %v type %v", *p[1].NowPlayingItemId, *p[1].ItemId, *p[1].MediaType)
	}
}

func TestTracker_RepeatOneCountsEachPlay(t *testing.T) {
	d := newDriver(t)
	d.tick(0, trackAt("s1", "t1", 0, 40))
	for pos := int64(5000); pos <= 35000; pos += 5000 {
		d.tick(5, trackAt("s1", "t1", pos, 40))
	}
	d.tick(5, trackAt("s1", "t1", 1000, 40)) // looped back to start
	for pos := int64(6000); pos <= 36000; pos += 5000 {
		d.tick(5, trackAt("s1", "t1", pos, 40))
	}
	d.tick(5)
	p := d.plays()
	if len(p) != 2 {
		t.Fatalf("plays = %d, want 2 (repeat-one)", len(p))
	}
	for i, r := range p {
		if !r.IsView {
			t.Errorf("play %d: %d s not a view", i, *r.PlaybackDuration)
		}
	}
}

// ---------------------------------------------------------------------------
// filtering & robustness

func TestTracker_OnlyMoviesEpisodesMusic(t *testing.T) {
	d := newDriver(t)
	tv := movieAt("s1", "c1", 0, false)
	tv.NowPlayingItem.Type = "TvChannel"
	theme := trackAt("s2", "theme", 0, 30)
	extra := "ThemeSong"
	theme.NowPlayingItem.ExtraType = &extra
	video := movieAt("s3", "v1", 0, false)
	video.NowPlayingItem.Type = "Video"
	d.tick(0, tv, theme, video)
	var n int64
	d.db.Model(&models.JFActivityWatchdog{}).Count(&n)
	if n != 0 {
		t.Fatalf("watchdog rows = %d, non-tracked media must be ignored", n)
	}
}

func TestTracker_JellyfinErrorClosesNothing(t *testing.T) {
	d := newDriver(t)
	d.tick(0, movieAt("s1", "m1", 0, false))
	d.tick(5, movieAt("s1", "m1", 5000, false))
	d.mock.server.Close() // Jellyfin down
	d.svc.SessionTick(context.Background())
	if len(d.plays()) != 0 {
		t.Fatal("a Jellyfin error must not close running plays")
	}
	if wd := d.watchdog("s1"); wd.WatchedMs != 5000 {
		t.Errorf("WatchedMs = %d", wd.WatchedMs)
	}
}

func TestTracker_FirstSeenMidPlayGetsNoHeadCredit(t *testing.T) {
	d := newDriver(t)
	d.tick(0, movieAt("s1", "m1", 3_600_000, false)) // resumed at 1:00:00
	if wd := d.watchdog("s1"); wd.WatchedMs != 0 {
		t.Errorf("WatchedMs = %d, resume point must not count as watched", wd.WatchedMs)
	}
}

// ---------------------------------------------------------------------------
// pure accounting

func TestAccumulate_TotalNeverExceedsWallTime(t *testing.T) {
	now := time.Now()
	started := now.Add(-10 * time.Second)
	last := now.Add(-5 * time.Second)
	pos := int64(5000) * ticksPerMs
	prev := models.JFActivityWatchdog{StartedAt: &started, LastTickAt: &last, PlaybackDuration: &pos, WatchedMs: 9000}
	o := observation{pos: ptr(int64(12000) * ticksPerMs)}
	next := accumulate(prev, o, now)
	if next.WatchedMs > 10000+stepSlackMs {
		t.Errorf("WatchedMs = %d exceeds wall time since start", next.WatchedMs)
	}
}

func ptr[T any](v T) *T { return &v }

// Real Jellyfin 10.11 (e2e): stopping a < 5 min item saves Played=true and
// position 0, so the stop position is unusable. The session stays listed with
// LastActivityDate = stop time, which gives the tail on the server clock.
func TestTracker_ShortItemTailFromServerActivity(t *testing.T) {
	d := newDriver(t)
	srv := time.Date(2026, 11, 22, 17, 0, 0, 0, time.UTC)
	at := func(s jellyfin.SessionInfo, ts time.Time) jellyfin.SessionInfo {
		s.LastActivityDate = jellyfin.FlexTime{Time: ts}
		s.LastPlaybackCheckIn = jellyfin.FlexTime{Time: srv}
		return s
	}
	d.tick(0, at(trackAt("s1", "t1", 0, 75), srv))
	d.tick(5, at(trackAt("s1", "t1", 5000, 75), srv.Add(5*time.Second)))
	// Stopped 4 s after that poll: Jellyfin reports Played=true / position 0.
	d.mock.userData["t1"] = jellyfin.UserItemData{Played: true}
	idle := jellyfin.SessionInfo{Id: "s1", UserId: "user-1", LastActivityDate: jellyfin.FlexTime{Time: srv.Add(9 * time.Second)}}
	d.tick(5, idle)
	if got := *d.plays()[0].PlaybackDuration; got != 9 {
		t.Errorf("watched = %d s, want 9 (5 s polled + 4 s tail from LastActivityDate)", got)
	}
}

// Real Jellyfin 10.11 (e2e): a play resumed mid-way lost the time between the
// client's Playing report and the first poll (30 s play recorded as 27 s).
func TestTracker_ResumedPlayHeadFromCheckIn(t *testing.T) {
	d := newDriver(t)
	srv := time.Date(2026, 11, 22, 17, 0, 0, 0, time.UTC)
	s := movieAt("s1", "m1", 1_503_500, false) // resumed at ~25:00, now at 25:03.5
	s.LastPlaybackCheckIn = jellyfin.FlexTime{Time: srv}
	s.LastActivityDate = jellyfin.FlexTime{Time: srv.Add(3500 * time.Millisecond)}
	d.tick(0, s)
	if wd := d.watchdog("s1"); wd.WatchedMs != 3500 {
		t.Errorf("head = %d ms, want 3500 (LastActivityDate − LastPlaybackCheckIn)", wd.WatchedMs)
	}
	paused := s
	paused.PlayState = &jellyfin.PlayState{PositionTicks: s.PlayState.PositionTicks, IsPaused: true}
	paused.Id = "s2"
	d.tick(0, paused)
	if wd := d.watchdog("s2"); wd.WatchedMs != 0 {
		t.Errorf("paused session got head credit %d ms", wd.WatchedMs)
	}
}
