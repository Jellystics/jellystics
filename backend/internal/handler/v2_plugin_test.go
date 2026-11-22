package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/service/auth"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func bp(b bool) *bool { return &b }

var (
	v2Day4 = testutil.RecentDay(4)
	v2Day5 = testutil.RecentDay(5)
)

// pluginRow is a Jellystics-plugin play (positions in seconds).
func pluginRow(id, user, item, mt string, start time.Time, dur, runtime, posStart, posEnd int64, skipped, seeks int, ptc *bool, open bool) models.JFPlaybackActivity {
	r := trackedRow(id, user, item, mt, start, dur, runtime, &posEnd)
	r.Source = "jellystics-plugin"
	r.PositionTicksStart = i64(posStart * secTicks)
	r.SkippedSeconds, r.SeekCount = &skipped, &seeks
	r.PlayedToCompletion = ptc
	r.PluginOpen = bp(open)
	switch mt {
	case "Episode":
		r.NowPlayingItemId, r.EpisodeId, r.SeriesName = sp("s1"), sp(item), sp("Les Rivages")
	case "Audio":
		r.NowPlayingItemId = sp("al1")
	}
	return r
}

// seedPlugin adds plugin plays on top of seedV2 (day1, UTC). Plugin rows of
// e1 / m1 carry a 600 s runtime (PlayedToCompletion is only trusted from
// 300 s on):
//
//	p1 u1 e1 10:00 480 s start 0     stop 600  skipped 20 s (intro)          PTC true
//	p2 u2 e1 11:00 360 s start 0     stop 600  skipped 240 s (> 15 %)        PTC false
//	p3 u2 e1 12:00 300 s start 0     stop 570  no skip                       PTC null
//	p4 u3 e1 13:00 240 s OPEN        stop 240  skipped 30 s                  (provisional)
//	p5 u1 e1 14:00  10 s click                 skipped 5 s
//	p7 u1 e1 16:00 120 s start 300   stop 600  skipped 15 s (not at start)   PTC true
//	p8 u2 e1 17:00  60 s start 0     stop 60   skipped 5 s (< 10 s)          PTC null
//	p6 u1 m1 15:00 300 s start 0     stop 300  (ratio 0.5)                   PTC true
func seedPlugin(t *testing.T, db *gorm.DB) {
	t.Helper()
	at := func(day time.Time, h int) time.Time { return day.Add(time.Duration(h) * time.Hour) }
	rows := []models.JFPlaybackActivity{
		pluginRow("p1", "u1", "e1", "Episode", at(v2Day1, 10), 480, 600, 0, 600, 20, 1, bp(true), false),
		pluginRow("p2", "u2", "e1", "Episode", at(v2Day1, 11), 360, 600, 0, 600, 240, 2, bp(false), false),
		pluginRow("p3", "u2", "e1", "Episode", at(v2Day1, 12), 300, 600, 0, 570, 0, 0, nil, false),
		pluginRow("p4", "u3", "e1", "Episode", at(v2Day1, 13), 240, 600, 0, 240, 30, 1, nil, true),
		pluginRow("p5", "u1", "e1", "Episode", at(v2Day1, 14), 10, 600, 0, 10, 5, 1, nil, false),
		pluginRow("p7", "u1", "e1", "Episode", at(v2Day1, 16), 120, 600, 300, 600, 15, 1, bp(true), false),
		pluginRow("p8", "u2", "e1", "Episode", at(v2Day1, 17), 60, 600, 0, 60, 5, 1, nil, false),
		pluginRow("p6", "u1", "m1", "Movie", at(v2Day1, 15), 300, 600, 0, 300, 0, 0, bp(true), false),
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed plugin: %v", err)
	}
}

type pteResp struct {
	Plays, Finished, Measured, MeasuredFinished int
	Ratio                                       *float64
}

