package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"gorm.io/datatypes"
)

// Plugin rows are interpreted in the configured TZ and skipped when the same
// play was already recorded by the Jellystics tracker.
func TestPluginMerge_TimezoneAndDedup(t *testing.T) {
	t.Setenv("TZ", "Europe/Paris")
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()
	p := func(s string) *string { return &s }
	d := func(v int64) *int64 { return &v }

	if err := repos.Item.Upsert(ctx, []models.JFLibraryItem{{Id: "mov-1", Name: p("Film"), Type: p("Movie"), Genres: datatypes.JSON("[]")}}); err != nil {
		t.Fatal(err)
	}
	// The tracker already recorded the play that started at 20:00 Paris (19:00 UTC).
	started := time.Date(2026, 11, 20, 19, 0, 0, 0, time.UTC)
	ended := started.Add(time.Hour)
	if err := repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{{
		Id: "tracked", UserId: p("u1"), ItemId: p("mov-1"), MediaType: p("Movie"),
		PlaybackDuration: d(3600), StartedAt: &started, EndedAt: &ended, IsView: true, Source: "watchdog",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := repos.PluginData.Upsert(ctx, []models.JFPluginData{
		{RowId: "1", ItemId: p("mov-1"), UserId: p("u1"), ItemType: p("Movie"), PlayDuration: d(3590), DateCreated: p("2026-11-20 20:00:20")}, // same play
		{RowId: "2", ItemId: p("mov-1"), UserId: p("u1"), ItemType: p("Movie"), PlayDuration: d(1200), DateCreated: p("2026-11-21 00:30:20")}, // another play
		{RowId: "3", ItemId: p("other"), UserId: p("u1"), ItemType: p("Book"), PlayDuration: d(500), DateCreated: p("2026-11-21 10:00:00")},   // not tracked
	}); err != nil {
		t.Fatal(err)
	}
	if err := repos.PluginData.MergeIntoPlaybackActivity(ctx); err != nil {
		t.Fatal(err)
	}
	var rows []models.JFPlaybackActivity
	db.Where("imported = true").Find(&rows)
	if len(rows) != 1 || rows[0].Id != "2" {
		t.Fatalf("imported %d rows, want only row 2: %+v", len(rows), rows)
	}
	want := time.Date(2026, 11, 20, 23, 30, 0, 0, time.UTC) // 00:30:20 Paris minus the plugin's 20 s delay
	if !rows[0].StartedAt.Equal(want) {
		t.Errorf("StartedAt = %v, want %v", rows[0].StartedAt.UTC(), want)
	}
	if !rows[0].IsView || *rows[0].MediaType != "Movie" {
		t.Errorf("IsView %v MediaType %v", rows[0].IsView, *rows[0].MediaType)
	}
}

// One play = one row: the legacy merge skips plays already recorded by an
// imported source (plugin PBR import, Jellystics history, TSV import).
func TestPluginMerge_SkipsPlaysCoveredByImports(t *testing.T) {
	t.Setenv("TZ", "UTC")
	ctx := context.Background()
	p := func(s string) *string { return &s }
	d := func(v int64) *int64 { return &v }
	start := time.Date(2026, 11, 10, 20, 0, 0, 0, time.UTC)
	for _, source := range []string{"import-pbr", "import-jellystics", "import"} {
		t.Run(source, func(t *testing.T) {
			db := setupTestDB(t)
			repos := repository.New(db)
			s, e := start.Add(5*time.Second), start.Add(10*time.Minute)
			if err := repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{{Id: "existing", UserId: p("u1"), ItemId: p("mov-1"), MediaType: p("Movie"),
				PlaybackDuration: d(600), StartedAt: &s, EndedAt: &e, Source: source, Imported: true}}); err != nil {
				t.Fatal(err)
			}
			_ = repos.Item.Upsert(ctx, []models.JFLibraryItem{{Id: "mov-1", Name: p("Film"), Type: p("Movie"), Genres: datatypes.JSON("[]")}})
			_ = repos.PluginData.Upsert(ctx, []models.JFPluginData{
				{RowId: "1", ItemId: p("mov-1"), UserId: p("u1"), ItemType: p("Movie"), PlayDuration: d(600), DateCreated: p("2026-11-10 20:00:20")},
			})
			if err := repos.PluginData.MergeIntoPlaybackActivity(ctx); err != nil {
				t.Fatal(err)
			}
			var n int64
			db.Model(&models.JFPlaybackActivity{}).Count(&n)
			if n != 1 {
				t.Fatalf("rows = %d, want 1", n)
			}
		})
	}
}

// Without TZ (Docker default) plugin dates are read as UTC, hours off; the
// same play must still be recognised (similar duration, start within ±14 h).
func TestPluginMerge_DedupWithoutTZ(t *testing.T) {
	t.Setenv("TZ", "")
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()
	p := func(s string) *string { return &s }
	d := func(v int64) *int64 { return &v }

	started := time.Date(2026, 11, 10, 19, 0, 0, 0, time.UTC) // 20:00 Paris
	ended := started.Add(22 * time.Minute)
	_ = repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{{
		Id: "tracked", UserId: p("u1"), ItemId: p("ep-1"), MediaType: p("Episode"),
		PlaybackDuration: d(1320), StartedAt: &started, EndedAt: &ended, IsView: true, Source: "watchdog",
	}})
	_ = repos.PluginData.Upsert(ctx, []models.JFPluginData{
		{RowId: "1", ItemId: p("ep-1"), UserId: p("u1"), ItemType: p("Episode"), PlayDuration: d(1300), DateCreated: p("2026-11-10 20:00:20")},
	})
	if err := repos.PluginData.MergeIntoPlaybackActivity(ctx); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Model(&models.JFPlaybackActivity{}).Where("imported = true").Count(&n)
	if n != 0 {
		t.Fatalf("episode imported a second time when TZ is unset")
	}
}

