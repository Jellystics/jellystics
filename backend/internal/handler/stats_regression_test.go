package handler_test

// Regression tests for the stats audit fixes (H1-H10, M1-M7). Rows are seeded
// with the accurate-tracking columns set explicitly (see trackedRow).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func richDB(t *testing.T) (*gorm.DB, *handler.StatsFrontendHandler, *gin.Engine) {
	t.Helper()
	db := testutil.NewDB(t)
	seedRichStats(t, db)
	gin.SetMode(gin.TestMode)
	return db, handler.NewStatsFrontendHandler(db, repository.New(db)), gin.New()
}

func decode(t *testing.T, body []byte, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("decode: %v (body %s)", err, body)
	}
}

func insertRows(t *testing.T, db *gorm.DB, rows ...models.JFPlaybackActivity) {
	t.Helper()
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("insert rows: %v", err)
	}
}

// ---------------------------------------------------------------------------
// H3 — day / hour / weekday buckets and date bounds in the display TZ
// ---------------------------------------------------------------------------

func TestTimezoneBucketing_EuropeParis(t *testing.T) {
	t.Setenv("TZ", "Europe/Paris") // CET = UTC+1 in January
	db := testutil.NewDB(t)
	movie := "Movie"
	if err := db.Create(&models.JFLibraryItem{Id: "m1", Name: sp("Inception"), Type: &movie, ParentId: sp("lib"),
		RunTimeTicks: i64(7200 * secTicks), Genres: datatypes.JSON([]byte("[]"))}).Error; err != nil {
		t.Fatal(err)
	}
	utc := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	insertRows(t, db,
		// Thu 2026-01-15 23:30 Paris.
		trackedRow("before", "u1", "m1", "Movie", utc("2026-01-15T22:30:00Z"), 600, 7200, nil),
		// Fri 2026-01-16 00:30 Paris (still the 15th in UTC).
		trackedRow("after", "u1", "m1", "Movie", utc("2026-01-15T23:30:00Z"), 600, 7200, nil),
		// Sun 2026-02-01 00:30 Paris (still January in UTC).
		trackedRow("feb", "u1", "m1", "Movie", utc("2026-01-31T23:30:00Z"), 600, 7200, nil),
	)
	gin.SetMode(gin.TestMode)
	h := handler.NewStatsFrontendHandler(db, repository.New(db))
	r := gin.New()
	r.GET("/over", h.GetWatchStatisticsOverTime)
	r.GET("/hour", h.GetPopularHourOfDay)
	r.GET("/dow", h.GetPopularDayOfWeek)
	r.GET("/heat", h.GetWatchHeatmap)
	r.GET("/tl", h.GetActivityTimeline)

	var days []struct {
		Date  string `json:"date"`
		Plays int    `json:"plays"`
	}
	decode(t, getOK(t, r, "/over?from=all").Body.Bytes(), &days)
	byDate := map[string]int{}
	for _, d := range days {
		byDate[d.Date] = d.Plays
	}
	if byDate["2026-01-15"] != 1 || byDate["2026-01-16"] != 1 || byDate["2026-02-01"] != 1 {
		t.Errorf("local days = %v, want 2026-01-15:1 2026-01-16:1 2026-02-01:1", byDate)
	}

	var hours []struct {
		Hour  int `json:"hour"`
		Plays int `json:"plays"`
	}
	decode(t, getOK(t, r, "/hour?from=all").Body.Bytes(), &hours)
	byHour := map[int]int{}
	for _, h := range hours {
		byHour[h.Hour] = h.Plays
	}
	if byHour[23] != 1 || byHour[0] != 2 {
		t.Errorf("local hours = %v, want 23:1 0:2", byHour)
	}

	var dows []struct {
		Day   int `json:"day"`
		Plays int `json:"plays"`
	}
	decode(t, getOK(t, r, "/dow?from=all").Body.Bytes(), &dows)
	byDow := map[int]int{}
	for _, d := range dows {
		byDow[d.Day] = d.Plays
	}
	if byDow[4] != 1 || byDow[5] != 1 || byDow[0] != 1 { // Thu, Fri, Sun
		t.Errorf("local weekdays = %v, want 4:1 5:1 0:1", byDow)
	}

	// Date-range bounds are local calendar days, end-exclusive at next midnight.
	decode(t, getOK(t, r, "/over?from=2026-01-16&to=2026-01-16").Body.Bytes(), &days)
	if len(days) != 1 || days[0].Date != "2026-01-16" || days[0].Plays != 1 {
		t.Errorf("from=to=2026-01-16 → %+v, want only the 00:30 play", days)
	}
	decode(t, getOK(t, r, "/over?from=2026-01-15&to=2026-01-15").Body.Bytes(), &days)
	if len(days) != 1 || days[0].Date != "2026-01-15" || days[0].Plays != 1 {
		t.Errorf("from=to=2026-01-15 → %+v, want only the 23:30 play", days)
	}

	var heat struct {
		Cells []struct {
			Day, Hour, Plays int
		} `json:"cells"`
	}
	decode(t, getOK(t, r, "/heat?from=all").Body.Bytes(), &heat)
	for _, c := range heat.Cells {
		if c.Day == 5 && c.Hour == 0 && c.Plays != 1 {
			t.Errorf("heatmap Fri 00h = %d, want 1", c.Plays)
		}
		if c.Day == 4 && c.Hour == 23 && c.Plays != 1 {
			t.Errorf("heatmap Thu 23h = %d, want 1", c.Plays)
		}
	}

	var tl []struct {
		Id string `json:"Id"`
	}
	decode(t, getOK(t, r, "/tl?year=2026&month=2").Body.Bytes(), &tl)
	if len(tl) != 1 || tl[0].Id != "feb" {
		t.Errorf("timeline Feb 2026 = %+v, want only the row local to Feb 1st", tl)
	}
}

