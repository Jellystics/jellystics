package pluginsync

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"gorm.io/gorm"
)

func legacyRow(t *testing.T, db *gorm.DB, id string, start time.Time, secs int64, repaired bool) {
	t.Helper()
	end := start.Add(time.Duration(secs) * time.Second)
	a := models.JFPlaybackActivity{Id: id, UserId: sptr("user-1"), ItemId: sptr("movie-1"), NowPlayingItemId: sptr("movie-1"),
		MediaType: sptr("Movie"), StartedAt: &start, EndedAt: &end, PlaybackDuration: &secs, IsView: secs >= 30,
		Source: "plugin", Imported: true, DurationRepaired: repaired}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
}

func pbrPlay(id string, start time.Time, watchedMs int64) jellyfin.JellysticsPlay {
	p := moviePlay(id, start, watchedMs)
	p.Source, p.DeviceId = "import-pbr", nil
	return p
}

func ids(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var out []string
	for _, p := range plays(t, db) {
		out = append(out, p.Id)
	}
	return fmt.Sprint(out)
}

func push(t *testing.T, s *Service) {
	t.Helper()
	if _, err := s.StartPush(ctx); err != nil {
		t.Fatal(err)
	}
	s.WaitPush()
}

func resync(t *testing.T, s *Service) {
	t.Helper()
	if err := s.Resync(ctx); err != nil {
		t.Fatal(err)
	}
	s.WaitResync()
}

// C1 (a): a legacy row pushed to the plugin comes back as import-jellystics
// in the same page as the plugin's own PBR import of it: one row only.
func TestC1_PushedLegacyRowAndItsPBRInOnePage(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.echoImports = true })
	s, db := newTestService(t, f)
	start := t0.Add(-72 * time.Hour)
	legacyRow(t, db, "10", start, 300, false)
	push(t, s)
	f.putPlay(pbrPlay("pbr-10", start, 300_000))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db); got != "[pbr-10]" {
		t.Fatalf("rows = %s, want [pbr-10]", got)
	}
	resync(t, s)
	if got := ids(t, db); got != "[pbr-10]" {
		t.Fatalf("after resync rows = %s, want [pbr-10]", got)
	}
}

// C1 (b): a tracker row pushed to the plugin, then replaced by a plugin play,
// never comes back.
func TestC1_DedupedTrackerRowDoesNotResurrect(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.echoImports = true })
	s, db := newTestService(t, f)
	insertTrackerRow(t, db, "w1", "dev-1", t0, 600)
	push(t, s)
	f.putPlay(moviePlay("p1", t0.Add(3*time.Second), 600_000))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db); got != "[p1]" {
		t.Fatalf("rows = %s, want [p1]", got)
	}
	resync(t, s)
	if got := ids(t, db); got != "[p1]" {
		t.Fatalf("after resync rows = %s, want [p1]", got)
	}
}

// C1 (c): same across separate pages, in both orders.
func TestC1_AcrossPages(t *testing.T) {
	for _, pbrFirst := range []bool{false, true} {
		t.Run(fmt.Sprint("pbrFirst=", pbrFirst), func(t *testing.T) {
			f := newFakePlugin(t)
			f.set(func(f *fakePlugin) { f.echoImports, f.pageSize = true, 1 })
			s, db := newTestService(t, f)
			start := t0.Add(-72 * time.Hour)
			legacyRow(t, db, "10", start, 300, false)
			if pbrFirst {
				f.putPlay(pbrPlay("pbr-10", start, 300_000))
				push(t, s)
			} else {
				push(t, s)
				f.putPlay(pbrPlay("pbr-10", start, 300_000))
			}
			if _, err := s.Run(ctx); err != nil {
				t.Fatal(err)
			}
			if got := ids(t, db); got != "[pbr-10]" {
				t.Fatalf("rows = %s", got)
			}
			resync(t, s)
			if got := ids(t, db); got != "[pbr-10]" {
				t.Fatalf("after resync rows = %s", got)
			}
		})
	}
}

// C1: Jellystics history lost locally still fills the gap.
func TestC1_ImportJellysticsFillsGenuineGaps(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	lost := moviePlay("js-lost", t0.Add(-48*time.Hour), 120_000)
	lost.Source = "import-jellystics"
	f.putPlay(lost)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db); got != "[js-lost]" {
		t.Fatalf("rows = %s", got)
	}
}

// I1: the repaired flag survives the PBR supersede (and later resyncs); the
// plugin's durationRepaired is stored.
func TestI1_RepairedFlagCarriedOverAndFromPlugin(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	start := t0.Add(-72 * time.Hour)
	legacyRow(t, db, "20", start, 300, true)
	f.putPlay(pbrPlay("pbr-20", start, 300_000))
	yes := true
	fromPlugin := pbrPlay("pbr-21", start.Add(24*time.Hour), 100_000)
	fromPlugin.DurationRepaired = &yes
	f.putPlay(fromPlugin)
	f.putPlay(pbrPlay("pbr-22", start.Add(48*time.Hour), 100_000))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	check := func(when string) {
		got := map[string]bool{}
		for _, p := range plays(t, db) {
			got[p.Id] = p.DurationRepaired
		}
		if fmt.Sprint(got) != "map[pbr-20:true pbr-21:true pbr-22:false]" {
			t.Fatalf("%s: repaired = %v", when, got)
		}
	}
	check("first sync")
	resync(t, s)
	check("after resync")
}

