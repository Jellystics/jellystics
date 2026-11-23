package handler_test

import (
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/testutil"
)

type dataQualityJSON struct {
	Totals  dqCounts
	Sources []struct {
		Key string
		dqCounts
		ViewShare, WatchShare float64
	}
	PluginCollectedSince *string
	CollectedSince       *string
	Undated              int
}

type dqCounts struct {
	Clicks, Views                                                         int
	WatchSeconds                                                          int64
	Repaired, Captured, Positions, Streams, Transcodes, TranscodeDetailed int
}

func TestV2DataQuality(t *testing.T) {
	t.Setenv("TZ", "UTC")
	db := testutil.NewDB(t)
	day := testutil.RecentDay(3).Add(10 * time.Hour)
	old := testutil.RecentDay(40).Add(10 * time.Hour)
	row := func(id, source, mt string, start time.Time, dur int64, imported bool) models.JFPlaybackActivity {
		r := trackedRow(id, "u1", "item-"+id, mt, start, dur, 3000, nil)
		r.Source, r.Imported = source, imported
		return r
	}
	pos := func(r models.JFPlaybackActivity) models.JFPlaybackActivity {
		r.PositionTicksEnd = i64(100 * secTicks)
		return r
	}
	transcode := func(r models.JFPlaybackActivity, detailed bool) models.JFPlaybackActivity {
		r.PlayMethod = sp("Transcode")
		if detailed {
			r.TranscodingInfo = js(`{"VideoCodec":"h264"}`)
		}
		return r
	}
	repaired := func(r models.JFPlaybackActivity) models.JFPlaybackActivity { r.DurationRepaired = true; return r }

	p1 := pos(row("p1", "jellystics-plugin", "Movie", day, 1000, false))
	p1.AudioStreamIndex, p1.PauseCount = ip(1), 2
	undated := row("undated", "watchdog", "Movie", day, 100, false)
	undated.StartedAt = nil
	undated2 := row("undated-u2", "watchdog", "Episode", day, 100, false)
	undated2.StartedAt, undated2.UserId = nil, sp("u2")
	undatedOther := row("undated-other", "watchdog", "Other", day, 100, false)
	undatedOther.StartedAt = nil
	rows := []models.JFPlaybackActivity{
		p1,
		pos(transcode(row("p2", "jellystics-plugin", "Episode", day, 500, false), true)),
		pos(transcode(row("w1", "watchdog", "Movie", day, 300, false), false)),
		row("w2", "watchdog", "Movie", day, 10, false),
		repaired(row("l1", "plugin", "Episode", day, 2700, true)),
		row("l2", "import-pbr", "Movie", day, 100, true),
		repaired(row("i1", "import", "Movie", day, 0, true)),
		row("r1", "restore", "Movie", day, 400, false),
		row("j1", "import-jellystics", "Movie", day, 200, true),
		row("x1", "weird", "Movie", day, 50, false),
		row("p0", "jellystics-plugin", "Movie", old, 60, false), // before the period
		row("w0", "watchdog", "Movie", old, 999, false),         // before the period
		undated, undated2, undatedOther,
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	var q dataQualityJSON
	v2JSON(t, db, "/api/v2/data-quality", &q)
	want := dqCounts{Clicks: 10, Views: 8, WatchSeconds: 5260, Repaired: 2, Captured: 1, Positions: 3, Streams: 1, Transcodes: 2, TranscodeDetailed: 1}
	if q.Totals != want {
		t.Fatalf("totals = %+v, want %+v", q.Totals, want)
	}
	wantSources := map[string]dqCounts{
		"plugin":            {Clicks: 2, Views: 2, WatchSeconds: 1500, Captured: 1, Positions: 2, Streams: 1, Transcodes: 1, TranscodeDetailed: 1},
		"tracker":           {Clicks: 2, Views: 1, WatchSeconds: 310, Positions: 1, Transcodes: 1},
		"playbackReporting": {Clicks: 3, Views: 2, WatchSeconds: 2800, Repaired: 2},
		"otherImports":      {Clicks: 1, Views: 1, WatchSeconds: 200},
		"restore":           {Clicks: 1, Views: 1, WatchSeconds: 400},
		"other":             {Clicks: 1, Views: 1, WatchSeconds: 50},
	}
	order := []string{"plugin", "tracker", "playbackReporting", "otherImports", "restore", "other"}
	if len(q.Sources) != len(order) {
		t.Fatalf("sources = %+v", q.Sources)
	}
	for i, s := range q.Sources {
		if s.Key != order[i] || s.dqCounts != wantSources[s.Key] {
			t.Errorf("source %d = %s %+v, want %s %+v", i, s.Key, s.dqCounts, order[i], wantSources[order[i]])
		}
		if !near(s.WatchShare, float64(s.WatchSeconds)/5260) || !near(s.ViewShare, float64(s.Views)/8) {
			t.Errorf("%s shares = %v / %v", s.Key, s.ViewShare, s.WatchShare)
		}
	}
	if q.PluginCollectedSince == nil || *q.PluginCollectedSince != old.Format("2006-01-02") {
		t.Errorf("pluginCollectedSince = %v", q.PluginCollectedSince)
	}
	// Undated: only tracked media types (the "Other" row is not a play).
	if q.CollectedSince == nil || *q.CollectedSince != day.Format("2006-01-02") || q.Undated != 2 {
		t.Errorf("collectedSince = %v, undated = %d", q.CollectedSince, q.Undated)
	}
	// User filter applies to the undated count too.
	v2JSON(t, db, "/api/v2/data-quality?userId=u1", &q)
	if q.Undated != 1 || q.Totals.Clicks != 10 {
		t.Errorf("u1: undated = %d, clicks = %d", q.Undated, q.Totals.Clicks)
	}

	// Period-aware: only the two old plays.
	from, to := old.Format("2006-01-02"), old.AddDate(0, 0, 1).Format("2006-01-02")
	v2JSON(t, db, "/api/v2/data-quality?from="+from+"&to="+to, &q)
	if q.Totals.Clicks != 2 || q.Totals.WatchSeconds != 1059 || q.Sources[0].Clicks != 1 || q.Sources[1].Clicks != 1 ||
		!near(q.Sources[0].WatchShare, 60.0/1059) {
		t.Fatalf("old period = %+v", q)
	}

	// Empty period: zero shares, every source listed.
	v2JSON(t, db, "/api/v2/data-quality?from=2001-01-01&to=2001-01-02", &q)
	if q.Totals.Clicks != 0 || len(q.Sources) != 6 || q.Sources[0].WatchShare != 0 {
		t.Fatalf("empty period = %+v", q)
	}
}

// Days are local (display TZ) and `to` is exclusive.
func TestV2DataQuality_LocalDaysToExclusive(t *testing.T) {
	t.Setenv("TZ", "Pacific/Auckland") // UTC+13 in March (NZDT)
	db := testutil.NewDB(t)
	late := time.Date(2026, 3, 10, 10, 30, 0, 0, time.UTC) // 23:30 local, March 10
	next := time.Date(2026, 3, 10, 11, 30, 0, 0, time.UTC) // 00:30 local, March 11
	early := time.Date(2026, 3, 9, 10, 59, 0, 0, time.UTC) // 23:59 local, March 9
	rows := []models.JFPlaybackActivity{
		trackedRow("a", "u1", "m1", "Movie", late, 100, 3000, nil),
		trackedRow("b", "u1", "m1", "Movie", next, 200, 3000, nil),
		trackedRow("c", "u1", "m1", "Movie", early, 400, 3000, nil),
	}
	rows[1].Source = "jellystics-plugin"
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	var q dataQualityJSON
	v2JSON(t, db, "/api/v2/data-quality?from=2026-03-10&to=2026-03-11", &q)
	if q.Totals.Clicks != 1 || q.Totals.WatchSeconds != 100 || q.Sources[1].Clicks != 1 {
		t.Fatalf("totals = %+v", q.Totals)
	}
	if q.PluginCollectedSince == nil || *q.PluginCollectedSince != "2026-03-11" {
		t.Fatalf("pluginCollectedSince = %v", q.PluginCollectedSince)
	}
}
