package handler_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"gorm.io/gorm"
)

func mustUTC(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// Europe/Paris, around the 2026-03-29 spring-forward day (23 h long):
//
//	p3 u1 m1 2026-03-28 22:30Z = 03-28 23:30 CET   60/100 s
//	p1 u1 m1 2026-03-28 23:30Z = 03-29 00:30 CET  100/100 s  (next local day)
//	p2 u1 m2 2026-03-29 01:30Z = 03-29 03:30 CEST  40/100 s  (after the DST jump)
//	p4 u2 m1 2026-03-27 22:50Z = 03-27 23:50 CET   1 h, runs across the range start
//	p5 u2 m2 2026-03-27 23:10Z = 03-28 00:10 CET   60/100 s  (overlaps p4)
func seedParis(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("TZ", "Europe/Paris")
	db := testutil.NewDB(t)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	must(db.Create(&[]models.JFUser{{Id: "u1", Name: sp("Alice")}, {Id: "u2", Name: sp("Bob")}}).Error)
	must(db.Create(&[]models.JFLibraryItem{
		{Id: "m1", Name: sp("One"), Type: sp("Movie"), RunTimeTicks: i64(100 * secTicks), Genres: js(`[]`)},
		{Id: "m2", Name: sp("Two"), Type: sp("Movie"), RunTimeTicks: i64(100 * secTicks), Genres: js(`[]`)},
	}).Error)
	pos := func(v int64) *int64 { return &v }
	rows := []models.JFPlaybackActivity{
		trackedRow("p3", "u1", "m1", "Movie", mustUTC(t, "2026-03-28T22:30:00Z"), 60, 100, pos(60)),
		trackedRow("p1", "u1", "m1", "Movie", mustUTC(t, "2026-03-28T23:30:00Z"), 100, 100, pos(100)),
		trackedRow("p2", "u1", "m2", "Movie", mustUTC(t, "2026-03-29T01:30:00Z"), 40, 100, pos(40)),
		trackedRow("p4", "u2", "m1", "Movie", mustUTC(t, "2026-03-27T22:50:00Z"), 3600, 100, pos(100)),
		trackedRow("p5", "u2", "m2", "Movie", mustUTC(t, "2026-03-27T23:10:00Z"), 60, 100, pos(60)),
	}
	must(db.Create(&rows).Error)
	return db
}

func TestV2LocalDaysParisDST(t *testing.T) {
	db := seedParis(t)
	var got struct {
		Bucket string
		Totals v2Totals
		Daily  []struct {
			Date  string
			Views int
		}
		WeekHour        [][]int
		ConcurrencyPeak []struct {
			Date string
			Peak int
		}
	}
	v2JSON(t, db, "/api/v2/overview?from=2026-03-28&to=2026-03-31", &got)
	if got.Bucket != "day" || len(got.Daily) != 3 || got.Daily[0].Date != "2026-03-28" || got.Daily[0].Views != 2 ||
		got.Daily[1].Date != "2026-03-29" || got.Daily[1].Views != 2 || got.Daily[2].Views != 0 {
		t.Errorf("daily = %+v", got.Daily)
	}
	// units: (u1,m1,03-28) 0.6, (u1,m1,03-29) 1.0, (u1,m2,03-29) 0.4, (u2,m2,03-28) 0.6
	if got.Totals.Views != 4 || got.Totals.Clicks != 4 || !approx(got.Totals.Completion, 0.65) {
		t.Errorf("totals = %+v completion %v", got.Totals, *got.Totals.Completion)
	}
	// Saturday = 5, Sunday = 6 (Monday first).
	if got.WeekHour[5][23] != 1 || got.WeekHour[5][0] != 1 || got.WeekHour[6][0] != 1 || got.WeekHour[6][3] != 1 || got.WeekHour[6][2] != 0 {
		t.Errorf("weekHour sat = %v sun = %v", got.WeekHour[5], got.WeekHour[6])
	}
	if got.ConcurrencyPeak[0].Date != "2026-03-28" || got.ConcurrencyPeak[0].Peak != 2 || got.ConcurrencyPeak[1].Peak != 1 {
		t.Errorf("peak = %+v", got.ConcurrencyPeak)
	}

	var cal []struct {
		Date  string
		Views int
	}
	// from/to are ignored by the calendar, even when invalid.
	v2JSON(t, db, "/api/v2/history/calendar?month=2026-03&from=bad&to=worse", &cal)
	if len(cal) != 31 || cal[27].Date != "2026-03-28" || cal[27].Views != 2 || cal[28].Views != 2 || cal[26].Views != 1 {
		t.Errorf("calendar 27..29 = %+v", cal[26:29])
	}

	// Ranges over 400 days are bucketed by ISO week (labelled by Monday).
	v2JSON(t, db, "/api/v2/overview?from=2024-01-01&to=2026-03-31", &got)
	if got.Bucket != "week" || got.Daily[0].Date != "2024-01-01" || got.Daily[len(got.Daily)-1].Date != "2026-03-30" {
		t.Fatalf("weekly bucket %s first %v last %v", got.Bucket, got.Daily[0], got.Daily[len(got.Daily)-1])
	}
	found := false
	for i, d := range got.Daily {
		if d.Date == "2026-03-23" {
			found = true
			// p4 starts on Friday 03-27 local: same ISO week as 03-28 and 03-29.
			if d.Views != 5 || got.ConcurrencyPeak[i].Date != d.Date || got.ConcurrencyPeak[i].Peak != 2 {
				t.Errorf("week 2026-03-23 = %+v peak %+v", d, got.ConcurrencyPeak[i])
			}
		}
	}
	if !found {
		t.Error("week 2026-03-23 missing")
	}
}

func TestV2SpecialsAndMisc(t *testing.T) {
	db := v2Seeded(t)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	// An unwatched special (season 0) must not drive retention nor count as a season.
	must(db.Create(&models.JFLibraryEpisode{Id: "e0", Name: sp("Making of"), SeriesId: sp("s1"), ParentIndexNumber: ip(0), IndexNumber: ip(1), RunTimeTicks: i64(100 * secTicks)}).Error)
	var series struct {
		Item    struct{ Seasons, Episodes int }
		Totals  struct{ Retention *float64 }
		Seasons []struct {
			Number   int
			Special  bool
			Episodes []struct{ ID string }
		}
	}
	v2JSON(t, db, "/api/v2/series/s1", &series)
	last := series.Seasons[len(series.Seasons)-1]
	if series.Item.Seasons != 2 || len(series.Seasons) != 3 || !last.Special || last.Number != 0 || last.Episodes[0].ID != "e0" ||
		series.Seasons[0].Special || !approx(series.Totals.Retention, 1) {
		t.Errorf("series = %+v", series)
	}
	var lib struct {
		Items struct {
			Rows []struct {
				ID        string
				Retention *float64
			}
		}
	}
	v2JSON(t, db, "/api/v2/libraries/lib-s", &lib)
	if !approx(lib.Items.Rows[0].Retention, 1) {
		t.Errorf("library retention = %+v", lib.Items.Rows)
	}
	var ep struct {
		Item struct{ PrevID, NextID *string }
	}
	v2JSON(t, db, "/api/v2/episodes/e3", &ep)
	if ep.Item.NextID == nil || *ep.Item.NextID != "e0" {
		t.Errorf("e3 next = %v (specials come last)", ep.Item.NextID)
	}

	// Ascending library sort with desc=0.
	var libm struct {
		Items struct{ Rows []struct{ ID string } }
	}
	v2JSON(t, db, "/api/v2/libraries/lib-m?sort=views&desc=0", &libm)
	if libm.Items.Rows[0].ID != "m2" {
		t.Errorf("asc sort = %+v", libm.Items.Rows)
	}

	// Album: an archived track does not block a full-album listen.
	must(db.Create(&models.JFMusicTrack{Id: "t3", Name: sp("Hidden"), AlbumId: sp("al1"), ArtistId: sp("ar1"), Archived: true, RunTimeTicks: i64(200 * secTicks), Genres: js(`[]`)}).Error)
	full := trackedRow("r10", "u2", "t2", "Audio", v2Day1.Add(8*time.Hour+10*time.Minute), 200, 200, i64(200))
	full.UserName, full.NowPlayingItemId = sp("Bob"), sp("al1")
	must(db.Create(&full).Error)
	var album struct {
		Item   struct{ Tracks int }
		Totals struct{ FullAlbumListens int }
	}
	v2JSON(t, db, "/api/v2/albums/al1", &album)
	if album.Item.Tracks != 2 || album.Totals.FullAlbumListens != 1 {
		t.Errorf("album = %+v", album)
	}

	// Stop position falls back to start + watched when the end is unknown.
	sp2 := trackedRow("r11", "u1", "m2", "Movie", v2Day1.Add(12*time.Hour), 40, 100, nil)
	sp2.PositionTicksStart = i64(50 * secTicks)
	must(db.Create(&sp2).Error)
	var movie struct{ StopPositions []int }
	v2JSON(t, db, "/api/v2/movies/m2", &movie)
	if movie.StopPositions[21] != 1 {
		t.Errorf("stops = %v", movie.StopPositions)
	}

	// CSV: formula-like cells are neutralised; the filename uses the last included day.
	must(db.Model(&models.JFUser{}).Where(`"Id" = ?`, "u1").Update("Name", "=Alice").Error)
	w := v2Get(t, db, "/api/v2/history.csv?userId=u1&from="+v2Day3.Format("2006-01-02")+"&to="+v2Day1.AddDate(0, 0, 1).Format("2006-01-02"))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), ",'=Alice,") || strings.Contains(w.Body.String(), ",=Alice,") {
		t.Errorf("csv = %d %s", w.Code, w.Body.String())
	}
	wantName := `filename="history_` + v2Day3.Format("2006-01-02") + `_` + v2Day1.Format("2006-01-02") + `.csv"`
	if !strings.Contains(w.Header().Get("Content-Disposition"), wantName) {
		t.Errorf("disposition = %s, want %s", w.Header().Get("Content-Disposition"), wantName)
	}
}