func TestV2PluginEpisodeSkipsAndPlayedToEnd(t *testing.T) {
	db := v2Seeded(t)
	seedPlugin(t, db)
	var got struct {
		Totals struct {
			Views         int
			AvgCompletion *float64
		}
		Funnel struct{ Views, Completed int }
		Skips  struct {
			Plays, Skipped, Intro, IntroPlays      int
			SkipRate, IntroRate, AvgSkippedSeconds *float64
		}
		PlayedToEnd pteResp
		PluginSince *string
		Viewers     []struct {
			User       struct{ ID string }
			Completion *float64
		}
	}
	v2JSON(t, db, "/api/v2/episodes/e1", &got)
	s := got.Skips
	// skipRate over every measured play; introRate over the plays started
	// at the beginning (p7 started at 50 %).
	if s.Plays != 5 || s.Skipped != 4 || s.IntroPlays != 4 || s.Intro != 1 || !approx(s.SkipRate, 0.8) || !approx(s.IntroRate, 0.25) || !approx(s.AvgSkippedSeconds, 70) {
		t.Errorf("skips = %+v", s)
	}
	// (user, item, day) units: u1 day2 (tracker, estimated finished), u1
	// day1 (PTC true), u2 day1 (PTC false although it stopped at the end).
	pe := got.PlayedToEnd
	if pe.Plays != 3 || pe.Finished != 2 || pe.Measured != 2 || pe.MeasuredFinished != 1 || !approx(pe.Ratio, 2.0/3) {
		t.Errorf("playedToEnd = %+v", pe)
	}
	// The funnel's last step is the same metric.
	if got.Funnel.Completed != 2 {
		t.Errorf("funnel completed = %d, want 2 (= playedToEnd.finished)", got.Funnel.Completed)
	}
	if got.PluginSince == nil || *got.PluginSince != v2Day1.Format("2006-01-02") {
		t.Errorf("pluginSince = %v", got.PluginSince)
	}
	// The open play p4 is a view but stays out of completion (else < 1).
	if got.Totals.Views != 7 || !approx(got.Totals.AvgCompletion, 1) {
		t.Errorf("totals = %+v", got.Totals)
	}
	for _, v := range got.Viewers {
		if v.User.ID == "u3" && v.Completion != nil {
			t.Errorf("open play completion = %v, want null", *v.Completion)
		}
	}
}

func TestV2PluginSeriesAndInsightsSkips(t *testing.T) {
	db := v2Seeded(t)
	seedPlugin(t, db)
	var series struct {
		Skips struct {
			Plays     int
			IntroRate *float64
		}
		PluginSince *string
		Seasons     []struct {
			Episodes []struct {
				ID                  string
				SkipPlays           int
				IntroRate, SkipRate *float64
				AvgSkippedSeconds   *float64
			}
		}
	}
	v2JSON(t, db, "/api/v2/series/s1", &series)
	if series.Skips.Plays != 5 || !approx(series.Skips.IntroRate, 0.25) || series.PluginSince == nil {
		t.Errorf("series skips = %+v", series.Skips)
	}
	e1, e2 := series.Seasons[0].Episodes[0], series.Seasons[0].Episodes[1]
	if e1.ID != "e1" || e1.SkipPlays != 5 || !approx(e1.IntroRate, 0.25) || !approx(e1.SkipRate, 0.8) || !approx(e1.AvgSkippedSeconds, 70) {
		t.Errorf("e1 = %+v", e1)
	}
	if e2.SkipPlays != 0 || e2.IntroRate != nil || e2.AvgSkippedSeconds != nil {
		t.Errorf("e2 (not measured) = %+v", e2)
	}

	var ins struct {
		CollectedSince *string
		Totals         struct{ Plays, Intro int }
		Series         []struct {
			ID, Name            string
			Plays               int
			IntroRate, SkipRate *float64
		}
	}
	v2JSON(t, db, "/api/v2/insights/skips", &ins)
	if ins.CollectedSince == nil || ins.Totals.Plays != 5 || ins.Totals.Intro != 1 || len(ins.Series) != 1 ||
		ins.Series[0].ID != "s1" || ins.Series[0].Name != "Les Rivages" || ins.Series[0].Plays != 5 || !approx(ins.Series[0].IntroRate, 0.25) {
		t.Errorf("insights skips = %+v", ins)
	}
	// Under 5 measured plays, a series is not ranked.
	v2JSON(t, db, "/api/v2/insights/skips?userId=u2", &ins)
	if ins.Totals.Plays != 3 || len(ins.Series) != 0 {
		t.Errorf("insights skips u2 = %+v", ins)
	}
}

