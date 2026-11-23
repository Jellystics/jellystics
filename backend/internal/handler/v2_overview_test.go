package handler_test

import (
	"encoding/csv"
	"net/http"
	"strings"
	"testing"
)

type v2Totals struct {
	WatchSeconds int64
	Views        int
	Clicks       int
	ActiveUsers  int
	Completion   *float64
}

func TestV2Overview(t *testing.T) {
	db := v2Seeded(t)

	var got struct {
		Totals   v2Totals
		Previous *v2Totals
		Daily    []struct {
			Date         string
			WatchSeconds int64
			Views        int
			ByType       struct {
				Movie, Series, Music struct {
					WatchSeconds  int64
					Views, Clicks int
				}
			}
		}
		PreviousDaily []struct{ Date string }
		TopItems      []struct {
			ID, Kind, Name, ArtID string
			Views                 int
		}
		TopUsers []struct {
			ID, Name     string
			WatchSeconds int64
		}
		ByMethod        struct{ Direct, Stream, Transcode int }
		WeekHour        [][]int
		ConcurrencyPeak []struct {
			Date string
			Peak int
		}
		OnThisDay []struct {
			ItemID, Kind, Name, Date string
			Users                    []string
		}
	}
	v2JSON(t, db, "/api/v2/overview?compare=1", &got)

	if got.Totals.Views != 6 || got.Totals.Clicks != 8 || got.Totals.WatchSeconds != 590 || got.Totals.ActiveUsers != 2 {
		t.Errorf("totals = %+v", got.Totals)
	}
	if got.Totals.Completion == nil || *got.Totals.Completion <= 0 || *got.Totals.Completion > 1 {
		t.Errorf("completion = %v", got.Totals.Completion)
	}
	if got.Previous == nil || got.Previous.Views != 0 {
		t.Errorf("previous = %+v", got.Previous)
	}
	if len(got.Daily) != 30 || len(got.PreviousDaily) != 30 {
		t.Fatalf("daily = %d, previousDaily = %d", len(got.Daily), len(got.PreviousDaily))
	}
	var sum int64
	musicViews := 0
	for _, d := range got.Daily {
		sum += d.WatchSeconds
		musicViews += d.ByType.Music.Views
	}
	if sum != 590 || musicViews != 1 {
		t.Errorf("daily sum = %d, music views = %d", sum, musicViews)
	}
	if len(got.TopItems) != 3 || got.TopItems[0].ID != "s1" || got.TopItems[0].Kind != "series" || got.TopItems[0].Views != 3 ||
		got.TopItems[1].ID != "m1" || got.TopItems[2].Kind != "album" || got.TopItems[2].ArtID != "al1" {
		t.Errorf("topItems = %+v", got.TopItems)
	}
	if len(got.TopUsers) != 2 || got.TopUsers[0].Name != "Alice" || got.TopUsers[0].WatchSeconds != 340 {
		t.Errorf("topUsers = %+v", got.TopUsers)
	}
	if got.ByMethod.Direct != 2 || got.ByMethod.Stream != 3 || got.ByMethod.Transcode != 1 {
		t.Errorf("byMethod = %+v", got.ByMethod)
	}
	cells := 0
	if len(got.WeekHour) != 7 {
		t.Fatalf("weekHour rows = %d", len(got.WeekHour))
	}
	for _, r := range got.WeekHour {
		if len(r) != 24 {
			t.Fatalf("weekHour cols = %d", len(r))
		}
		for _, v := range r {
			cells += v
		}
	}
	if cells != 6 {
		t.Errorf("weekHour total = %d", cells)
	}
	day3 := v2Day3.Format("2006-01-02")
	peak := -1
	for _, p := range got.ConcurrencyPeak {
		if p.Date == day3 {
			peak = p.Peak
		}
	}
	if peak != 2 || len(got.ConcurrencyPeak) != 30 {
		t.Errorf("peak on %s = %d (%d points)", day3, peak, len(got.ConcurrencyPeak))
	}
	if len(got.OnThisDay) != 1 || got.OnThisDay[0].ItemID != "m1" || got.OnThisDay[0].Kind != "movie" ||
		len(got.OnThisDay[0].Users) != 1 || got.OnThisDay[0].Users[0] != "Alice" {
		t.Errorf("onThisDay = %+v", got.OnThisDay)
	}
}

func TestV2OverviewFilters(t *testing.T) {
	db := v2Seeded(t)
	cases := []struct {
		query string
		views int
	}{
		{"type=movie", 2},
		{"type=music", 1},
		{"libraryId=lib-s", 3},
		{"userId=u2", 2},
	}
	for _, tc := range cases {
		var got struct {
			Totals   v2Totals
			Previous *v2Totals
		}
		v2JSON(t, db, "/api/v2/overview?"+tc.query, &got)
		if got.Totals.Views != tc.views || got.Previous != nil {
			t.Errorf("%s: totals = %+v prev = %v", tc.query, got.Totals, got.Previous)
		}
	}
	if w := v2Get(t, db, "/api/v2/overview?from=2026-02-30"); w.Code != http.StatusBadRequest {
		t.Errorf("bad from: %d", w.Code)
	}
}