// Plugin rows of the last day (duration still growing) and plays currently
// tracked are not imported yet.
func TestPluginMerge_SkipsRecentAndLiveRows(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()
	p := func(s string) *string { return &s }
	d := func(v int64) *int64 { return &v }

	recent := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	_ = repos.Watchdog.Upsert(ctx, []models.JFActivityWatchdog{{Id: "sess", UserId: p("u1"), ItemId: p("mov-live")}})
	_ = repos.PluginData.Upsert(ctx, []models.JFPluginData{
		{RowId: "1", ItemId: p("mov-a"), UserId: p("u1"), ItemType: p("Movie"), PlayDuration: d(300), DateCreated: p(recent)},
		{RowId: "2", ItemId: p("mov-live"), UserId: p("u1"), ItemType: p("Movie"), PlayDuration: d(300), DateCreated: p("2026-01-05 20:00:00")},
	})
	if err := repos.PluginData.MergeIntoPlaybackActivity(ctx); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Model(&models.JFPlaybackActivity{}).Count(&n)
	if n != 0 {
		t.Fatalf("imported %d rows, want 0 (recent / live)", n)
	}
}

func TestBackfillTracking_DatesAndDeletedItems(t *testing.T) {
	t.Setenv("TZ", "Europe/Paris")
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	// Legacy rows (no ItemId / StartedAt), items no longer in the catalog.
	if err := db.Exec(`INSERT INTO jf_playback_activity ("Id","UserId","NowPlayingItemId","EpisodeId","SeasonId","PlaybackDuration","ActivityDateInserted",imported) VALUES
		('restored-z', 'u1', 'series-x', 'ep-x', 'season-x', 1200, '2024-01-01T10:00:00Z', true),
		('legacy-off', 'u1', 'album-x', 'track-x', NULL, 20, '2024-01-01 10:00:00.000+01:00', false),
		('plugin-old', 'u1', 'movie-x', NULL, NULL, 90, '2024-01-01 10:00:20', true)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := repos.Playback.BackfillTracking(ctx); err != nil {
		t.Fatal(err)
	}
	get := func(id string) models.JFPlaybackActivity {
		var a models.JFPlaybackActivity
		db.Where(`"Id" = ?`, id).First(&a)
		return a
	}
	cases := []struct {
		id, item, mt string
		start        time.Time
		view         bool
	}{
		{"restored-z", "ep-x", "Episode", time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), true},  // offset honoured even for imported rows
		{"legacy-off", "track-x", "Audio", time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC), false}, // +01:00
		{"plugin-old", "movie-x", "Movie", time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC), true},  // Paris local, minus the plugin's 20 s
	}
	for _, c := range cases {
		a := get(c.id)
		if a.ItemId == nil || *a.ItemId != c.item || a.MediaType == nil || *a.MediaType != c.mt {
			t.Errorf("%s: item %v type %v, want %s %s", c.id, a.ItemId, a.MediaType, c.item, c.mt)
		}
		if a.StartedAt == nil || !a.StartedAt.Equal(c.start) {
			t.Errorf("%s: StartedAt %v, want %v", c.id, a.StartedAt, c.start)
		}
		if a.IsView != c.view {
			t.Errorf("%s: IsView %v, want %v", c.id, a.IsView, c.view)
		}
		if a.EndedAt == nil {
			t.Errorf("%s: EndedAt not set", c.id)
		}
	}
}