func TestTimezone_InvalidFallsBackToUTC(t *testing.T) {
	t.Setenv("TZ", "Not/AZone'; DROP TABLE x; --")
	_, h, r := richDB(t)
	r.GET("/hour", h.GetPopularHourOfDay)
	var hours []struct {
		Hour  int `json:"hour"`
		Plays int `json:"plays"`
	}
	decode(t, getOK(t, r, "/hour").Body.Bytes(), &hours)
	byHour := map[int]int{}
	for _, h := range hours {
		byHour[h.Hour] = h.Plays
	}
	if byHour[14] != 3 || byHour[9] != 2 {
		t.Errorf("hours = %v, want UTC buckets 14:3 9:2", byHour)
	}
}

// ---------------------------------------------------------------------------
// H1 — series / albums carry the plays of their episodes / tracks
// ---------------------------------------------------------------------------

func TestLibraryItems_SeriesAndAlbumPlayCounts(t *testing.T) {
	_, h, r := richDB(t)
	r.GET("/li", h.GetLibraryItems)
	r.POST("/liws", h.GetLibraryItemsWithStats)

	for _, tc := range []struct{ lib, id string }{{"lib-tv", "series-1"}, {"lib-music", "alb1"}} {
		var rows []struct {
			Id        string `json:"Id"`
			PlayCount int    `json:"PlayCount"`
		}
		decode(t, getOK(t, r, "/li?libraryId="+tc.lib).Body.Bytes(), &rows)
		if len(rows) != 1 || rows[0].Id != tc.id || rows[0].PlayCount != 1 {
			t.Errorf("getLibraryItems(%s) = %+v, want %s with 1 play", tc.lib, rows, tc.id)
		}

		var resp struct {
			Results []struct {
				Id          string `json:"Id"`
				TimesPlayed int    `json:"times_played"`
			} `json:"results"`
		}
		decode(t, postOK(t, r, "/liws", `{"libraryid":"`+tc.lib+`"}`).Body.Bytes(), &resp)
		if len(resp.Results) != 1 || resp.Results[0].TimesPlayed != 1 {
			t.Errorf("getLibraryItemsWithStats(%s) = %+v, want 1 play", tc.lib, resp.Results)
		}
	}
}

