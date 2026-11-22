package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
)

type recapJSON struct {
	User        struct{ ID, Name string }
	Period      string
	Granularity string
	From, To    string
	Totals      struct {
		WatchSeconds      int64
		Views, ActiveDays int
	}
	Previous                         struct{ Views int }
	TopSeries, TopMovies, TopArtists []struct {
		ID, Name     string
		WatchSeconds int64
		Views        int
	}
	Buckets []struct {
		Key               string
		WatchSeconds      int64
		Views, NewArtists int
	}
	Records struct {
		BestBucket *struct {
			Key          string
			WatchSeconds int64
		}
		Binge *struct {
			SeriesID, SeriesName, Date string
			Episodes                   int
		}
		LongestStreak int
	}
	Ranking []struct {
		ID, Kind, Name string
		ParentName     *string
		Views, Rank    int
		PreviousRank   *int
	}
}

// seedRecap adds fixed-date 2024 rows for Alice (u1) on the seedV2 catalog,
// evaluated with TZ=Europe/Paris (CET, UTC+1 in winter):
//
//	j1 m1   2024-01-31 22:30Z = Jan 31 23:30 Paris
//	j2 m1   2024-01-31 23:30Z = Feb  1 00:30 Paris (January in UTC!)
//	b1-3 e1,e2,e3 2024-02-02 20:00–20:20Z (binge)
//	k1 t1   2024-02-03 08:00Z (first listen of the artist)
//	k2 t2   2024-03-05 08:00Z
//	p1 e1   2023-06-01 12:00Z (previous year)
func seedRecap(t *testing.T) func(string, interface{}) {
	db := v2Seeded(t)
	t.Setenv("TZ", "Europe/Paris")
	utc := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	full := func(id, item, mt, at string, dur int64) models.JFPlaybackActivity {
		pos := dur
		r := trackedRow(id, "u1", item, mt, utc(at), dur, dur, &pos)
		r.UserName = sp("Alice")
		if mt == "Episode" {
			r.EpisodeId = sp(item)
		}
		return r
	}
	rows := []models.JFPlaybackActivity{
		full("j1", "m1", "Movie", "2024-01-31T22:30:00Z", 100),
		full("j2", "m1", "Movie", "2024-01-31T23:30:00Z", 100),
		full("b1", "e1", "Episode", "2024-02-02T20:00:00Z", 100),
		full("b2", "e2", "Episode", "2024-02-02T20:10:00Z", 100),
		full("b3", "e3", "Episode", "2024-02-02T20:20:00Z", 100),
		full("k1", "t1", "Audio", "2024-02-03T08:00:00Z", 200),
		full("k2", "t2", "Audio", "2024-03-05T08:00:00Z", 200),
		full("p1", "e1", "Episode", "2023-06-01T12:00:00Z", 100),
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return func(target string, out interface{}) {
		t.Helper()
		v2JSON(t, db, target, out)
	}
}

func TestV2RecapYear(t *testing.T) {
	get := seedRecap(t)
	var r recapJSON
	get("/api/v2/recap/u1/2024", &r)

	if r.User.Name != "Alice" || r.Granularity != "month" || r.From != "2024-01-01" || r.To != "2025-01-01" {
		t.Errorf("header = %+v", r)
	}
	if r.Totals.Views != 7 || r.Totals.WatchSeconds != 900 || r.Totals.ActiveDays != 5 || r.Previous.Views != 1 {
		t.Errorf("totals = %+v previous = %+v", r.Totals, r.Previous)
	}
	if len(r.Buckets) != 12 {
		t.Fatalf("buckets = %d", len(r.Buckets))
	}
	jan, feb, mar := r.Buckets[0], r.Buckets[1], r.Buckets[2]
	// j2 is in February in Paris time although it is January in UTC.
	if jan.Key != "2024-01" || jan.Views != 1 || jan.WatchSeconds != 100 ||
		feb.Views != 5 || feb.WatchSeconds != 600 || feb.NewArtists != 1 || mar.Views != 1 || mar.NewArtists != 0 {
		t.Errorf("buckets = %+v", r.Buckets[:3])
	}
	if r.Records.BestBucket == nil || r.Records.BestBucket.Key != "2024-02" || r.Records.BestBucket.WatchSeconds != 600 {
		t.Errorf("best = %+v", r.Records.BestBucket)
	}
	if r.Records.Binge == nil || r.Records.Binge.SeriesID != "s1" || r.Records.Binge.Episodes != 3 || r.Records.Binge.Date != "2024-02-02" {
		t.Errorf("binge = %+v", r.Records.Binge)
	}
	if r.Records.LongestStreak != 4 { // Jan 31, Feb 1, 2, 3
		t.Errorf("streak = %d", r.Records.LongestStreak)
	}
	if len(r.TopSeries) != 1 || r.TopSeries[0].ID != "s1" || r.TopSeries[0].WatchSeconds != 300 ||
		len(r.TopMovies) != 1 || r.TopMovies[0].Views != 2 ||
		len(r.TopArtists) != 1 || r.TopArtists[0].ID != "ar1" || r.TopArtists[0].Name != "The Band" || r.TopArtists[0].Views != 2 {
		t.Errorf("tops = %+v / %+v / %+v", r.TopSeries, r.TopMovies, r.TopArtists)
	}
	if len(r.Ranking) != 3 {
		t.Fatalf("ranking = %+v", r.Ranking)
	}
	s, al, m := r.Ranking[0], r.Ranking[1], r.Ranking[2]
	if s.ID != "s1" || s.Rank != 1 || s.PreviousRank == nil || *s.PreviousRank != 1 {
		t.Errorf("rank 1 = %+v", s)
	}
	if al.ID != "al1" || al.Kind != "album" || al.Rank != 2 || al.PreviousRank != nil || al.ParentName == nil || *al.ParentName != "The Band" {
		t.Errorf("rank 2 = %+v", al)
	}
	if m.ID != "m1" || m.Rank != 3 || m.PreviousRank != nil {
		t.Errorf("rank 3 = %+v", m)
	}
}

func TestV2RecapMonthTimezone(t *testing.T) {
	get := seedRecap(t)
	var jan, feb recapJSON
	get("/api/v2/recap/u1/2024-01", &jan)
	get("/api/v2/recap/u1/2024-02", &feb)
	if jan.Granularity != "day" || len(jan.Buckets) != 31 || jan.Totals.Views != 1 || jan.Buckets[30].Views != 1 || jan.To != "2024-02-01" {
		t.Errorf("january = %+v", jan.Totals)
	}
	if len(feb.Buckets) != 29 || feb.Totals.Views != 5 || feb.Buckets[0].Views != 1 || feb.Records.LongestStreak != 3 {
		t.Errorf("february = %+v streak %d", feb.Totals, feb.Records.LongestStreak)
	}
}

func TestV2RecapErrors(t *testing.T) {
	db := v2Seeded(t)
	for target, want := range map[string]int{
		"/api/v2/recap/u1/2024-13":  http.StatusBadRequest,
		"/api/v2/recap/u1/24":       http.StatusBadRequest,
		"/api/v2/recap/u1/2024-1":   http.StatusBadRequest,
		"/api/v2/recap/nobody/2024": http.StatusNotFound,
		"/api/v2/recap/u2/2024":     http.StatusOK,
	} {
		if w := v2Get(t, db, target); w.Code != want {
			t.Errorf("%s: status %d, want %d", target, w.Code, want)
		}
	}
}