func TestV2PluginMovieAndUserPlayedToEnd(t *testing.T) {
	db := v2Seeded(t)
	seedPlugin(t, db)
	var movie struct {
		Totals      struct{ CompletedRatio *float64 }
		Funnel      struct{ Completed int }
		PlayedToEnd pteResp
		PluginSince *string
	}
	v2JSON(t, db, "/api/v2/movies/m1", &movie)
	// p6 stopped at 50 % but Jellyfin reported it played to completion.
	if m := movie.PlayedToEnd; m.Plays != 3 || m.Finished != 2 || m.Measured != 1 || m.MeasuredFinished != 1 || !approx(m.Ratio, 2.0/3) || movie.PluginSince == nil {
		t.Errorf("movie playedToEnd = %+v", m)
	}
	if !approx(movie.Totals.CompletedRatio, 2.0/3) || movie.Funnel.Completed != 2 {
		t.Errorf("movie finished tile %v / funnel %d disagree with playedToEnd", movie.Totals.CompletedRatio, movie.Funnel.Completed)
	}
	var user struct {
		PlayedToEnd pteResp
		Profile     struct{ Completion *float64 }
	}
	v2JSON(t, db, "/api/v2/users/u1", &user)
	if u := user.PlayedToEnd; u.Plays != 6 || u.Finished != 5 || u.Measured != 2 || u.MeasuredFinished != 2 || !approx(u.Ratio, 5.0/6) {
		t.Errorf("user playedToEnd = %+v", u)
	}
	if !approx(user.Profile.Completion, 5.0/6) {
		t.Errorf("profile completion = %v, want the playedToEnd ratio", user.Profile.Completion)
	}
}