// ---------------------------------------------------------------------------
// H2 — a music play (album item + track) is counted once per library
// ---------------------------------------------------------------------------

func TestMusicLibrary_PlaysNotDoubled(t *testing.T) {
	_, h, r := richDB(t)
	r.GET("/ls", h.GetLibraryStats)
	r.GET("/libs", h.GetLibraries)
	r.POST("/mvl", h.GetMostViewedLibraries)
	r.GET("/pblot", h.GetPlaybacksByLibraryOverTime)

	var ls struct {
		TotalItems     int `json:"TotalItems"`
		TotalPlayCount int `json:"TotalPlayCount"`
		TotalWatchTime int `json:"TotalWatchTime"`
	}
	decode(t, getOK(t, r, "/ls?libraryId=lib-music").Body.Bytes(), &ls)
	if ls.TotalPlayCount != 1 || ls.TotalWatchTime != 3 || ls.TotalItems != 1 {
		t.Errorf("music library stats = %+v, want 1 play, 3 min, 1 track", ls)
	}

	var libs []struct {
		Id             string `json:"Id"`
		TotalPlayCount int    `json:"TotalPlayCount"`
	}
	decode(t, getOK(t, r, "/libs").Body.Bytes(), &libs)
	for _, l := range libs {
		if l.Id == "lib-music" && l.TotalPlayCount != 1 {
			t.Errorf("getLibraries music plays = %d, want 1", l.TotalPlayCount)
		}
	}

	var mvl []struct {
		Name  string `json:"Name"`
		Count int    `json:"Count"`
	}
	decode(t, postOK(t, r, "/mvl", `{}`).Body.Bytes(), &mvl)
	for _, l := range mvl {
		if l.Name == "Music" && l.Count != 1 {
			t.Errorf("most viewed Music = %d, want 1", l.Count)
		}
	}

	var over []struct {
		LibraryId string `json:"libraryId"`
		Count     int    `json:"count"`
	}
	decode(t, getOK(t, r, "/pblot").Body.Bytes(), &over)
	total := 0
	for _, o := range over {
		total += o.Count
	}
	if total != 5 {
		t.Errorf("plays by library over time = %d, want 5", total)
	}
}

// ---------------------------------------------------------------------------
// M5 — plays = IsView rows, clicks = all rows, watch time = all rows
// ---------------------------------------------------------------------------

func TestPlaysVersusClicks(t *testing.T) {
	db, h, r := richDB(t)
	// A 10 s click on Inception by Alice, inside the default window.
	insertRows(t, db, trackedRow("click1", "u1", "m1", "Movie", richSeedDay().Add(20*time.Hour), 10, 1200, nil))

	r.GET("/g", h.GetGlobalStats)
	r.GET("/mpi", h.GetMostPlayedItems)
	r.GET("/id", h.GetItemDetails)
	r.GET("/mau", h.GetMostActiveUsers)
	r.POST("/pms", h.GetPlaybackMethodStats)

	var g struct {
		TotalPlays     int `json:"TotalPlays"`
		TotalClicks    int `json:"TotalClicks"`
		TotalWatchTime int `json:"TotalWatchTime"`
	}
	decode(t, getOK(t, r, "/g").Body.Bytes(), &g)
	if g.TotalPlays != 5 || g.TotalClicks != 6 || g.TotalWatchTime != 48 { // 2890 s → 48 min
		t.Errorf("global = %+v, want plays 5 clicks 6 watch 48", g)
	}

	var items []struct {
		Id         string `json:"Id"`
		PlayCount  int    `json:"PlayCount"`
		ClickCount int    `json:"ClickCount"`
	}
	decode(t, getOK(t, r, "/mpi?type=Movie").Body.Bytes(), &items)
	if len(items) != 1 || items[0].PlayCount != 3 || items[0].ClickCount != 4 {
		t.Errorf("most played movies = %+v, want m1 plays 3 clicks 4", items)
	}

	var det struct {
		Stats struct {
			TotalPlays  int `json:"TotalPlays"`
			TotalClicks int `json:"TotalClicks"`
		} `json:"stats"`
		History []struct {
			IsView bool `json:"IsView"`
		} `json:"history"`
	}
	decode(t, getOK(t, r, "/id?itemId=m1").Body.Bytes(), &det)
	if det.Stats.TotalPlays != 3 || det.Stats.TotalClicks != 4 || len(det.History) != 4 {
		t.Errorf("item details = %+v (history %d), want plays 3 clicks 4 history 4", det.Stats, len(det.History))
	}

	var users []struct {
		UserId      string `json:"UserId"`
		TotalPlays  int    `json:"TotalPlays"`
		TotalClicks int    `json:"TotalClicks"`
	}
	decode(t, getOK(t, r, "/mau").Body.Bytes(), &users)
	if len(users) == 0 || users[0].UserId != "u1" || users[0].TotalPlays != 3 || users[0].TotalClicks != 4 {
		t.Errorf("most active users = %+v, want u1 plays 3 clicks 4", users)
	}

	var pms []struct {
		Name  string `json:"Name"`
		Count int    `json:"Count"`
	}
	decode(t, postOK(t, r, "/pms", `{}`).Body.Bytes(), &pms)
	for _, p := range pms {
		if p.Name == "Unknown" && p.Count != 0 {
			t.Errorf("click (no play method) counted as a play: %+v", pms)
		}
	}
}

