package pluginsync

import (
	"fmt"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
)

// Back-to-back replays: A 0:00–L, B L–2L. Two plays whatever the source mix
// and the page boundaries, for a 3-min and a 90-s track.
var trackLengths = []int64{180, 90}

var trackA = t0

func trackB(secs int64) time.Time { return t0.Add(time.Duration(secs) * time.Second) }

func trackPlay(id, source string, start time.Time, secs int64) jellyfin.JellysticsPlay {
	p := moviePlay(id, start, secs*1000)
	p.ItemType, p.ItemId = sptr("Audio"), "track-1"
	p.Source = source
	if source != "plugin" {
		p.DeviceId = nil
	}
	return p
}

func trackRow(t *testing.T, s *Service, id, source, device string, start time.Time, secs int64) {
	t.Helper()
	end := start.Add(time.Duration(secs) * time.Second)
	a := models.JFPlaybackActivity{Id: id, UserId: sptr("user-1"), ItemId: sptr("track-1"), NowPlayingItemId: sptr("track-1"),
		MediaType: sptr("Audio"), StartedAt: &start, EndedAt: &end, PlaybackDuration: &secs, IsView: true, Source: source,
		Imported: source != "watchdog" && source != "jellystics-plugin"}
	if device != "" {
		a.DeviceId = sptr(device)
	}
	if err := s.repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{a}); err != nil {
		t.Fatal(err)
	}
}

func eachLength(t *testing.T, fn func(t *testing.T, L int64)) {
	for _, L := range trackLengths {
		t.Run(fmt.Sprintf("%ds", L), func(t *testing.T) { fn(t, L) })
	}
}

func TestSamePlay_BackToBackImportJellysticsOnPageBoundary(t *testing.T) {
	eachLength(t, func(t *testing.T, L int64) {
		f := newFakePlugin(t)
		f.set(func(f *fakePlugin) { f.pageSize = 1 })
		s, db := newTestService(t, f)
		trackRow(t, s, "pbr-1", "import-pbr", "", trackA, L) // only A known locally
		f.putPlay(trackPlay("js-a", "import-jellystics", trackA, L))
		f.putPlay(trackPlay("js-b", "import-jellystics", trackB(L), L))
		if _, err := s.Run(ctx); err != nil {
			t.Fatal(err)
		}
		if got := ids(t, db); got != "[pbr-1 js-b]" {
			t.Fatalf("rows = %s, want [pbr-1 js-b]", got)
		}
	})
}

// Rule 1: Jellystics history rows never block each other.
func TestSamePlay_ImportJellysticsNotBlockedByImportJellystics(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	trackRow(t, s, "js-a", "import-jellystics", "", trackA, 180)
	f.putPlay(trackPlay("js-a2", "import-jellystics", trackA.Add(10*time.Second), 180))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db); got != "[js-a js-a2]" {
		t.Fatalf("rows = %s", got)
	}
}

func TestSamePlay_BackToBackPBRWithTrackerCoverage(t *testing.T) {
	eachLength(t, func(t *testing.T, L int64) {
		f := newFakePlugin(t)
		f.set(func(f *fakePlugin) { f.pageSize = 1 })
		s, db := newTestService(t, f)
		trackRow(t, s, "tr-a", "watchdog", "dev-1", trackA, L) // the tracker saw A only
		trackRow(t, s, "10", "plugin", "", trackA.Add(5*time.Second), L)
		f.putPlay(trackPlay("pbr-11", "import-pbr", trackB(L), L))
		if _, err := s.Run(ctx); err != nil {
			t.Fatal(err)
		}
		// Legacy A stays (it is not the same play as B); B is imported.
		if got := ids(t, db); got != "[tr-a 10 pbr-11]" {
			t.Fatalf("rows = %s", got)
		}
	})
}

func TestSamePlay_BackToBackTrackerAndPluginPlay(t *testing.T) {
	eachLength(t, func(t *testing.T, L int64) {
		f := newFakePlugin(t)
		s, db := newTestService(t, f)
		trackRow(t, s, "tr-a", "watchdog", "dev-1", trackA, L)
		f.putPlay(trackPlay("p-b", "plugin", trackB(L), L))
		if _, err := s.Run(ctx); err != nil {
			t.Fatal(err)
		}
		if got := ids(t, db); got != "[tr-a p-b]" {
			t.Fatalf("rows = %s", got)
		}
	})
}

// While the plugin is unhealthy, the tracker play B is kept next to the
// plugin play A (closed or a stale open checkpoint).
func TestSamePlay_BackToBackFilterTrackerPlays(t *testing.T) {
	eachLength(t, func(t *testing.T, L int64) {
		f := newFakePlugin(t)
		s, db := newTestService(t, f)
		open := trackPlay("p-a", "plugin", trackA, L)
		open.EndedAt, open.State = nil, "open"
		f.putPlay(open)
		if _, err := s.Run(ctx); err != nil {
			t.Fatal(err)
		}
		start, end := trackB(L), trackB(2*L)
		b := models.JFPlaybackActivity{Id: "tr-b", UserId: sptr("user-1"), ItemId: sptr("track-1"), DeviceId: sptr("dev-1"),
			StartedAt: &start, EndedAt: &end, PlaybackDuration: &L, Source: "watchdog"}
		kept, err := s.repos.PluginSync.FilterTrackerPlays(ctx, []models.JFPlaybackActivity{b})
		if err != nil || len(kept) != 1 {
			t.Fatalf("kept = %v, %v", kept, err)
		}
		if got := ids(t, db); got != "[p-a]" {
			t.Fatalf("open plugin play A deleted: rows = %s", got)
		}
	})
}

// Genuine duplicates still collapse: start ±30 s / similar duration for a
// 3-min track, start ±5 s / same duration for a 90-s track.
func TestSamePlay_GenuineDuplicatesCollapse(t *testing.T) {
	for _, c := range []struct {
		L      int64
		offset time.Duration
		trDur  int64
	}{{180, 30 * time.Second, 170}, {90, 5 * time.Second, 90}} {
		t.Run(fmt.Sprintf("%ds", c.L), func(t *testing.T) {
			f := newFakePlugin(t)
			s, db := newTestService(t, f)
			trackRow(t, s, "tr-a", "watchdog", "dev-1", trackA.Add(c.offset), c.trDur)
			f.putPlay(trackPlay("pbr-1", "import-pbr", trackA, c.L))       // covered by the tracker row
			f.putPlay(trackPlay("js-a", "import-jellystics", trackA, c.L)) // same play again
			if _, err := s.Run(ctx); err != nil {
				t.Fatal(err)
			}
			if got := ids(t, db); got != "[tr-a]" {
				t.Fatalf("rows = %s", got)
			}
			f.putPlay(trackPlay("p-a", "plugin", trackA.Add(c.offset-time.Second), c.L))
			if _, err := s.Run(ctx); err != nil {
				t.Fatal(err)
			}
			if got := ids(t, db); got != "[p-a]" {
				t.Fatalf("rows = %s, want the plugin play only", got)
			}
		})
	}
}

// A partial tracker row inside a plugin play (tracker took over mid-play) is
// still replaced by the plugin play.
func TestSamePlay_PartialTrackerRowInsidePluginPlay(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	insertTrackerRow(t, db, "tr-part", "dev-1", t0.Add(20*time.Minute), 300)
	f.putPlay(moviePlay("p1", t0, 3_600_000))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db); got != "[p1]" {
		t.Fatalf("rows = %s", got)
	}
}
