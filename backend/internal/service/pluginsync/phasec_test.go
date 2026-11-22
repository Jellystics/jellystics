package pluginsync

import (
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
)

// Imported plays never measured seeks: their counters stay NULL, not 0.
func TestMapPlay_ImportsHaveNoSkipCounters(t *testing.T) {
	for _, src := range []string{"import-pbr", "import-jellystics"} {
		p := moviePlay("i", t0, 60_000)
		p.Source = src
		a, ok := MapPlay(p)
		if !ok || a.SeekCount != nil || a.SkippedSeconds != nil {
			t.Fatalf("%s: seek %v skipped %v", src, a.SeekCount, a.SkippedSeconds)
		}
	}
	p := moviePlay("p", t0, 60_000)
	p.Source, p.SeekCount, p.SkippedMs = "plugin", 2, 12_400
	a, _ := MapPlay(p)
	if a.SeekCount == nil || *a.SeekCount != 2 || a.SkippedSeconds == nil || *a.SkippedSeconds != 12 {
		t.Fatalf("plugin: seek %v skipped %v", a.SeekCount, a.SkippedSeconds)
	}
}

// Plugin migration v2 rewrites legacy "added" rows as "present" (same id,
// new version, dateCreated = at): the re-read must overwrite the stored row.
func TestRun_LegacyAddedRewrittenAsPresent(t *testing.T) {
	legacyAt := time.Date(2021, 3, 4, 5, 0, 0, 0, time.UTC)
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) {
		f.library = []jellyfin.JellysticsLibraryEvent{{Id: "add-x", Kind: "added", At: jellyfin.FlexTime{Time: legacyAt}, ItemId: sptr("x"), Version: 1}}
		f.seq = 1
	})
	s, db := newTestService(t, f)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	f.set(func(f *fakePlugin) {
		f.library = []jellyfin.JellysticsLibraryEvent{{Id: "add-x", Kind: "present", At: jellyfin.FlexTime{Time: legacyAt},
			ItemId: sptr("x"), DateCreated: &jellyfin.FlexTime{Time: legacyAt}, Version: 2}}
		f.seq = 2
	})
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	var rows []models.JFLibraryEvent
	db.Find(&rows)
	if len(rows) != 1 || rows[0].Kind != "present" || rows[0].DateCreated == nil || !rows[0].DateCreated.Equal(legacyAt) || rows[0].Version != 2 {
		t.Fatalf("rows = %+v", rows)
	}
}

// A live add on an item with a snapshot row overwrites that row (same id):
// the row becomes a real addition at its new time.
func TestRun_PresentRewrittenAsAdded(t *testing.T) {
	created := time.Date(2020, 1, 2, 3, 0, 0, 0, time.UTC)
	live := t0.Add(48 * time.Hour)
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) {
		f.library = []jellyfin.JellysticsLibraryEvent{{Id: "present-x", Kind: "present", At: jellyfin.FlexTime{Time: t0}, ItemId: sptr("x"),
			DateCreated: &jellyfin.FlexTime{Time: created}, Version: 1}}
		f.seq = 1
	})
	s, db := newTestService(t, f)
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	f.set(func(f *fakePlugin) {
		f.library = []jellyfin.JellysticsLibraryEvent{{Id: "present-x", Kind: "added", At: jellyfin.FlexTime{Time: live}, ItemId: sptr("x"),
			DateCreated: &jellyfin.FlexTime{Time: created}, Version: 2}}
		f.seq = 2
	})
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	var rows []models.JFLibraryEvent
	db.Find(&rows)
	if len(rows) != 1 || rows[0].Kind != "added" || !rows[0].At.Equal(live) || rows[0].DateCreated == nil || !rows[0].DateCreated.Equal(created) {
		t.Fatalf("rows = %+v", rows)
	}
}

// Snapshot ("present") rows and live additions keep the item DateCreated.
func TestRun_LibraryEventsKeepDateCreated(t *testing.T) {
	created := time.Date(2019, 5, 1, 10, 0, 0, 0, time.UTC)
	f := newFakePlugin(t)
	f.set(func(f *fakePlugin) {
		f.library = []jellyfin.JellysticsLibraryEvent{
			{Id: "s1", Kind: "present", At: jellyfin.FlexTime{Time: t0}, ItemId: sptr("a"), DateCreated: &jellyfin.FlexTime{Time: created}, Version: 1},
			{Id: "l1", Kind: "added", At: jellyfin.FlexTime{Time: t0}, ItemId: sptr("b"), Version: 2},
		}
		f.seq = 2
	})
	s, db := newTestService(t, f)
	calls := 0
	s.OnDataChanged = func() { calls++ }
	if _, err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	var snap, live models.JFLibraryEvent
	db.First(&snap, `"Id" = 's1'`)
	db.First(&live, `"Id" = 'l1'`)
	if snap.Kind != "present" || snap.DateCreated == nil || !snap.DateCreated.Equal(created) || live.DateCreated != nil {
		t.Fatalf("snapshot = %+v, live = %+v", snap, live)
	}
	// Annex rows change plugin-fed stats: the caches are reset.
	if calls != 1 {
		t.Fatalf("OnDataChanged calls = %d, want 1", calls)
	}
}