// Other media types (and rows of unknown type) are never counted.
func TestOtherMediaTypesIgnored(t *testing.T) {
	db, h, r := richDB(t)
	other := trackedRow("trailer", "u1", "trailer-1", "Other", richSeedDay().Add(15*time.Hour), 600, 600, nil)
	insertRows(t, db, other)
	r.GET("/g", h.GetGlobalStats)
	var g struct {
		TotalPlays  int `json:"TotalPlays"`
		TotalClicks int `json:"TotalClicks"`
	}
	decode(t, getOK(t, r, "/g").Body.Bytes(), &g)
	if g.TotalPlays != 5 || g.TotalClicks != 5 {
		t.Errorf("global = %+v, 'Other' row must be ignored", g)
	}
}

// ---------------------------------------------------------------------------
// M2 — binge = >= 3 distinct episodes, same series, same user, same local day
// ---------------------------------------------------------------------------

func seedBinge(t *testing.T, db *gorm.DB) time.Time {
	t.Helper()
	series := "Series"
	if err := db.Create(&models.JFLibraryItem{Id: "show", Name: sp("The Show"), Type: &series, ParentId: sp("lib-tv"),
		Genres: datatypes.JSON([]byte(`["Drama"]`))}).Error; err != nil {
		t.Fatal(err)
	}
	var eps []models.JFLibraryEpisode
	for i := 1; i <= 4; i++ {
		eps = append(eps, models.JFLibraryEpisode{Id: fmt.Sprintf("e%d", i), Name: sp(fmt.Sprintf("Ep %d", i)),
			SeriesId: sp("show"), RunTimeTicks: i64(1800 * secTicks)})
	}
	if err := db.Create(&eps).Error; err != nil {
		t.Fatal(err)
	}
	ep := func(id, user, item string, start time.Time) models.JFPlaybackActivity {
		r := trackedRow(id, user, item, "Episode", start, 1500, 1800, nil)
		r.NowPlayingItemId = sp("show")
		r.SeriesName = sp("The Show")
		return r
	}
	day := testutil.RecentDay(4).Add(18 * time.Hour)
	insertRows(t, db,
		// u1: 3 distinct episodes on one day → 1 binge.
		ep("b1", "u1", "e1", day), ep("b2", "u1", "e2", day.Add(time.Hour)), ep("b3", "u1", "e3", day.Add(2*time.Hour)),
		// u2: the same episode 3 times → not a binge.
		ep("r1", "u2", "e1", day), ep("r2", "u2", "e1", day.Add(time.Hour)), ep("r3", "u2", "e1", day.Add(2*time.Hour)),
		// u3: 3 distinct episodes, but one is a 10 s click → not a binge.
		ep("c1", "u3", "e1", day), ep("c2", "u3", "e2", day.Add(time.Hour)),
		func() models.JFPlaybackActivity {
			r := trackedRow("c3", "u3", "e3", "Episode", day.Add(2*time.Hour), 10, 1800, nil)
			r.NowPlayingItemId = sp("show")
			return r
		}(),
	)
	return day
}

