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

// pluginRow is a Jellystics-plugin play of an episode / movie.
func pluginRow(id, user, item, mt string, start time.Time, dur, posStart, posEnd int64, skipped, seeks int, ptc *bool, open bool) models.JFPlaybackActivity {
	r := trackedRow(id, user, item, mt, start, dur, 100, &posEnd)
	r.Source = "jellystics-plugin"
	r.PositionTicksStart = i64(posStart * secTicks)
	r.SkippedSeconds, r.SeekCount = &skipped, &seeks
	r.PlayedToCompletion = ptc
	r.PluginOpen = bp(open)
	if mt == "Episode" {
		r.NowPlayingItemId, r.EpisodeId, r.SeriesName = sp("s1"), sp(item), sp("Les Rivages")
	}
	return r
}

// seedPlugin adds plugin data on top of seedV2 (all at fixed times of the
// seeded days, UTC):
//
//	p1 u1 e1 day1 10:00 80 s  start 0  skipped 10 s (intro)   PTC true
//	p2 u2 e1 day1 11:00 60 s  start 0  skipped 40 s (> 15 %)  PTC false, stop 100 %
//	p3 u2 e1 day1 12:00 50 s  start 0  no skip                PTC null, stop 95 %
//	p4 u3 e1 day1 13:00 40 s  OPEN, skipped 5 s, stop 40 %    (provisional)
//	p5 u1 e1 day1 14:00 10 s  click, skipped 5 s
//	p6 u1 m1 day1 15:00 100 s PTC true
func seedPlugin(t *testing.T, db *gorm.DB) {
	t.Helper()
	at := func(day time.Time, h int) time.Time { return day.Add(time.Duration(h) * time.Hour) }
	rows := []models.JFPlaybackActivity{
		pluginRow("p1", "u1", "e1", "Episode", at(v2Day1, 10), 80, 0, 100, 10, 1, bp(true), false),
		pluginRow("p2", "u2", "e1", "Episode", at(v2Day1, 11), 60, 0, 100, 40, 2, bp(false), false),
		pluginRow("p3", "u2", "e1", "Episode", at(v2Day1, 12), 50, 0, 95, 0, 0, nil, false),
		pluginRow("p4", "u3", "e1", "Episode", at(v2Day1, 13), 40, 0, 40, 5, 1, nil, true),
		pluginRow("p5", "u1", "e1", "Episode", at(v2Day1, 14), 10, 0, 10, 5, 1, nil, false),
		pluginRow("p6", "u1", "m1", "Movie", at(v2Day1, 15), 100, 0, 100, 0, 0, bp(true), false),
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed plugin: %v", err)
	}
}

