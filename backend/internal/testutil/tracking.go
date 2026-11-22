package testutil

import (
	"testing"
	"time"

	"gorm.io/gorm"
)

// BackfillTracking fills the accurate-tracking columns (migration 008) of
// playback rows seeded in the legacy shape, exactly like the migration does
// for pre-existing data: ItemId = COALESCE(EpisodeId, NowPlayingItemId),
// MediaType / RunTimeTicks from the catalog, StartedAt from
// ActivityDateInserted, EndedAt = StartedAt + PlaybackDuration and
// IsView = PlaybackDuration >= 30 s. Rows that already carry an ItemId are
// considered explicitly seeded and only get their NULL columns filled.
func BackfillTracking(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		// Legacy-shaped rows are the ones without ItemId: derive IsView first.
		`UPDATE jf_playback_activity SET "IsView" = COALESCE("PlaybackDuration", 0) >= 30 WHERE "ItemId" IS NULL`,
		`UPDATE jf_playback_activity SET "ItemId" = COALESCE("EpisodeId", "NowPlayingItemId") WHERE "ItemId" IS NULL`,
		`UPDATE jf_playback_activity a SET "MediaType" = CASE
		    WHEN EXISTS (SELECT 1 FROM jf_library_episodes e WHERE e."Id" = a."ItemId") THEN 'Episode'
		    WHEN EXISTS (SELECT 1 FROM jf_music_tracks t WHERE t."Id" = a."ItemId") THEN 'Audio'
		    WHEN EXISTS (SELECT 1 FROM jf_library_items i WHERE i."Id" = a."ItemId" AND i."Type" = 'Movie') THEN 'Movie'
		    ELSE 'Other' END
		 WHERE "MediaType" IS NULL`,
		`UPDATE jf_playback_activity a SET "RunTimeTicks" = COALESCE(
		    (SELECT e."RunTimeTicks" FROM jf_library_episodes e WHERE e."Id" = a."ItemId"),
		    (SELECT t."RunTimeTicks" FROM jf_music_tracks t WHERE t."Id" = a."ItemId"),
		    (SELECT i."RunTimeTicks" FROM jf_library_items i WHERE i."Id" = a."ItemId"))
		 WHERE "RunTimeTicks" IS NULL`,
		`UPDATE jf_playback_activity SET "StartedAt" = "ActivityDateInserted"::timestamptz
		 WHERE "StartedAt" IS NULL AND "ActivityDateInserted" IS NOT NULL`,
		`UPDATE jf_playback_activity SET "EndedAt" = "StartedAt" + make_interval(secs => COALESCE("PlaybackDuration", 0))
		 WHERE "EndedAt" IS NULL AND "StartedAt" IS NOT NULL`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("backfill tracking columns: %v", err)
		}
	}
}

// RecentDay returns UTC midnight `daysAgo` days before today, so fixtures stay
// inside default "last 30 days" windows whatever the date the tests run.
func RecentDay(daysAgo int) time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -daysAgo)
}

// TS formats t like the text ActivityDateInserted column.
func TS(t time.Time) string { return t.UTC().Format("2006-01-02 15:04:05-07:00") }
