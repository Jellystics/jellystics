package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/repository"
	"gorm.io/gorm"
)

// BackfillTracking fills the accurate-tracking columns of legacy-shaped
// fixtures using the production backfill (playbackRepo.BackfillTracking).
func BackfillTracking(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := repository.New(db).Playback.BackfillTracking(context.Background()); err != nil {
		t.Fatalf("backfill tracking columns: %v", err)
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