// PlayedToCompletion is ignored on music and on media shorter than 300 s.
func TestV2PluginPlayedToCompletionTrust(t *testing.T) {
	db := v2Seeded(t)
	at := func(day time.Time, h int) time.Time { return day.Add(time.Duration(h) * time.Hour) }
	if err := db.Create(&models.JFUser{Id: "u3", Name: sp("Chloé")}).Error; err != nil {
		t.Fatal(err)
	}
	rows := []models.JFPlaybackActivity{
		pluginRow("a1", "u3", "t1", "Audio", at(v2Day1, 9), 200, 200, 0, 200, 0, 0, bp(false), false),
		pluginRow("a2", "u3", "e2", "Episode", at(v2Day1, 10), 100, 100, 0, 100, 0, 0, bp(false), false),
		pluginRow("a3", "u3", "m1", "Movie", at(v2Day1, 11), 300, 600, 0, 300, 0, 0, bp(true), false),
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	var user struct{ PlayedToEnd pteResp }
	v2JSON(t, db, "/api/v2/users/u3", &user)
	if u := user.PlayedToEnd; u.Plays != 3 || u.Finished != 3 || u.Measured != 1 || u.MeasuredFinished != 1 {
		t.Errorf("playedToEnd = %+v", u)
	}
}

// An open play counts neither in the numerator nor the denominator.
func TestV2PluginAlbumFullListensIgnoreOpenPlays(t *testing.T) {
	db := v2Seeded(t)
	at := func(day time.Time, h int) time.Time { return day.Add(time.Duration(h) * time.Hour) }
	if err := db.Create(&[]models.JFPlaybackActivity{
		pluginRow("o1", "u1", "t2", "Audio", at(v2Day1, 9), 100, 200, 0, 100, 0, 0, nil, true),
	}).Error; err != nil {
		t.Fatal(err)
	}
	var album struct {
		Totals struct {
			Listens         int
			FullListenRatio *float64
		}
	}
	v2JSON(t, db, "/api/v2/albums/al1", &album)
	if album.Totals.Listens != 2 || !approx(album.Totals.FullListenRatio, 1) {
		t.Errorf("album totals = %+v", album.Totals)
	}
}

func TestV2PluginNoData(t *testing.T) {
	db := v2Seeded(t)
	var ep struct {
		Skips struct {
			Plays    int
			SkipRate *float64
		}
		PlayedToEnd struct{ Plays, Measured int }
		PluginSince *string
	}
	v2JSON(t, db, "/api/v2/episodes/e1", &ep)
	if ep.PluginSince != nil || ep.Skips.Plays != 0 || ep.Skips.SkipRate != nil || ep.PlayedToEnd.Measured != 0 || ep.PlayedToEnd.Plays != 1 {
		t.Errorf("episode without plugin data = %+v", ep)
	}
	var growth struct {
		Source      string
		PreciseFrom *string
	}
	v2JSON(t, db, "/api/v2/libraries/lib-s/growth", &growth)
	if growth.Source != "catalog" || growth.PreciseFrom != nil {
		t.Errorf("growth = %+v", growth)
	}
	var logins struct {
		CollectedSince *string
		Count          int
		Devices        []any
	}
	v2JSON(t, db, "/api/v2/users/u1/logins", &logins)
	if logins.CollectedSince != nil || logins.Count != 0 || len(logins.Devices) != 0 {
		t.Errorf("logins = %+v", logins)
	}
	var marks struct {
		CollectedSince *string
		Manual         int
	}
	v2JSON(t, db, "/api/v2/items/m1/marks", &marks)
	if marks.CollectedSince != nil || marks.Manual != 0 {
		t.Errorf("marks = %+v", marks)
	}
}

// "Nothing yet" is cached too (5 min), until the plugin sync resets the caches.
func TestV2PluginSinceNegativeCache(t *testing.T) {
	db := v2Seeded(t)
	var ep struct{ PluginSince *string }
	v2JSON(t, db, "/api/v2/episodes/e1", &ep)
	if ep.PluginSince != nil {
		t.Fatalf("pluginSince = %v", *ep.PluginSince)
	}
	seedPlugin(t, db)
	v2JSON(t, db, "/api/v2/episodes/e1", &ep)
	if ep.PluginSince != nil {
		t.Fatalf("negative result not cached: %v", *ep.PluginSince)
	}
	handler.ResetStatsCaches()
	v2JSON(t, db, "/api/v2/episodes/e1", &ep)
	if ep.PluginSince == nil || *ep.PluginSince != v2Day1.Format("2006-01-02") {
		t.Fatalf("after reset: %v", ep.PluginSince)
	}
	var n int64
	db.Raw(`SELECT COUNT(*) FROM pg_indexes WHERE indexname IN ('idx_playback_plugin_started_at', 'idx_library_events_library_at')`).Scan(&n)
	if n != 2 {
		t.Errorf("migration 016 indexes = %d, want 2", n)
	}
}

// seedLibraryEvents: a first-run snapshot (kind present, day5), then live
// events. The global cutoff is the first live event: le5 (lib-m, day3 06:00).
func seedLibraryEvents(t *testing.T, db *gorm.DB) {
	t.Helper()
	at := func(day time.Time, h int) time.Time { return day.Add(time.Duration(h) * time.Hour) }
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed events: %v", err)
		}
	}
	// Catalog: e1 created day4 08:00 (before the cutoff: catalog date counted);
	// e2 created day1 (after the cutoff: only its live event counts).
	must(db.Model(&models.JFLibraryEpisode{}).Where(`"Id" = ?`, "e1").Update("DateCreated", testutil.TS(at(v2Day4, 8))).Error)
	must(db.Model(&models.JFLibraryEpisode{}).Where(`"Id" = ?`, "e2").Update("DateCreated", testutil.TS(at(v2Day1, 8))).Error)
	// An archived (removed) episode of the series.
	must(db.Create(&models.JFLibraryEpisode{Id: "ex", EpisodeId: sp("ex"), Name: sp("Gone"), SeriesId: sp("s1"), Archived: true}).Error)
	old := at(v2Day5.AddDate(-3, 0, 0), 9)
	must(db.Create(&[]models.JFLibraryEvent{
		{Id: "sn1", Kind: "present", At: at(v2Day5, 9), ItemId: sp("e1"), ItemType: sp("Episode"), LibraryId: sp("lib-s"), DateCreated: &old},
		{Id: "sn2", Kind: "present", At: at(v2Day5, 9), ItemId: sp("e3"), ItemType: sp("Episode"), LibraryId: sp("lib-s"), DateCreated: &old},
		{Id: "le1", Kind: "added", At: at(v2Day2, 9), ItemId: sp("e2"), ItemType: sp("Episode"), LibraryId: sp("lib-s")},
		{Id: "le2", Kind: "added", At: at(v2Day2, 10), ItemId: sp("e3"), ItemType: sp("Episode")},                       // no library: catalog match
		{Id: "le3", Kind: "removed", At: at(v2Day1, 9), ItemId: sp("ex"), ItemType: sp("Episode")},                      // archived catalog row
		{Id: "le6", Kind: "removed", At: at(v2Day1, 10), ItemId: sp("ey"), ItemType: sp("Episode"), SeriesId: sp("s1")}, // via the series
		{Id: "le4", Kind: "added", At: at(v2Day2, 9), ItemId: sp("s1"), ItemType: sp("Series"), LibraryId: sp("lib-s")},
		{Id: "le5", Kind: "added", At: at(v2Day3, 6), ItemId: sp("m2"), ItemType: sp("Movie"), LibraryId: sp("lib-m")},
	}).Error)
}

