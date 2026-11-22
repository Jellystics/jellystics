package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var spStart = time.Date(2026, 11, 10, 20, 0, 0, 0, time.UTC)

func spRow(id, source string, start time.Time, secs int64) models.JFPlaybackActivity {
	end := start.Add(time.Duration(secs) * time.Second)
	u, i, mt := "u1", "track-1", "Audio"
	return models.JFPlaybackActivity{Id: id, UserId: &u, ItemId: &i, NowPlayingItemId: &i, MediaType: &mt, StartedAt: &start, EndedAt: &end,
		PlaybackDuration: &secs, IsView: true, Source: source, Imported: source != "watchdog" && source != "jellystics-plugin"}
}

func spIDs(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var ids []string
	if err := db.Raw(`SELECT "Id" FROM jf_playback_activity ORDER BY "StartedAt", "Id"`).Scan(&ids).Error; err != nil {
		t.Fatal(err)
	}
	return fmt.Sprint(ids)
}

// TSV rows: B right after a plugin play A is kept; a duplicate of A is dropped.
func TestSamePlay_TSVBackToBack(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()
	if err := repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{
		spRow("p-a", "jellystics-plugin", spStart, 180),
		spRow("tsv-a", "import", spStart.Add(20*time.Second), 170),
		spRow("tsv-b", "import", spStart.Add(3*time.Minute+20*time.Second), 180),
	}); err != nil {
		t.Fatal(err)
	}
	n, err := repos.Playback.DeleteCoveredTSVImports(ctx, []string{"tsv-a", "tsv-b"})
	if err != nil || n != 1 {
		t.Fatalf("deleted %d, %v", n, err)
	}
	if got := spIDs(t, db); got != "[p-a tsv-b]" {
		t.Fatalf("rows = %s", got)
	}
}

// Legacy merge: B right after a tracker play A is imported; A again is not.
func TestSamePlay_LegacyMergeBackToBack(t *testing.T) {
	t.Setenv("TZ", "UTC")
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()
	p := func(s string) *string { return &s }
	d := func(v int64) *int64 { return &v }
	_ = repos.MusicTrack.Upsert(ctx, []models.JFMusicTrack{{Id: "track-1", Name: p("Song"), RunTimeTicks: d(180 * 10_000_000), Genres: datatypes.JSON("[]")}})
	if err := repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{spRow("tr-a", "watchdog", spStart, 180)}); err != nil {
		t.Fatal(err)
	}
	_ = repos.PluginData.Upsert(ctx, []models.JFPluginData{
		{RowId: "1", ItemId: p("track-1"), UserId: p("u1"), ItemType: p("Audio"), PlayDuration: d(180), DateCreated: p("2026-11-10 20:00:20")}, // A
		{RowId: "2", ItemId: p("track-1"), UserId: p("u1"), ItemType: p("Audio"), PlayDuration: d(180), DateCreated: p("2026-11-10 20:03:20")}, // B
	})
	if err := repos.PluginData.MergeIntoPlaybackActivity(ctx); err != nil {
		t.Fatal(err)
	}
	if got := spIDs(t, db); got != "[tr-a 2]" {
		t.Fatalf("rows = %s", got)
	}
}

// Restore into a live database: the same row is updated by Id, a duplicate of
// an existing play from another source is skipped, new plays are restored.
func TestRestorePlays_LiveAndEmpty(t *testing.T) {
	ctx := context.Background()
	backup := func() []models.JFPlaybackActivity {
		same := spRow("tr-a", "watchdog", spStart, 185) // same row, newer duration
		return []models.JFPlaybackActivity{
			same,
			spRow("old-a", "plugin", spStart.Add(20*time.Second), 180),               // legacy duplicate of tr-a
			spRow("old-b", "plugin", spStart.Add(3*time.Minute+20*time.Second), 180), // next play: restored
		}
	}

	db := setupTestDB(t)
	repos := repository.New(db)
	if err := repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{spRow("tr-a", "watchdog", spStart, 180)}); err != nil {
		t.Fatal(err)
	}
	restored, skipped, err := repos.Playback.RestorePlays(ctx, backup())
	if err != nil || restored != 2 || skipped != 1 {
		t.Fatalf("live: restored %d skipped %d, %v", restored, skipped, err)
	}
	if got := spIDs(t, db); got != "[tr-a old-b]" {
		t.Fatalf("live rows = %s", got)
	}
	var a models.JFPlaybackActivity
	db.First(&a, `"Id" = ?`, "tr-a")
	if *a.PlaybackDuration != 185 {
		t.Fatalf("tr-a not updated: %d", *a.PlaybackDuration)
	}

	empty := setupTestDB(t)
	restored, skipped, err = repository.New(empty).Playback.RestorePlays(ctx, backup())
	// Empty database: the backup is restored as is (only existing rows are
	// compared against).
	if err != nil || restored != 3 || skipped != 0 || spIDs(t, empty) != "[tr-a old-a old-b]" {
		t.Fatalf("empty: restored %d skipped %d, %v, rows %s", restored, skipped, err, spIDs(t, empty))
	}
}
