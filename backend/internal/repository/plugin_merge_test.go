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
		{RowId: "3", ItemId: p("other"), UserId: p("u1"), ItemType: p("Book"), PlayDuration: d(500), DateCreated: p("2026-11-21 10:00:00")},  // not tracked
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