func TestBingeStats_Definition(t *testing.T) {
	db, h, r := richDB(t)
	day := seedBinge(t, db)
	r.GET("/binge", h.GetBingeStats)

	type resp struct {
		TotalBingeSessions int `json:"totalBingeSessions"`
		TopBingedSeries    []struct {
			SeriesId             string `json:"seriesId"`
			SeriesName           string `json:"seriesName"`
			TotalEpisodesWatched int    `json:"totalEpisodesWatched"`
		} `json:"topBingedSeries"`
		TopBingeUsers []struct {
			UserId string `json:"userId"`
		} `json:"topBingeUsers"`
	}
	var got resp
	decode(t, getOK(t, r, "/binge").Body.Bytes(), &got)
	if got.TotalBingeSessions != 1 {
		t.Fatalf("binge sessions = %d, want 1 (only u1)", got.TotalBingeSessions)
	}
	if len(got.TopBingedSeries) != 1 || got.TopBingedSeries[0].SeriesId != "show" || got.TopBingedSeries[0].TotalEpisodesWatched != 3 {
		t.Errorf("top series = %+v, want show with 3 episodes", got.TopBingedSeries)
	}
	if len(got.TopBingeUsers) != 1 || got.TopBingeUsers[0].UserId != "u1" {
		t.Errorf("top users = %+v, want u1", got.TopBingeUsers)
	}

	// The range `to` is honoured.
	before := day.AddDate(0, 0, -1).Format("2006-01-02")
	decode(t, getOK(t, r, "/binge?from=2000-01-01&to="+before).Body.Bytes(), &got)
	if got.TotalBingeSessions != 0 {
		t.Errorf("binge before the binge day = %d, want 0", got.TotalBingeSessions)
	}

	// userId is a bound parameter (former SQL injection).
	w := do(t, r, "/binge?userId="+strings.ReplaceAll("x' OR '1'='1", " ", "%20"))
	if w.Code != http.StatusOK {
		t.Fatalf("injection attempt status = %d", w.Code)
	}
	decode(t, w.Body.Bytes(), &got)
	if got.TotalBingeSessions != 0 {
		t.Errorf("injection attempt returned %d sessions, want 0", got.TotalBingeSessions)
	}
}

// ---------------------------------------------------------------------------
// H5 — genre radar for Series / Music uses the series / album / track genres
// ---------------------------------------------------------------------------

func TestGenreUserStats_SeriesAndMusic(t *testing.T) {
	_, h, r := richDB(t)
	r.GET("/gu", h.GetGenreUserStats)
	var resp struct {
		Results []struct {
			Genre string `json:"genre"`
			Plays int    `json:"plays"`
		} `json:"results"`
	}
	decode(t, getOK(t, r, "/gu?userid=u1&type=Series").Body.Bytes(), &resp)
	if len(resp.Results) != 1 || resp.Results[0].Genre != "Drama" || resp.Results[0].Plays != 1 {
		t.Errorf("Alice Series genres = %+v, want Drama:1", resp.Results)
	}
	decode(t, getOK(t, r, "/gu?userid=u2&type=Audio").Body.Bytes(), &resp)
	if len(resp.Results) != 1 || resp.Results[0].Genre != "Rock" || resp.Results[0].Plays != 1 {
		t.Errorf("Bob Audio genres = %+v, want Rock:1", resp.Results)
	}
}

// M3 — plays of items missing from the catalog do not land in "No Genre".
func TestGenreStats_OutOfLibraryNotNoGenre(t *testing.T) {
	db, h, r := richDB(t)
	insertRows(t, db, trackedRow("ghost", "u1", "deleted-movie", "Movie", richSeedDay().Add(16*time.Hour), 600, 6000, nil))
	r.GET("/gs", h.GetGenreStats)
	var rows []struct {
		Genre string `json:"Genre"`
	}
	decode(t, getOK(t, r, "/gs?userId=u1").Body.Bytes(), &rows)
	for _, row := range rows {
		if row.Genre == "No Genre" {
			t.Errorf("out-of-library play counted as No Genre: %+v", rows)
		}
	}
}

