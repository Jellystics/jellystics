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

// seedUsersExtra inserts fixed-date rows (TZ=UTC):
//
//	weeks of 2026-01-05 (W0), 2026-01-12 (W1), 2026-01-19 (W2):
//	  a: W0 100 s, W1 100 s, W2 100 s
//	  b: W0 300 s,           W2 300 s
//	  c:           W1  50 s
//	2026-03-09 23:50 b 1200 s (crosses midnight), 2026-03-10 10:00 a 3600 s
//	movie, 2026-03-10 10:30 c 600 s episode, 2026-03-11 01:00 a (next day).
func seedUsersExtra(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("TZ", "UTC")
	db := testutil.NewDB(t)
	d := func(s string) time.Time {
		v, err := time.Parse("2006-01-02 15:04", s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	mk := func(id, user, item, mt, start string, dur int64) models.JFPlaybackActivity {
		pos := dur
		return trackedRow(id, user, item, mt, d(start), dur, dur, &pos)
	}
	rows := []models.JFPlaybackActivity{
		mk("x1", "a", "m1", "Movie", "2026-01-05 20:00", 100),
		mk("x2", "b", "m1", "Movie", "2026-01-06 20:00", 300),
		mk("x3", "a", "m1", "Movie", "2026-01-13 20:00", 100),
		mk("x4", "c", "m1", "Movie", "2026-01-14 20:00", 50),
		mk("x5", "a", "m1", "Movie", "2026-01-20 20:00", 100),
		mk("x6", "b", "m1", "Movie", "2026-01-21 20:00", 300),
		mk("y1", "b", "m2", "Movie", "2026-03-09 23:50", 1200),
		mk("y2", "a", "m2", "Movie", "2026-03-10 10:00", 3600),
		mk("y3", "c", "e1", "Episode", "2026-03-10 10:30", 600),
		mk("y4", "a", "m2", "Movie", "2026-03-11 01:00", 600),
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	return db
}

func TestV2UsersRetention(t *testing.T) {
	db := seedUsersExtra(t)
	var got struct {
		Offsets int
		Rows    []struct {
			Week     string
			Cohort   int
			Retained []*int
		}
	}
	v2JSON(t, db, "/api/v2/users/retention?from=2026-01-05&to=2026-01-22", &got)
	if got.Offsets != 5 || len(got.Rows) != 3 {
		t.Fatalf("retention = %+v", got)
	}
	want := [][]int{{2, 1, 2, -1, -1}, {2, 1, -1, -1, -1}, {2, -1, -1, -1, -1}}
	weeks := []string{"2026-01-05", "2026-01-12", "2026-01-19"}
	for i, r := range got.Rows {
		if r.Week != weeks[i] || r.Cohort != 2 || len(r.Retained) != 5 {
			t.Fatalf("row %d = %+v", i, r)
		}
		for k, v := range r.Retained {
			if (want[i][k] < 0) != (v == nil) || (v != nil && *v != want[i][k]) {
				t.Errorf("row %d offset %d = %v, want %d", i, k, v, want[i][k])
			}
		}
	}
	if w := v2Get(t, db, "/api/v2/users/retention?type=bogus"); w.Code != http.StatusBadRequest {
		t.Errorf("bad type: %d", w.Code)
	}
}

func TestV2UsersShare(t *testing.T) {
	db := seedUsersExtra(t)
	var got struct {
		Bucket  string
		Periods []string
		Users   []struct {
			ID           string
			WatchSeconds int64
			Share        float64
			Values       []float64
		}
	}
	v2JSON(t, db, "/api/v2/users/share?from=2026-01-05&to=2026-01-22", &got)
	if got.Bucket != "week" || strings.Join(got.Periods, ",") != "2026-01-05,2026-01-12,2026-01-19" || len(got.Users) != 3 {
		t.Fatalf("share = %+v", got)
	}
	near := func(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }
	b, a, c := got.Users[0], got.Users[1], got.Users[2]
	if b.ID != "b" || a.ID != "a" || c.ID != "c" || b.WatchSeconds != 600 || !near(a.Share, 300.0/950) {
		t.Fatalf("order/share = %+v", got.Users)
	}
	if !near(a.Values[0], 0.25) || !near(a.Values[1], 2.0/3) || !near(c.Values[1], 1.0/3) || c.Values[0] != 0 || !near(b.Values[2], 0.75) {
		t.Errorf("values a=%v b=%v c=%v", a.Values, b.Values, c.Values)
	}
	v2JSON(t, db, "/api/v2/users/share?from=2025-10-01&to=2026-03-01", &got)
	if got.Bucket != "month" || len(got.Periods) != 2 || got.Periods[0] != "2026-01-01" || len(got.Users[0].Values) != 2 {
		t.Errorf("monthly = %+v", got)
	}
}

func TestV2Concurrency(t *testing.T) {
	db := seedUsersExtra(t)
	var got struct {
		Date        string
		Peak        int
		PeakAt      *string
		SlotMinutes int
		Slots       []int
		Sessions    []struct {
			User       struct{ ID, Name string }
			Kind, Name string
			Start, End string
		}
	}
	v2JSON(t, db, "/api/v2/concurrency?date=2026-03-10", &got)
	if got.Date != "2026-03-10" || got.SlotMinutes != 15 || len(got.Slots) != 96 || len(got.Sessions) != 3 {
		t.Fatalf("concurrency = %+v", got)
	}
	if got.Slots[0] != 1 || got.Slots[1] != 0 || got.Slots[40] != 1 || got.Slots[42] != 2 || got.Slots[44] != 1 || got.Slots[45] != 0 {
		t.Errorf("slots = %v", got.Slots)
	}
	if got.Peak != 2 || got.PeakAt == nil || !strings.Contains(*got.PeakAt, "T10:30:00") {
		t.Errorf("peak = %d at %v", got.Peak, got.PeakAt)
	}
	first := got.Sessions[0]
	if first.User.ID != "b" || !strings.Contains(first.Start, "T00:00:00") || !strings.Contains(first.End, "T00:10:00") {
		t.Errorf("clipped session = %+v", first)
	}
	if got.Sessions[2].Kind != "series" || got.Sessions[1].Kind != "movie" {
		t.Errorf("kinds = %+v", got.Sessions)
	}
	v2JSON(t, db, "/api/v2/concurrency?date=2026-03-10&userId=a", &got)
	if len(got.Sessions) != 1 || got.Peak != 1 {
		t.Errorf("userId filter = %+v", got)
	}
	if w := v2Get(t, db, "/api/v2/concurrency?date=2026-02-30"); w.Code != http.StatusBadRequest {
		t.Errorf("bad date: %d", w.Code)
	}
}

func TestV2UsersRetentionPartialWeeks(t *testing.T) {
	db := seedUsersExtra(t)
	var got struct {
		Rows []struct {
			Week     string
			Cohort   int
			Retained []*int
		}
	}
	// Range starts on a Wednesday and ends on a Tuesday: the first and last weeks are partial.
	v2JSON(t, db, "/api/v2/users/retention?from=2026-01-07&to=2026-01-21", &got)
	if len(got.Rows) != 3 {
		t.Fatalf("rows = %+v", got.Rows)
	}
	for i, r := range got.Rows {
		if r.Cohort != 2 {
			t.Errorf("row %d (%s) cohort = %d, want 2", i, r.Week, r.Cohort)
		}
	}
	if r := got.Rows[0].Retained; r[1] == nil || *r[1] != 1 || r[2] == nil || *r[2] != 2 {
		t.Errorf("row 0 retained = %v", r)
	}
}

func TestV2ConcurrencyParisTimezone(t *testing.T) {
	t.Setenv("TZ", "Europe/Paris")
	db := testutil.NewDB(t)
	d := func(s string) time.Time {
		v, err := time.Parse("2006-01-02 15:04", s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	pos := int64(1200)
	rows := []models.JFPlaybackActivity{
		// 23:50 Paris on 2026-03-09 (22:50Z), 20 min: crosses local midnight.
		trackedRow("p1", "a", "m1", "Movie", d("2026-03-09 22:50"), 1200, 1200, &pos),
		// 2026-03-29 spring-forward: 01:30Z is 03:30 CEST, 150 minutes into the 23-hour day.
		trackedRow("p2", "b", "m1", "Movie", d("2026-03-29 01:30"), 600, 600, func() *int64 { v := int64(600); return &v }()),
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	var got struct {
		Slots    []int
		Sessions []struct{ Start, End string }
	}
	v2JSON(t, db, "/api/v2/concurrency?date=2026-03-10", &got)
	if len(got.Slots) != 96 || len(got.Sessions) != 1 || got.Slots[0] != 1 || got.Slots[1] != 0 {
		t.Fatalf("normal day = %+v", got)
	}
	if s := got.Sessions[0]; !strings.Contains(s.Start, "T00:00:00+01:00") || !strings.Contains(s.End, "T00:10:00+01:00") {
		t.Errorf("clipped at local midnight = %+v", s)
	}
	v2JSON(t, db, "/api/v2/concurrency?date=2026-03-29", &got)
	if len(got.Slots) != 92 || got.Slots[10] != 1 || got.Slots[9] != 0 || got.Slots[11] != 0 {
		t.Errorf("DST day slots = %v", got.Slots)
	}
}