func TestV2PluginEpisodeSkipsAndPlayedToEnd(t *testing.T) {
	db := v2Seeded(t)
	seedPlugin(t, db)
	var got struct {
		Totals struct {
			Views         int
			AvgCompletion *float64
		}
		Skips struct {
			Plays, Skipped, Intro                  int
			SkipRate, IntroRate, AvgSkippedSeconds *float64
		}
		PlayedToEnd struct {
			Plays, Finished, Measured, MeasuredFinished int
			Ratio                                       *float64
		}
		PluginSince *string
		Viewers     []struct {
			User       struct{ ID string }
			Completion *float64
		}
	}
	v2JSON(t, db, "/api/v2/episodes/e1", &got)
	s := got.Skips
	if s.Plays != 3 || s.Skipped != 2 || s.Intro != 1 || !approx(s.SkipRate, 2.0/3) || !approx(s.IntroRate, 1.0/3) || !approx(s.AvgSkippedSeconds, 25) {
		t.Errorf("skips = %+v", s)
	}
	pe := got.PlayedToEnd
	if pe.Plays != 4 || pe.Finished != 3 || pe.Measured != 2 || pe.MeasuredFinished != 1 || !approx(pe.Ratio, 0.75) {
		t.Errorf("playedToEnd = %+v", pe)
	}
	if got.PluginSince == nil || *got.PluginSince != v2Day1.Format("2006-01-02") {
		t.Errorf("pluginSince = %v", got.PluginSince)
	}
	// The open play p4 is a view but stays out of completion (else 0.85).
	if got.Totals.Views != 5 || !approx(got.Totals.AvgCompletion, 1) {
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
	if series.Skips.Plays != 3 || !approx(series.Skips.IntroRate, 1.0/3) || series.PluginSince == nil {
		t.Errorf("series skips = %+v", series.Skips)
	}
	e1, e2 := series.Seasons[0].Episodes[0], series.Seasons[0].Episodes[1]
	if e1.ID != "e1" || e1.SkipPlays != 3 || !approx(e1.IntroRate, 1.0/3) || !approx(e1.SkipRate, 2.0/3) || !approx(e1.AvgSkippedSeconds, 25) {
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
	if ins.CollectedSince == nil || ins.Totals.Plays != 3 || ins.Totals.Intro != 1 || len(ins.Series) != 1 ||
		ins.Series[0].ID != "s1" || ins.Series[0].Name != "Les Rivages" || ins.Series[0].Plays != 3 || !approx(ins.Series[0].IntroRate, 1.0/3) {
		t.Errorf("insights skips = %+v", ins)
	}
	// Under the minimum of measured plays, a series is not ranked.
	v2JSON(t, db, "/api/v2/insights/skips?userId=u2", &ins)
	if ins.Totals.Plays != 2 || len(ins.Series) != 0 {
		t.Errorf("insights skips u2 = %+v", ins)
	}
}

func TestV2PluginMovieAndUserPlayedToEnd(t *testing.T) {
	db := v2Seeded(t)
	seedPlugin(t, db)
	type pte struct {
		Plays, Finished, Measured, MeasuredFinished int
		Ratio                                       *float64
	}
	var movie struct {
		PlayedToEnd pte
		PluginSince *string
	}
	v2JSON(t, db, "/api/v2/movies/m1", &movie)
	if m := movie.PlayedToEnd; m.Plays != 3 || m.Finished != 2 || m.Measured != 1 || m.MeasuredFinished != 1 || !approx(m.Ratio, 2.0/3) || movie.PluginSince == nil {
		t.Errorf("movie playedToEnd = %+v", m)
	}
	var user struct {
		PlayedToEnd pte
		PluginSince *string
	}
	v2JSON(t, db, "/api/v2/users/u1", &user)
	if u := user.PlayedToEnd; u.Plays != 6 || u.Finished != 5 || u.Measured != 2 || u.MeasuredFinished != 2 || !approx(u.Ratio, 5.0/6) {
		t.Errorf("user playedToEnd = %+v", u)
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

// seedLibraryEvents: real additions of lib-s (cutoff = le1, day2 09:00).
func seedLibraryEvents(t *testing.T, db *gorm.DB) {
	t.Helper()
	at := func(day time.Time, h int) time.Time { return day.Add(time.Duration(h) * time.Hour) }
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed events: %v", err)
		}
	}
	// Catalog: e1 added day3 08:00 (before the cutoff: catalog date counted);
	// e2 created day1 (after the cutoff: its event is the source).
	must(db.Model(&models.JFLibraryEpisode{}).Where(`"Id" = ?`, "e1").Update("DateCreated", testutil.TS(at(v2Day3, 8))).Error)
	must(db.Model(&models.JFLibraryEpisode{}).Where(`"Id" = ?`, "e2").Update("DateCreated", testutil.TS(at(v2Day1, 8))).Error)
	must(db.Create(&[]models.JFLibraryEvent{
		{Id: "le1", Kind: "added", At: at(v2Day2, 9), ItemId: sp("e2"), ItemType: sp("Episode"), LibraryId: sp("lib-s")},
		{Id: "le2", Kind: "added", At: at(v2Day2, 10), ItemId: sp("e3"), ItemType: sp("Episode")}, // no library: catalog match
		{Id: "le3", Kind: "removed", At: at(v2Day1, 9), ItemId: sp("ex"), ItemType: sp("Episode"), LibraryId: sp("lib-s")},
		{Id: "le4", Kind: "added", At: at(v2Day2, 9), ItemId: sp("s1"), ItemType: sp("Series"), LibraryId: sp("lib-s")},
		{Id: "le5", Kind: "added", At: at(v2Day1, 9), ItemId: sp("m2"), ItemType: sp("Movie"), LibraryId: sp("lib-m")},
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
	if got.Bucket != "day" || got.Source != "events" || got.PreciseFrom == nil || *got.PreciseFrom != d(v2Day2) {
		t.Fatalf("growth = %s %s %v", got.Bucket, got.Source, got.PreciseFrom)
	}
	byDate := map[string]int{}
	for i, p := range got.Points {
		byDate[p.Date] = i
	}
	p3, p2, p1 := got.Points[byDate[d(v2Day3)]], got.Points[byDate[d(v2Day2)]], got.Points[byDate[d(v2Day1)]]
	if p3.Added != 1 || p3.Removed != 0 || p3.Precise {
		t.Errorf("day3 = %+v", p3)
	}
	if p2.Added != 2 || p2.FirstViews != 2 || p2.Precise {
		t.Errorf("day2 = %+v", p2)
	}
	if p1.Added != 0 || p1.Removed != 1 || p1.FirstViews != 1 || !p1.Precise {
		t.Errorf("day1 = %+v", p1)
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
	// e2: added day2 09:00, first view day2 21:05 (12 h 05); e3: added day2
	// 10:00, first view day1 21:00 (35 h).
	if delay.CollectedSince == nil || *delay.CollectedSince != d(v2Day2) || delay.Added != 2 || delay.Viewed != 2 || delay.NotViewed != 0 || !approx(delay.MedianSeconds, 84750) {
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
	// Another library without events: no collected data.
	v2JSON(t, db, "/api/v2/libraries/lib-mu/first-view-delay", &delay)
	if delay.CollectedSince != nil || delay.Added != 0 {
		t.Errorf("lib-mu delay = %+v", delay)
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