// ---------------------------------------------------------------------------
// M1 — item totals are not capped by the history LIMIT
// ---------------------------------------------------------------------------

func TestItemDetails_TotalsNotCapped(t *testing.T) {
	db, h, r := richDB(t)
	var rows []models.JFPlaybackActivity
	base := richSeedDay().Add(-24 * time.Hour)
	for i := 0; i < 205; i++ {
		rows = append(rows, trackedRow(fmt.Sprintf("many-%d", i), "u1", "m1", "Movie", base.Add(time.Duration(i)*time.Minute), 60, 1200, nil))
	}
	insertRows(t, db, rows...)
	r.GET("/id", h.GetItemDetails)
	var det struct {
		Stats struct {
			TotalPlays     int `json:"TotalPlays"`
			TotalWatchTime int `json:"TotalWatchTime"`
		} `json:"stats"`
		Users []struct {
			UserId    string `json:"UserId"`
			PlayCount int    `json:"PlayCount"`
		} `json:"users"`
		History []interface{} `json:"history"`
	}
	decode(t, getOK(t, r, "/id?itemId=m1").Body.Bytes(), &det)
	if det.Stats.TotalPlays != 208 {
		t.Errorf("TotalPlays = %d, want 208 (205 + 3)", det.Stats.TotalPlays)
	}
	if det.Stats.TotalWatchTime != 205+25 {
		t.Errorf("TotalWatchTime = %d, want 230 min", det.Stats.TotalWatchTime)
	}
	if len(det.History) != 200 {
		t.Errorf("history = %d, want 200 (list limit)", len(det.History))
	}
	if len(det.Users) == 0 || det.Users[0].UserId != "u1" || det.Users[0].PlayCount != 207 {
		t.Errorf("users = %+v, want u1 with 207 plays", det.Users)
	}
}

// Series detail aggregates its episodes.
func TestItemDetails_SeriesAggregatesEpisodes(t *testing.T) {
	_, h, r := richDB(t)
	r.GET("/id", h.GetItemDetails)
	var det struct {
		Stats struct {
			TotalPlays int `json:"TotalPlays"`
		} `json:"stats"`
	}
	decode(t, getOK(t, r, "/id?itemId=series-1").Body.Bytes(), &det)
	if det.Stats.TotalPlays != 1 {
		t.Errorf("series TotalPlays = %d, want 1", det.Stats.TotalPlays)
	}
	var ep struct {
		Item struct {
			Id   string `json:"Id"`
			Type string `json:"Type"`
		} `json:"item"`
	}
	decode(t, getOK(t, r, "/id?itemId=ep1").Body.Bytes(), &ep)
	if ep.Item.Id != "ep1" || ep.Item.Type != "Episode" {
		t.Errorf("episode item = %+v, want ep1 Episode", ep.Item)
	}
}

// ---------------------------------------------------------------------------
// M7 — UserDetail cards: FavoriteGenre and Plays are returned
// ---------------------------------------------------------------------------

func TestUserDetailCards(t *testing.T) {
	_, h, r := richDB(t)
	r.GET("/us", h.GetUserStats)
	r.POST("/gus", h.GetGlobalUserStats)

	var us struct {
		FavoriteGenre *string `json:"FavoriteGenre"`
		TotalClicks   int     `json:"TotalClicks"`
	}
	decode(t, getOK(t, r, "/us?userId=u2").Body.Bytes(), &us)
	// Bob: Action 300 s (Inception) vs Rock 180 s (track) → Action.
	if us.FavoriteGenre == nil || *us.FavoriteGenre != "Action" {
		t.Errorf("FavoriteGenre = %v, want Action", us.FavoriteGenre)
	}
	if us.TotalClicks != 2 {
		t.Errorf("TotalClicks = %d, want 2", us.TotalClicks)
	}

	var gus struct {
		Plays int `json:"Plays"`
	}
	decode(t, postOK(t, r, "/gus", `{"userid":"u2","hours":2400}`).Body.Bytes(), &gus)
	if gus.Plays != 2 {
		t.Errorf("Plays = %d, want 2", gus.Plays)
	}
	decode(t, postOK(t, r, "/gus", `{"userid":"u2"}`).Body.Bytes(), &gus)
	if gus.Plays != 0 {
		t.Errorf("Plays (24h) = %d, want 0 (seed is 3 days old)", gus.Plays)
	}
}

