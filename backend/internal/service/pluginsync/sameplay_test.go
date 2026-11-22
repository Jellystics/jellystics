package pluginsync

import (
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
)

// Back-to-back replay of a 3-min track: A 0:00–3:00, B 3:00–6:00. Two plays,
// whatever the source mix and the page boundaries.
var (
	trackA = t0
	trackB = t0.Add(3 * time.Minute)
)

func trackPlay(id, source string, start time.Time) jellyfin.JellysticsPlay {
	p := moviePlay(id, start, 180_000)
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

func TestSamePlay_BackToBackImportJellysticsOnPageBoundary(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.pageSize = 1 })
	s, db := newTestService(t, f)
	trackRow(t, s, "pbr-1", "import-pbr", "", trackA, 180) // only A known locally
	f.putPlay(trackPlay("js-a", "import-jellystics", trackA))
	f.putPlay(trackPlay("js-b", "import-jellystics", trackB))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db); got != "[pbr-1 js-b]" {
		t.Fatalf("rows = %s, want [pbr-1 js-b]", got)
	}
}

// Rule 1: Jellystics history rows never block each other.
func TestSamePlay_ImportJellysticsNotBlockedByImportJellystics(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	trackRow(t, s, "js-a", "import-jellystics", "", trackA, 180)
	f.putPlay(trackPlay("js-a2", "import-jellystics", trackA.Add(10*time.Second)))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db); got != "[js-a js-a2]" {
		t.Fatalf("rows = %s", got)
	}
}

func TestSamePlay_BackToBackPBRWithTrackerCoverage(t *testing.T) {
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) { f.pageSize = 1 })
	s, db := newTestService(t, f)
	trackRow(t, s, "tr-a", "watchdog", "dev-1", trackA, 180) // the tracker saw A only
	trackRow(t, s, "10", "plugin", "", trackA.Add(20*time.Second), 180)
	f.putPlay(trackPlay("pbr-11", "import-pbr", trackB))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	// Legacy A stays (it is not the same play as B); B is imported.
	if got := ids(t, db); got != "[tr-a 10 pbr-11]" {
		t.Fatalf("rows = %s", got)
	}
}

func TestSamePlay_BackToBackTrackerAndPluginPlay(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	trackRow(t, s, "tr-a", "watchdog", "dev-1", trackA, 180)
	f.putPlay(trackPlay("p-b", "plugin", trackB))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db); got != "[tr-a p-b]" {
		t.Fatalf("rows = %s", got)
	}
}

// Genuine duplicates (start ±30 s, similar duration) still collapse.
func TestSamePlay_GenuineDuplicatesCollapse(t *testing.T) {
	f := newFakePlugin(t)
	s, db := newTestService(t, f)
	trackRow(t, s, "tr-a", "watchdog", "dev-1", trackA.Add(30*time.Second), 170)
	f.putPlay(trackPlay("pbr-1", "import-pbr", trackA))       // covered by the tracker row
	f.putPlay(trackPlay("js-a", "import-jellystics", trackA)) // same play again
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db); got != "[tr-a]" {
		t.Fatalf("rows = %s", got)
	}
	f.putPlay(trackPlay("p-a", "plugin", trackA.Add(25*time.Second)))
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db); got != "[p-a]" {
		t.Fatalf("rows = %s, want the plugin play only", got)
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