func TestV2PluginLibraryGrowthFromEvents(t *testing.T) {
	db := v2Seeded(t)
	seedLibraryEvents(t, db)
	var got struct {
		Bucket, Source string
		PreciseFrom    *string
		Points         []struct {
			Date                       string
			Added, Removed, FirstViews int
			Precise                    bool
		}
	}
	v2JSON(t, db, "/api/v2/libraries/lib-s/growth", &got)
	d := func(day time.Time) string { return day.Format("2006-01-02") }
	// Cutoff = first live event of the plugin, whatever the library.
	if got.Bucket != "day" || got.Source != "events" || got.PreciseFrom == nil || *got.PreciseFrom != d(v2Day3) {
		t.Fatalf("growth = %s %s %v", got.Bucket, got.Source, got.PreciseFrom)
	}
	byDate := map[string]int{}
	for i, p := range got.Points {
		byDate[p.Date] = i
	}
	pt := func(day time.Time) struct {
		Date                       string
		Added, Removed, FirstViews int
		Precise                    bool
	} {
		return got.Points[byDate[d(day)]]
	}
	if p := pt(v2Day5); p.Added != 0 || p.Precise {
		t.Errorf("day5 (snapshot rows are not additions) = %+v", p)
	}
	if p := pt(v2Day4); p.Added != 1 || p.Precise {
		t.Errorf("day4 (catalog) = %+v", p)
	}
	if p := pt(v2Day3); p.Added != 0 || p.Precise {
		t.Errorf("day3 (cutoff inside the bucket) = %+v", p)
	}
	if p := pt(v2Day2); p.Added != 2 || p.FirstViews != 2 || !p.Precise {
		t.Errorf("day2 = %+v", p)
	}
	if p := pt(v2Day1); p.Added != 0 || p.Removed != 2 || p.FirstViews != 1 || !p.Precise {
		t.Errorf("day1 = %+v", p)
	}
	total := 0
	for _, p := range got.Points {
		total += p.Added
	}
	if total != 3 {
		t.Errorf("total added = %d, want 3", total)
	}

	var delay struct {
		CollectedSince           *string
		Added, Viewed, NotViewed int
		MedianSeconds            *float64
		Buckets                  []struct {
			Bucket string
			Count  int
		}
	}
	v2JSON(t, db, "/api/v2/libraries/lib-s/first-view-delay", &delay)
	// Live additions only: e2 (day2 09:00 → first view day2 21:05, 12 h 05)
	// and e3 (day2 10:00 → day1 21:00, 35 h); the snapshot rows are ignored.
	if delay.CollectedSince == nil || *delay.CollectedSince != d(v2Day3) || delay.Added != 2 || delay.Viewed != 2 || delay.NotViewed != 0 || !approx(delay.MedianSeconds, 84750) {
		t.Errorf("delay = %+v", delay)
	}
	want := map[string]int{"<1h": 0, "1-24h": 1, "1-7d": 1, "1-4w": 0, ">4w": 0}
	if len(delay.Buckets) != 5 {
		t.Fatalf("buckets = %+v", delay.Buckets)
	}
	for _, b := range delay.Buckets {
		if want[b.Bucket] != b.Count {
			t.Errorf("bucket %s = %d, want %d", b.Bucket, b.Count, want[b.Bucket])
		}
	}
	// Another library: the plugin was watching, nothing was added.
	v2JSON(t, db, "/api/v2/libraries/lib-mu/first-view-delay", &delay)
	if delay.CollectedSince == nil || delay.Added != 0 {
		t.Errorf("lib-mu delay = %+v", delay)
	}
}