// ---------------------------------------------------------------------------
// M4 — time to watch includes episodes
// ---------------------------------------------------------------------------

func TestTimeToWatch_IncludesEpisodes(t *testing.T) {
	db, h, r := richDB(t)
	added := testutil.TS(richSeedDay().AddDate(0, 0, -10))
	if err := db.Exec(`UPDATE jf_library_episodes SET "DateCreated" = ? WHERE "Id" = 'ep1'`, added).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`UPDATE jf_library_items SET "DateCreated" = ? WHERE "Id" = 'm1'`, added).Error; err != nil {
		t.Fatal(err)
	}
	r.GET("/ttw", h.GetTimeToWatch)
	var got struct {
		SlowestItems []struct {
			Id   string `json:"id"`
			Type string `json:"type"`
		} `json:"slowestItems"`
	}
	decode(t, getOK(t, r, "/ttw").Body.Bytes(), &got)
	types := map[string]bool{}
	for _, it := range got.SlowestItems {
		types[it.Id] = true
	}
	if !types["ep1"] || !types["m1"] || len(got.SlowestItems) != 2 {
		t.Errorf("time to watch items = %+v, want m1 and ep1", got.SlowestItems)
	}
}

// ---------------------------------------------------------------------------
// H9 — library play-method chart filters on real timestamps, sorted keys
// ---------------------------------------------------------------------------

func TestLibraryPlayMethodStats_DateFilter(t *testing.T) {
	_, h, r := richDB(t)
	r.POST("/lipms", h.GetLibraryItemsPlayMethodStats)
	day := richSeedDay().Format("2006-01-02")
	var resp struct {
		Stats []struct {
			Key         string `json:"Key"`
			Transcodes  int    `json:"Transcodes"`
			DirectPlays int    `json:"DirectPlays"`
		} `json:"stats"`
	}
	decode(t, postOK(t, r, "/lipms", `{"libraryid":"lib-movies","startDate":"`+day+`","endDate":"`+day+`"}`).Body.Bytes(), &resp)
	if len(resp.Stats) == 0 {
		t.Fatalf("no buckets for the seed day")
	}
	maxT, maxD := 0, 0
	var prev time.Time
	for i, s := range resp.Stats {
		k, err := time.Parse("Jan 02, 06 15:04", s.Key)
		if err != nil {
			t.Fatalf("bad key %q", s.Key)
		}
		if i > 0 && !k.After(prev) {
			t.Errorf("keys not chronological: %s after %s", s.Key, prev)
		}
		prev = k
		maxT = max(maxT, s.Transcodes)
		maxD = max(maxD, s.DirectPlays)
	}
	if maxT != 1 || maxD != 2 {
		t.Errorf("peak transcodes/direct = %d/%d, want 1/2", maxT, maxD)
	}

	next := richSeedDay().AddDate(0, 0, 2).Format("2006-01-02")
	decode(t, postOK(t, r, "/lipms", `{"libraryid":"lib-movies","startDate":"`+next+`","endDate":"`+next+`"}`).Body.Bytes(), &resp)
	if len(resp.Stats) != 0 {
		t.Errorf("buckets outside the range = %d, want 0", len(resp.Stats))
	}
}

// ---------------------------------------------------------------------------
// Unwatched content: series watched through an episode, albums through tracks
// ---------------------------------------------------------------------------

