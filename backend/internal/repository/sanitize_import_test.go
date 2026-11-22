package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
)

// Imported Playback Reporting rows with impossible durations (negative
// int32 overflow, sessions left open for days) are repaired; tracked rows are not.
func TestBackfillTracking_SanitizesImportedDurations(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()
	p := func(s string) *string { return &s }
	d := func(v int64) *int64 { return &v }
	start := time.Date(2026, 5, 8, 14, 0, 0, 0, time.UTC)
	ep := int64(45 * 60 * 10_000_000) // 45 min runtime
	row := func(id string, dur int64, rt *int64, imported bool) models.JFPlaybackActivity {
		s := start
		return models.JFPlaybackActivity{Id: id, UserId: p("u1"), ItemId: p("i-" + id), MediaType: p("Episode"), PlaybackDuration: d(dur), RunTimeTicks: rt, StartedAt: &s, Imported: imported, Source: map[bool]string{true: "import", false: "watchdog"}[imported]}
	}
	if err := repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{
		row("neg", -2147483648, &ep, true),
		row("open", 166552, &ep, true),
		row("long-no-rt", 100000, nil, true),
		row("ok", 3000, &ep, true), // < 1.5× runtime: kept
		row("tracked", 100000, nil, false),
	}); err != nil {
		t.Fatal(err)
	}
	if err := repos.Playback.BackfillTracking(ctx); err != nil {
		t.Fatal(err)
	}
	want := map[string]int64{"neg": 0, "open": 2700, "long-no-rt": 43200, "ok": 3000, "tracked": 100000}
	for id, w := range want {
		var a models.JFPlaybackActivity
		if err := db.First(&a, "\"Id\" = ?", id).Error; err != nil {
			t.Fatal(err)
		}
		if a.PlaybackDuration == nil || *a.PlaybackDuration != w {
			t.Errorf("%s: duration = %v, want %d", id, a.PlaybackDuration, w)
		}
		if id != "tracked" && (a.EndedAt == nil || !a.EndedAt.Equal(start.Add(time.Duration(w)*time.Second))) {
			t.Errorf("%s: EndedAt = %v, want start+%ds", id, a.EndedAt, w)
		}
		if wantRepaired := id == "neg" || id == "open" || id == "long-no-rt"; a.DurationRepaired != wantRepaired {
			t.Errorf("%s: DurationRepaired = %v, want %v", id, a.DurationRepaired, wantRepaired)
		}
	}
	var neg models.JFPlaybackActivity
	db.First(&neg, "\"Id\" = ?", "neg")
	if neg.IsView {
		t.Error("neg: a 0 s play must not count as a view")
	}
}