// I2: a TSV-imported Playback Reporting row and the plugin's import of the
// same play are one row.
func TestI2_TSVImportSupersededByPBR(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	start := t0.Add(-72 * time.Hour)
	end := start.Add(300 * time.Second)
	secs := int64(300)
	db.Create(&models.JFPlaybackActivity{Id: "import-user-1-movie-1-2026-11-17 20:00:20", UserId: sptr("user-1"), ItemId: sptr("movie-1"),
		MediaType: sptr("Movie"), StartedAt: &start, EndedAt: &end, PlaybackDuration: &secs, Source: "import", Imported: true, DurationRepaired: true})
	f.putPlay(pbrPlay("pbr-30", start.Add(20*time.Second), 300_000))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	r := plays(t, db)
	if len(r) != 1 || r[0].Id != "pbr-30" || !r[0].DurationRepaired {
		t.Fatalf("rows = %+v", r)
	}
}

// Round trip: pushed tracker rows (ids in any GUID format) come back without
// duplicating anything.
func TestPush_RoundTripKeepsOneRowPerPlay(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.echoImports = true })
	s, db := newTestService(t, f)
	seedHistory(t, db)
	before := len(plays(t, db))
	push(t, s)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	resync(t, s)
	if after := len(plays(t, db)); after != before {
		t.Fatalf("rows %d -> %d", before, after)
	}
}

// Resync requested while a sync is running: the running sync cannot save its
// old cursor over the reset, everything is read again from 0.
func TestResync_DuringRunningTick(t *testing.T) {
	f := newFakePlugin(t)
	f.putPlay(moviePlay("p1", t0, 60_000))
	f.putPlay(moviePlay("p2", t0.Add(time.Hour), 60_000))
	s, _ := newTestService(t, f)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	f.set(func(f *fakePlugin) { f.playsDelay = 150 * time.Millisecond; f.sinces = map[string][]int64{} })
	done := make(chan struct{})
	go func() { s.Tick(ctx); close(done) }()
	time.Sleep(50 * time.Millisecond) // the tick is waiting for its Plays page
	resync(t, s)
	<-done
	sinces := f.sinces["/Jellystics/Plays"]
	if len(sinces) != 2 || sinces[0] != 2 || sinces[1] != 0 {
		t.Fatalf("plays read since %v, want [2 0]", sinces)
	}
	if st := state(t, s); st.CursorPlays != 2 {
		t.Fatalf("cursor = %d", st.CursorPlays)
	}
}

// M1: plays are pushed in (StartedAt, Id) order and resume from that cursor.
func TestPush_OrderedByStartedAt(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.failOn["/Jellystics/Import/Plays"] = 2 })
	s, db := newTestService(t, f)
	var rows []models.JFPlaybackActivity
	for i := 0; i < 600; i++ { // ids in reverse start order
		rows = append(rows, histRow(fmt.Sprintf("r%04d", 999-i), "watchdog", t0.Add(time.Duration(i)*time.Minute), 60))
	}
	if err := db.CreateInBatches(rows, 500).Error; err != nil {
		t.Fatal(err)
	}
	push(t, s) // second batch fails
	push(t, s) // resumes
	var got []string
	for _, b := range f.pushBatches {
		for _, p := range b {
			got = append(got, p.Id)
		}
	}
	if len(got) != 600 || got[0] != "r0999" || got[499] != "r0500" || got[500] != "r0499" || got[599] != "r0400" {
		t.Fatalf("order: %d plays, first %v", len(got), got[:3])
	}
	if st := pushStatus(t, s); st.State != PushDone || st.Sent != 600 {
		t.Fatalf("status = %+v", st)
	}
}

// M5: an unreachable plugin is reported as such (502 in the handler).
func TestPush_UnreachablePlugin(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	db.Exec(`UPDATE app_config SET "JF_HOST" = 'http://127.0.0.1:1'`)
	if _, err := s.StartPush(ctx); !errors.Is(err, ErrPluginUnreachable) {
		t.Fatalf("err = %v", err)
	}
}

// M6 + durationRepaired: ids sent in N format, repaired flag sent.
func TestToPlay_NormalizesIdsAndSendsRepaired(t *testing.T) {
	start := t0
	dur := int64(60)
	a := models.JFPlaybackActivity{Id: "x", UserId: sptr("AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE"), ItemId: sptr("11111111-2222-3333-4444-555555555555"),
		NowPlayingItemId: sptr("99999999-2222-3333-4444-555555555555"), MediaType: sptr("Episode"), PlaybackDuration: &dur, StartedAt: &start,
		DurationRepaired: true}
	p := ToPlay(a)
	if p.UserId != "aaaaaaaabbbbccccddddeeeeeeeeeeee" || p.ItemId != "11111111222233334444555555555555" || *p.SeriesId != "99999999222233334444555555555555" {
		t.Fatalf("ids = %q %q %q", p.UserId, p.ItemId, *p.SeriesId)
	}
	if p.DurationRepaired == nil || !*p.DurationRepaired {
		t.Fatalf("durationRepaired = %v", p.DurationRepaired)
	}
	a.DurationRepaired = false
	if p := ToPlay(a); p.DurationRepaired == nil || *p.DurationRepaired {
		t.Fatalf("durationRepaired = %v", p.DurationRepaired)
	}
}