func TestUnwatched_AlbumWatchedThroughTrack(t *testing.T) {
	_, h, r := richDB(t)
	r.GET("/uc", h.GetUnwatchedContent)
	var got struct {
		Summary struct {
			TotalItems     int `json:"totalItems"`
			UnwatchedItems int `json:"unwatchedItems"`
		} `json:"summary"`
	}
	decode(t, getOK(t, r, "/uc?libraryId=lib-music").Body.Bytes(), &got)
	if got.Summary.TotalItems != 1 || got.Summary.UnwatchedItems != 0 {
		t.Errorf("music unwatched = %+v, want 1 item, 0 unwatched", got.Summary)
	}
}

// ---------------------------------------------------------------------------
// H6/H7/H8 — Prometheus collector
// ---------------------------------------------------------------------------

func gatherMetrics(t *testing.T, db *gorm.DB) map[string][]string {
	t.Helper()
	reg := prometheus.NewRegistry()
	reg.MustRegister(handler.NewMetricsCollector(repository.New(db), db))
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err) // duplicate label sets fail here (H8)
	}
	out := map[string][]string{}
	for _, mf := range mfs {
		for _, m := range mf.GetMetric() {
			var labels []string
			for _, l := range m.GetLabel() {
				labels = append(labels, l.GetName()+"="+l.GetValue())
			}
			v := m.GetCounter().GetValue() + m.GetGauge().GetValue()
			out[mf.GetName()] = append(out[mf.GetName()], fmt.Sprintf("%s %g", strings.Join(labels, ","), v))
		}
	}
	return out
}

func hasSample(samples []string, want string) bool {
	for _, s := range samples {
		if s == want {
			return true
		}
	}
	return false
}

func TestMetrics_RenameMusicCompletionBinge(t *testing.T) {
	db, _, _ := richDB(t)
	// Alice was renamed: old rows carry the old name, jf_users the new one.
	if err := db.Exec(`UPDATE jf_playback_activity SET "UserName" = 'Alice (old)' WHERE "Id" = 'p1'`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`UPDATE jf_users SET "Name" = 'Alicia' WHERE "Id" = 'u1'`).Error; err != nil {
		t.Fatal(err)
	}
	// A different user id that shares the new name.
	insertRows(t, db, trackedRow("dup", "u9", "m1", "Movie", richSeedDay().Add(21*time.Hour), 600, 1200, nil))
	if err := db.Exec(`INSERT INTO jf_users ("Id", "Name") VALUES ('u9', 'Alicia')`).Error; err != nil {
		t.Fatal(err)
	}
	seedBinge(t, db)

	m := gatherMetrics(t, db)
	// u1: 3 rich-seed views + 3 binge episodes, under its current name only.
	if !hasSample(m["jellyfin_plays_by_user_total"], "user=Alicia 6") {
		t.Errorf("plays_by_user = %v, want user=Alicia 6 (one series per id)", m["jellyfin_plays_by_user_total"])
	}
	if !hasSample(m["jellyfin_plays_by_user_total"], "user=Alicia (2) 1") {
		t.Errorf("plays_by_user = %v, want deduped label for the second Alicia", m["jellyfin_plays_by_user_total"])
	}
	if !hasSample(m["jellyfin_plays_by_media_type_total"], "type=Audio 1") {
		t.Errorf("plays_by_media_type = %v, want type=Audio 1", m["jellyfin_plays_by_media_type_total"])
	}
	if !hasSample(m["jellyfin_binge_sessions_total"], " 1") {
		t.Errorf("binge_sessions = %v, want 1", m["jellyfin_binge_sessions_total"])
	}
	// Completion uses the played item's runtime: Inception (1200 s) watched
	// 600/600/300/600 s → abandoned; Pilot 1200/1500 s = 0.8 → abandoned
	// (threshold 0.9); Song 180/240 = 0.75 → abandoned; binge episodes
	// 1500/1800 = 0.83 → abandoned. Nothing reaches 90 %.
	if !hasSample(m["jellyfin_completed_plays_total"], " 0") {
		t.Errorf("completed = %v, want 0", m["jellyfin_completed_plays_total"])
	}
}