// A first-run snapshot alone is no live data: catalog approximation.
func TestV2PluginSnapshotOnlyIsNotPrecise(t *testing.T) {
	db := v2Seeded(t)
	old := v2Day5.AddDate(-3, 0, 0)
	if err := db.Create(&models.JFLibraryEvent{Id: "sn1", Kind: "present", At: v2Day5, ItemId: sp("e1"), ItemType: sp("Episode"), LibraryId: sp("lib-s"), DateCreated: &old}).Error; err != nil {
		t.Fatal(err)
	}
	var growth struct {
		Source      string
		PreciseFrom *string
	}
	v2JSON(t, db, "/api/v2/libraries/lib-s/growth", &growth)
	if growth.Source != "catalog" || growth.PreciseFrom != nil {
		t.Errorf("growth = %+v", growth)
	}
	var delay struct {
		CollectedSince *string
		Added          int
	}
	v2JSON(t, db, "/api/v2/libraries/lib-s/first-view-delay", &delay)
	if delay.CollectedSince != nil || delay.Added != 0 {
		t.Errorf("delay = %+v", delay)
	}
}

// v2GetAs serves target with the given claims attached (as the auth middleware does).
func v2GetAs(t *testing.T, db *gorm.DB, target string, claims *auth.Claims, out interface{}) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("claims", claims); c.Next() })
	handler.NewV2Handler(db).Register(&r.RouterGroup)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("%s: status %d, body %s", target, w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestV2PluginUserLogins(t *testing.T) {
	db := v2Seeded(t)
	at := func(day time.Time, h int) time.Time { return day.Add(time.Duration(h) * time.Hour) }
	yearAgo := testutil.RecentDay(0).AddDate(-1, 0, 0)
	if err := db.Create(&[]models.JFUserLogin{
		{Id: "l1", At: at(v2Day2, 9), UserId: sp("u1"), DeviceName: sp("Chrome"), Client: sp("Web"), ApplicationVersion: sp("10.9"), Success: true},
		{Id: "l2", At: at(v2Day1, 9), UserId: sp("u1"), DeviceName: sp("Chrome"), Client: sp("Web"), ApplicationVersion: sp("10.9"), Success: true},
		{Id: "l3", At: at(v2Day1, 10), UserId: sp("u1"), DeviceName: sp("Pixel"), Client: sp("Android"), ApplicationVersion: sp("1.0"), Success: true},
		{Id: "l4", At: at(v2Day1, 11), UserName: sp("alice"), Success: false},
		{Id: "l5", At: at(v2Day1, 12), UserId: sp("u1"), Success: false},
		{Id: "l6", At: at(v2Day1, 12), UserId: sp("u2"), Success: false},
		{Id: "l7", At: at(yearAgo, 9), UserId: sp("u1"), DeviceName: sp("Old"), Success: true},
	}).Error; err != nil {
		t.Fatal(err)
	}
	type loginsResp struct {
		CollectedSince *string
		Count          int
		LastLogin      *time.Time
		Devices        []struct {
			Device, Client, Version string
			Count                   int
			LastAt                  time.Time
		}
		Failed *int
	}
	var got loginsResp
	v2JSON(t, db, "/api/v2/users/u1/logins", &got)
	if got.CollectedSince == nil || *got.CollectedSince != yearAgo.Format("2006-01-02") || got.Count != 3 || len(got.Devices) != 2 ||
		got.Devices[0].Device != "Chrome" || got.Devices[0].Count != 2 || !got.Devices[0].LastAt.Equal(at(v2Day1, 9)) ||
		got.Devices[1].Device != "Pixel" || got.LastLogin == nil || !got.LastLogin.Equal(at(v2Day1, 10)) {
		t.Errorf("logins = %+v", got)
	}
	if got.Failed != nil {
		t.Errorf("failed attempts shown to a non-admin: %d", *got.Failed)
	}
	var admin loginsResp
	v2GetAs(t, db, "/api/v2/users/u1/logins", &auth.Claims{IsAdmin: true}, &admin)
	if admin.Failed == nil || *admin.Failed != 2 {
		t.Errorf("admin failed = %v, want 2", admin.Failed)
	}
	var user loginsResp
	v2GetAs(t, db, "/api/v2/users/u1/logins", &auth.Claims{IsAdmin: false}, &user)
	if user.Failed != nil {
		t.Errorf("non-admin failed = %v", *user.Failed)
	}
	if w := v2Get(t, db, "/api/v2/users/nope/logins"); w.Code != http.StatusNotFound {
		t.Errorf("unknown user: %d", w.Code)
	}
}

func TestV2PluginItemMarks(t *testing.T) {
	db := v2Seeded(t)
	seedPlugin(t, db)
	at := func(day time.Time, h int) time.Time { return day.Add(time.Duration(h) * time.Hour) }
	if err := db.Create(&[]models.JFUserDataEvent{
		{Id: "d1", At: at(v2Day3, 9), UserId: sp("u1"), ItemId: sp("m1"), Kind: "played", Reason: sp("TogglePlayed"), Version: 1},
		{Id: "d2", At: at(v2Day2, 9), UserId: sp("u1"), ItemId: sp("m1"), Kind: "unplayed", Reason: sp("TogglePlayed"), Version: 2},
		{Id: "d3", At: at(v2Day1, 9), UserId: sp("u1"), ItemId: sp("m1"), Kind: "played", Reason: sp("PlaybackFinished"), Version: 3},
		{Id: "d4", At: at(v2Day1, 9), UserId: sp("u2"), ItemId: sp("m1"), Kind: "played", Reason: sp("TogglePlayed"), Version: 4},
		{Id: "d5", At: at(v2Day1, 9), UserId: sp("u3"), ItemId: sp("m1"), Kind: "played", Reason: sp("TogglePlayed"), Version: 5},
		{Id: "d6", At: at(v2Day1, 9), UserId: sp("u4"), ItemId: sp("m1"), Kind: "played", Reason: sp("Import"), Version: 6},
		{Id: "d7", At: at(v2Day1, 9), UserId: sp("u5"), ItemId: sp("m1"), Kind: "unplayed", Reason: sp("TogglePlayed"), Version: 7},
		{Id: "d8", At: at(v2Day1, 10), UserId: sp("u1"), ItemId: sp("m1"), Kind: "favorite", Reason: sp("UpdateUserRating"), Version: 8},
		{Id: "d9", At: at(v2Day1, 9), UserId: sp("u1"), ItemId: sp("e1"), Kind: "played", Reason: sp("TogglePlayed"), Version: 9},
	}).Error; err != nil {
		t.Fatal(err)
	}
	type marks struct {
		CollectedSince                                                 *string
		Manual, Automatic, Other, Unplayed, Watched, ManualWithoutView int
	}
	var got marks
	v2JSON(t, db, "/api/v2/items/m1/marks", &got)
	want := marks{Manual: 2, Automatic: 1, Other: 1, Unplayed: 1, Watched: 2, ManualWithoutView: 1}
	if got.CollectedSince == nil || *got.CollectedSince != v2Day3.Format("2006-01-02") {
		t.Errorf("collectedSince = %v", got.CollectedSince)
	}
	got.CollectedSince = nil
	if got != want {
		t.Errorf("marks = %+v, want %+v", got, want)
	}
	v2JSON(t, db, "/api/v2/items/m1/marks?userId=u2", &got)
	got.CollectedSince = nil
	if got != (marks{Manual: 1, Watched: 1}) {
		t.Errorf("marks u2 = %+v", got)
	}
	// Episode: u1 marked it by hand and has views; u3's open play is a view too.
	v2JSON(t, db, "/api/v2/items/e1/marks", &got)
	got.CollectedSince = nil
	if got != (marks{Manual: 1, Watched: 3}) {
		t.Errorf("marks e1 = %+v", got)
	}
}
