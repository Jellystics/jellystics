package handler_test

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// v5Get serves one GET through the v2 + analytics route tables.
func v5Get(t *testing.T, db *gorm.DB, target string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewV2Handler(db)
	h.Register(&r.RouterGroup)
	h.RegisterAnalytics(&r.RouterGroup)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	return w
}

func v5JSON(t *testing.T, db *gorm.DB, target string, out interface{}) {
	t.Helper()
	w := v5Get(t, db, target)
	if w.Code != http.StatusOK {
		t.Fatalf("%s: status %d, body %s", target, w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
		t.Fatalf("%s: decode: %v (%s)", target, err, w.Body.String())
	}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func ptrNear(p *float64, want float64) bool { return p != nil && near(*p, want) }

func mustExec(t *testing.T, db *gorm.DB, sql string, args ...interface{}) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func TestV2InsightsCompletionSessionsMoments(t *testing.T) {
	db := v2Seeded(t)

	var comp struct {
		Distribution []int
		ByType       struct{ Movie, Series, Music *float64 }
		Units        int
	}
	v5JSON(t, db, "/api/v2/insights/completion", &comp)
	// Units: m1 u1 1.0, m1 u2 0.3, e1 1.0, e2 0.95, e3 0.5, t1 1.0.
	want := []int{0, 0, 0, 1, 0, 1, 0, 0, 0, 4}
	for i := range want {
		if comp.Distribution[i] != want[i] {
			t.Fatalf("distribution = %v, want %v", comp.Distribution, want)
		}
	}
	if comp.Units != 6 || !ptrNear(comp.ByType.Movie, 0.5) || !ptrNear(comp.ByType.Series, 2.0/3) || !ptrNear(comp.ByType.Music, 1) {
		t.Errorf("completion = %+v", comp)
	}
	var movieOnly struct{ Units int }
	v5JSON(t, db, "/api/v2/insights/completion?type=movie", &movieOnly)
	if movieOnly.Units != 2 {
		t.Errorf("type=movie units = %d", movieOnly.Units)
	}

	var sess struct {
		Buckets       []int
		Sessions      int
		MedianSeconds *float64
	}
	v5JSON(t, db, "/api/v2/insights/sessions", &sess)
	// Views only (the two 10 s false starts are left out): 100, 30, 100, 90, 50, 200.
	if len(sess.Buckets) != 9 || sess.Buckets[0] != 2 || sess.Buckets[1] != 4 || sess.Sessions != 6 || !ptrNear(sess.MedianSeconds, 95) {
		t.Errorf("sessions = %+v", sess)
	}

	var emptySess struct {
		Sessions      int
		MedianSeconds *float64
	}
	v5JSON(t, db, "/api/v2/insights/sessions?from=2020-01-01&to=2020-01-02", &emptySess)
	if emptySess.Sessions != 0 || emptySess.MedianSeconds != nil {
		t.Errorf("empty sessions = %+v", emptySess)
	}

	var moments struct{ Movie, Series, Music []int }
	v5JSON(t, db, "/api/v2/insights/moments", &moments)
	if moments.Movie[20] != 2 || moments.Series[21] != 3 || moments.Music[8] != 1 {
		t.Errorf("moments = %+v", moments)
	}

	var pauses []struct {
		ID, Kind, Name              string
		Ratio                       float64
		PausedSeconds, WatchSeconds int64
		PauseCount                  int
	}
	mustExec(t, db, `UPDATE jf_playback_activity SET "PauseCount" = 3 WHERE "Id" = 'r5'`)
	v5JSON(t, db, "/api/v2/insights/pauses", &pauses)
	if len(pauses) != 1 || pauses[0].ID != "s1" || pauses[0].Kind != "series" || pauses[0].PausedSeconds != 42 || pauses[0].PauseCount != 3 ||
		pauses[0].WatchSeconds != 240 || !near(pauses[0].Ratio, 42.0/240) {
		t.Errorf("pauses = %+v", pauses)
	}

	var rhythm struct{ WeekHour [][]int }
	v5JSON(t, db, "/api/v2/insights/rhythm", &rhythm)
	sum := 0
	for _, row := range rhythm.WeekHour {
		for _, v := range row {
			sum += v
		}
	}
	if len(rhythm.WeekHour) != 7 || sum != 6 {
		t.Errorf("rhythm sum = %d", sum)
	}

	var genres struct {
		Months []string
		Series []struct {
			Name   string
			Values []int64
		}
		Other []int64
	}
	v5JSON(t, db, "/api/v2/insights/genres", &genres)
	if len(genres.Series) != 3 || genres.Series[0].Name != "Drama" || genres.Series[1].Name != "Rock" || genres.Series[2].Name != "Sci-Fi" {
		t.Fatalf("genres = %+v", genres)
	}
	total := func(v []int64) (s int64) {
		for _, x := range v {
			s += x
		}
		return
	}
	if total(genres.Series[0].Values) != 240 || total(genres.Series[1].Values) != 210 || total(genres.Series[2].Values) != 140 ||
		total(genres.Other) != 0 || len(genres.Months) < 2 || len(genres.Other) != len(genres.Months) {
		t.Errorf("genres = %+v", genres)
	}

	if w := v5Get(t, db, "/api/v2/insights/completion?from=2026-02-30"); w.Code != http.StatusBadRequest {
		t.Errorf("bad from: status %d", w.Code)
	}
}

func TestV2InsightsResumesRewatchesBinges(t *testing.T) {
	db := v2Seeded(t)

	var res struct {
		FromStart, Resumed int
		ByType             struct{ Movie, Series, Music *float64 }
	}
	v5JSON(t, db, "/api/v2/insights/resumes", &res)
	if res.FromStart != 0 || res.Resumed != 0 || res.ByType.Series != nil {
		t.Errorf("resumes without start positions = %+v", res)
	}
	mustExec(t, db, `UPDATE jf_playback_activity SET "PositionTicksStart" = 0 WHERE "Id" = 'r4'`)
	mustExec(t, db, `UPDATE jf_playback_activity SET "PositionTicksStart" = ? WHERE "Id" = 'r5'`, 40*secTicks)
	v5JSON(t, db, "/api/v2/insights/resumes", &res)
	if res.FromStart != 1 || res.Resumed != 1 || !ptrNear(res.ByType.Series, 0.5) || res.ByType.Movie != nil {
		t.Errorf("resumes = %+v", res)
	}

	var rew []struct {
		ID, Kind, Name string
		ParentID       *string
		Views          int
		User           struct{ ID, Name string }
	}
	v5JSON(t, db, "/api/v2/insights/rewatches", &rew)
	if len(rew) != 0 {
		t.Errorf("rewatches before = %+v", rew)
	}
	at := func(day time.Time, h int) time.Time { return day.Add(time.Duration(h) * time.Hour) }
	pos := int64(100)
	extra := []models.JFPlaybackActivity{
		trackedRow("x1", "u1", "m1", "Movie", at(v2Day1, 12), 100, 100, &pos),   // 2nd day of m1 for Alice
		trackedRow("x2", "u1", "e3", "Episode", at(v2Day2, 22), 100, 100, &pos), // 3rd episode on day2 → binge
		trackedRow("x3", "u2", "t1", "Audio", at(v2Day1, 9), 200, 200, &pos),    // 2nd listen of t1 for Bob
	}
	extra[0].UserName = sp("Alice")
	extra[1].UserName, extra[1].EpisodeId = sp("Alice"), sp("e3")
	extra[2].UserName = sp("Bob")
	if err := db.Create(&extra).Error; err != nil {
		t.Fatal(err)
	}
	v5JSON(t, db, "/api/v2/insights/rewatches", &rew)
	// m1 (Alice, days 3 and 1), e3 (Alice, days 1 and 2), t1 (Bob, 2 listens).
	if len(rew) != 3 || rew[0].Views != 2 || rew[2].Views != 2 {
		t.Fatalf("rewatches = %+v", rew)
	}
	byID := map[string]int{}
	for i, r := range rew {
		byID[r.ID] = i
	}
	m, tr := rew[byID["m1"]], rew[byID["t1"]]
	if m.Kind != "movie" || m.User.ID != "u1" || m.User.Name != "Alice" || m.ParentID != nil || m.Name != "Inception" {
		t.Errorf("movie rewatch = %+v", m)
	}
	if tr.Kind != "track" || tr.User.ID != "u2" || tr.ParentID == nil || *tr.ParentID != "al1" {
		t.Errorf("track rewatch = %+v", tr)
	}

	var binges []struct {
		ID, Name      string
		Binges, Users int
	}
	v5JSON(t, db, "/api/v2/insights/binges", &binges)
	if len(binges) != 1 || binges[0].ID != "s1" || binges[0].Name != "Les Rivages" || binges[0].Binges != 1 || binges[0].Users != 1 {
		t.Errorf("binges = %+v", binges)
	}
}

// Hours are local: with TZ=Europe/Paris the 20:00 UTC movie views move to the
// Paris hour of that instant.
func TestV2InsightsMomentsTimezone(t *testing.T) {
	db := v2Seeded(t)
	t.Setenv("TZ", "Europe/Paris")
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatal(err)
	}
	var moments struct{ Movie, Series, Music []int }
	v5JSON(t, db, "/api/v2/insights/moments", &moments)
	movieHour := v2Day3.Add(20 * time.Hour).In(paris).Hour()
	musicHour := v2Day1.Add(8 * time.Hour).In(paris).Hour()
	if movieHour == 20 || moments.Movie[movieHour] != 2 || moments.Movie[20] != 0 || moments.Music[musicHour] != 1 {
		t.Errorf("paris moments: movie hour %d → %v, music hour %d → %v", movieHour, moments.Movie, musicHour, moments.Music)
	}
}

// Month (genres) and day (rewatches) boundaries are local: 2025-01-31 23:30
// UTC is already February in Paris, and two views at 21:00 / 23:30 UTC on
// the same UTC day fall on two Paris days.
func TestV2InsightsLocalBoundaries(t *testing.T) {
	db := v2Seeded(t)
	pos := int64(100)
	rows := []models.JFPlaybackActivity{
		trackedRow("tz1", "u1", "m1", "Movie", time.Date(2025, 1, 31, 23, 30, 0, 0, time.UTC), 100, 100, &pos),
		trackedRow("tz2", "u1", "m1", "Movie", time.Date(2025, 1, 10, 21, 0, 0, 0, time.UTC), 100, 100, &pos),
		trackedRow("tz3", "u1", "m1", "Movie", time.Date(2025, 1, 10, 23, 30, 0, 0, time.UTC), 100, 100, &pos),
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	type genresJSON struct {
		Bucket string
		Months []string
		Series []struct {
			Name   string
			Values []int64
		}
	}
	type rewatchJSON []struct {
		ID    string
		Views int
	}
	const genresURL = "/api/v2/insights/genres?from=2025-01-01&to=2025-03-01"
	const rewatchURL = "/api/v2/insights/rewatches?from=2025-01-01&to=2025-01-31"

	var g genresJSON
	v5JSON(t, db, genresURL, &g)
	if g.Bucket != "month" || len(g.Months) != 2 || len(g.Series) != 1 || g.Series[0].Values[0] != 300 || g.Series[0].Values[1] != 0 {
		t.Errorf("utc genres = %+v", g)
	}
	var rw rewatchJSON
	v5JSON(t, db, rewatchURL, &rw)
	if len(rw) != 0 {
		t.Errorf("utc rewatches = %+v", rw)
	}

	t.Setenv("TZ", "Europe/Paris")
	g = genresJSON{}
	v5JSON(t, db, genresURL, &g)
	if len(g.Months) != 2 || g.Months[1] != "2025-02" || len(g.Series) != 1 || g.Series[0].Values[0] != 200 || g.Series[0].Values[1] != 100 {
		t.Errorf("paris genres = %+v", g)
	}
	rw = nil
	v5JSON(t, db, rewatchURL, &rw)
	if len(rw) != 1 || rw[0].ID != "m1" || rw[0].Views != 2 {
		t.Errorf("paris rewatches = %+v", rw)
	}
}

// Ranges over 400 days are bucketed by week, leading empty weeks trimmed.
func TestV2PlaybackMethodsLongRange(t *testing.T) {
	db := v2Seeded(t)
	var m struct {
		Bucket string
		Daily  []struct {
			Date                      string
			Direct, Stream, Transcode int
		}
	}
	v5JSON(t, db, "/api/v2/playback/methods?from=2000-01-01", &m)
	if m.Bucket != "week" || len(m.Daily) == 0 || len(m.Daily) > 60 {
		t.Fatalf("long range methods: bucket %q, %d points", m.Bucket, len(m.Daily))
	}
	if d := m.Daily[0]; d.Direct+d.Stream+d.Transcode == 0 {
		t.Errorf("leading empty week kept: %+v", d)
	}
}