type v2HistoryRow struct {
	ID        string
	StartedAt string
	User      struct{ ID, Name string }
	Item      struct {
		ID, Kind, Name       string
		ParentID, ParentName *string
		Season, Episode      *int
	}
	MediaType     string
	WatchSeconds  int64
	Completion    *float64
	Method        string
	IsView        bool
	PausedSeconds int64
}

func TestV2History(t *testing.T) {
	db := v2Seeded(t)
	type page struct {
		Rows  []v2HistoryRow
		Total int
	}
	var all page
	v2JSON(t, db, "/api/v2/history", &all)
	if all.Total != 8 || len(all.Rows) != 8 || all.Rows[0].ID != "r6" {
		t.Fatalf("history = %d rows, total %d, first %s", len(all.Rows), all.Total, all.Rows[0].ID)
	}
	var r5 *v2HistoryRow
	for i := range all.Rows {
		if all.Rows[i].ID == "r5" {
			r5 = &all.Rows[i]
		}
	}
	if r5 == nil || r5.Item.Kind != "episode" || r5.Item.Name != "Second" || r5.Item.ParentID == nil || *r5.Item.ParentID != "s1" ||
		r5.Item.Season == nil || *r5.Item.Season != 1 || *r5.Item.Episode != 2 || r5.PausedSeconds != 42 ||
		r5.Completion == nil || *r5.Completion != 0.95 || r5.Method != "DirectStream" || r5.User.Name != "Alice" {
		t.Errorf("r5 = %+v", r5)
	}
	if !strings.HasSuffix(r5.StartedAt, "Z") && !strings.Contains(r5.StartedAt, "+") {
		t.Errorf("startedAt without offset: %s", r5.StartedAt)
	}

	cases := []struct {
		query   string
		total   int
		firstID string
	}{
		{"views=1", 6, ""},
		{"sort=watchSeconds&desc=1", 8, "r7"},
		{"sort=startedAt&desc=0", 8, "r1"},
		{"q=rivages", 3, ""},
		{"q=Alice", 4, ""},
		{"method=transcode", 1, "r2"},
		{"client=Android&type=music", 2, ""},
		{"device=Chrome&userId=u1", 4, ""},
		{"pageSize=3&page=3", 8, ""},
		{"q=%25", 0, ""},
	}
	for _, tc := range cases {
		var got page
		v2JSON(t, db, "/api/v2/history?"+tc.query, &got)
		if got.Total != tc.total || (tc.firstID != "" && (len(got.Rows) == 0 || got.Rows[0].ID != tc.firstID)) {
			t.Errorf("%s: total %d rows %+v", tc.query, got.Total, got.Rows)
		}
	}
	var last page
	v2JSON(t, db, "/api/v2/history?pageSize=3&page=3", &last)
	if len(last.Rows) != 2 {
		t.Errorf("last page rows = %d", len(last.Rows))
	}
}

func TestV2HistoryCSV(t *testing.T) {
	db := v2Seeded(t)
	w := v2Get(t, db, "/api/v2/history.csv?views=1")
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("status %d type %s", w.Code, w.Header().Get("Content-Type"))
	}
	recs, err := csv.NewReader(strings.NewReader(w.Body.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 7 || recs[0][0] != "startedAt" || len(recs[1]) != len(recs[0]) {
		t.Errorf("csv = %v", recs)
	}
}

func TestV2HistoryCalendar(t *testing.T) {
	db := v2Seeded(t)
	month := v2Day2.Format("2006-01")
	var got []struct {
		Date         string
		Views        int
		WatchSeconds int64
		Items        []struct{ ID, Name, Kind, User string }
	}
	v2JSON(t, db, "/api/v2/history/calendar?month="+month, &got)
	if len(got) < 28 || got[0].Date != month+"-01" {
		t.Fatalf("calendar days = %d first %v", len(got), got)
	}
	day2 := v2Day2.Format("2006-01-02")
	found := false
	for _, d := range got {
		if d.Date == day2 {
			found = true
			// r3 (click on m1) + r4, r5 (episodes of s1): 2 views, 1 distinct title viewed
			if d.Views != 2 || d.WatchSeconds != 200 || len(d.Items) != 1 || d.Items[0].ID != "s1" || d.Items[0].User != "Alice" {
				t.Errorf("day2 = %+v", d)
			}
		}
	}
	if !found {
		t.Error("day2 missing")
	}
	if w := v2Get(t, db, "/api/v2/history/calendar?month=2026-13"); w.Code != http.StatusBadRequest {
		t.Errorf("bad month: %d", w.Code)
	}
}
