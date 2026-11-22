package handler_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/testutil"
)

type scatterItem struct {
	ID         string
	Kind       string
	Name       string
	ParentID   *string
	ParentName *string
}

type scatterPlays struct {
	Total  int
	Limit  int
	Capped bool
	Points []struct {
		ID           string
		StartedAt    string
		WatchSeconds int64
		Type         string
		Item         scatterItem
		User         struct{ ID, Name string }
	}
}

type scatterLibraries struct {
	Bucket    string
	Libraries []struct {
		ID    string
		Name  string
		Views int
	}
	Points []struct {
		Date      string
		LibraryID string
		Views     int
	}
}

func playIDs(p scatterPlays) []string {
	ids := make([]string, len(p.Points))
	for i, pt := range p.Points {
		ids[i] = pt.ID
	}
	return ids
}

func TestV2OverviewPlays(t *testing.T) {
	db := v2Seeded(t)
	var got scatterPlays
	v2JSON(t, db, "/api/v2/overview/plays", &got)
	// Newest first; r9 (one year ago) is outside the default 30 days.
	if want := []string{"r6", "r8", "r7", "r5", "r4", "r3", "r2", "r1"}; !reflect.DeepEqual(playIDs(got), want) {
		t.Fatalf("ids = %v, want %v", playIDs(got), want)
	}
	if got.Total != 8 || got.Limit != 5000 || got.Capped {
		t.Errorf("total %d limit %d capped %v", got.Total, got.Limit, got.Capped)
	}
	ep, tr, mv := got.Points[0], got.Points[2], got.Points[7]
	if ep.Type != "series" || ep.WatchSeconds != 50 || ep.Item.Kind != "episode" || ep.Item.ID != "e3" || ep.Item.Name != "Return" ||
		ep.Item.ParentID == nil || *ep.Item.ParentID != "s1" || ep.Item.ParentName == nil || *ep.Item.ParentName != "Les Rivages" ||
		ep.User.ID != "u1" || ep.User.Name != "Alice" {
		t.Errorf("episode point = %+v", ep)
	}
	if want := v2Day1.Add(21 * time.Hour).Format(time.RFC3339); ep.StartedAt != want {
		t.Errorf("startedAt = %s, want %s", ep.StartedAt, want)
	}
	if tr.Type != "music" || tr.Item.Kind != "track" || tr.Item.Name != "Opening" || tr.Item.ParentID == nil || *tr.Item.ParentID != "al1" ||
		*tr.Item.ParentName != "Blue Album" || tr.WatchSeconds != 200 || tr.User.Name != "Bob" {
		t.Errorf("track point = %+v", tr)
	}
	if mv.Type != "movie" || mv.Item.Kind != "movie" || mv.Item.Name != "Inception" || mv.Item.ParentID != nil || mv.WatchSeconds != 100 {
		t.Errorf("movie point = %+v", mv)
	}

	// Filters of the v2 contract.
	v2JSON(t, db, "/api/v2/overview/plays?type=music", &got)
	if want := []string{"r8", "r7"}; !reflect.DeepEqual(playIDs(got), want) || got.Total != 2 {
		t.Errorf("type=music ids = %v total %d", playIDs(got), got.Total)
	}
	v2JSON(t, db, "/api/v2/overview/plays?userId=u1", &got)
	if want := []string{"r6", "r5", "r4", "r1"}; !reflect.DeepEqual(playIDs(got), want) {
		t.Errorf("userId=u1 ids = %v", playIDs(got))
	}
	v2JSON(t, db, "/api/v2/overview/plays?libraryId=lib-m", &got)
	if want := []string{"r3", "r2", "r1"}; !reflect.DeepEqual(playIDs(got), want) {
		t.Errorf("libraryId=lib-m ids = %v", playIDs(got))
	}
	// `to` is exclusive: day2 only.
	v2JSON(t, db, fmt.Sprintf("/api/v2/overview/plays?from=%s&to=%s", v2Day2.Format("2006-01-02"), v2Day1.Format("2006-01-02")), &got)
	if want := []string{"r5", "r4", "r3"}; !reflect.DeepEqual(playIDs(got), want) {
		t.Errorf("day2 ids = %v", playIDs(got))
	}
	if w := v2Get(t, db, "/api/v2/overview/plays?type=book"); w.Code != 400 {
		t.Errorf("bad type: status %d", w.Code)
	}
}

func TestV2OverviewPlaysCapAndNegative(t *testing.T) {
	t.Setenv("TZ", "UTC")
	db := testutil.NewDB(t)
	base := mustUTC(t, "2026-03-01T00:00:00Z")
	rows := make([]models.JFPlaybackActivity, 0, 5003)
	for i := 0; i < 5003; i++ {
		rows = append(rows, trackedRow(fmt.Sprintf("p%04d", i), "u1", "m1", "Movie", base.Add(time.Duration(i)*time.Minute), 60, 100, nil))
	}
	rows[5002].PlaybackDuration = i64(-50) // newest row: a corrupt negative duration
	if err := db.CreateInBatches(&rows, 500).Error; err != nil {
		t.Fatal(err)
	}
	// Outside the period: never counted.
	out := trackedRow("late", "u1", "m1", "Movie", mustUTC(t, "2026-03-10T00:00:00Z"), 60, 100, nil)
	if err := db.Create(&out).Error; err != nil {
		t.Fatal(err)
	}
	var got scatterPlays
	v2JSON(t, db, "/api/v2/overview/plays?from=2026-03-01&to=2026-03-10", &got)
	if got.Total != 5003 || !got.Capped || len(got.Points) != 5000 {
		t.Fatalf("total %d capped %v points %d", got.Total, got.Capped, len(got.Points))
	}
	if got.Points[0].ID != "p5002" || got.Points[0].WatchSeconds != 0 || got.Points[1].WatchSeconds != 60 || got.Points[4999].ID != "p0003" {
		t.Errorf("first %+v second %+v last %s", got.Points[0], got.Points[1], got.Points[4999].ID)
	}
}

func TestV2OverviewLibrariesDaily(t *testing.T) {
	db := v2Seeded(t)
	d := func(t time.Time) string { return t.Format("2006-01-02") }
	var got scatterLibraries
	v2JSON(t, db, "/api/v2/overview/libraries-daily", &got)
	if got.Bucket != "day" {
		t.Errorf("bucket = %s", got.Bucket)
	}
	libs := fmt.Sprint(got.Libraries)
	if want := "[{lib-s Séries 3} {lib-m Films 2} {lib-mu Musique 1}]"; libs != want {
		t.Errorf("libraries = %s, want %s", libs, want)
	}
	// Views only (r3 and r8 are clicks), one point per non-empty (day, library).
	pts := fmt.Sprint(got.Points)
	want := fmt.Sprintf("[{%s lib-m 2} {%s lib-s 2} {%s lib-mu 1} {%s lib-s 1}]", d(v2Day3), d(v2Day2), d(v2Day1), d(v2Day1))
	if pts != want {
		t.Errorf("points = %s, want %s", pts, want)
	}

	v2JSON(t, db, "/api/v2/overview/libraries-daily?libraryId=lib-m", &got)
	if fmt.Sprint(got.Libraries) != "[{lib-m Films 2}]" || fmt.Sprint(got.Points) != fmt.Sprintf("[{%s lib-m 2}]", d(v2Day3)) {
		t.Errorf("libraryId=lib-m: %v %v", got.Libraries, got.Points)
	}
	v2JSON(t, db, "/api/v2/overview/libraries-daily?userId=u2&type=music", &got)
	if fmt.Sprint(got.Libraries) != "[{lib-mu Musique 1}]" || len(got.Points) != 1 {
		t.Errorf("u2 music: %v %v", got.Libraries, got.Points)
	}
	v2JSON(t, db, "/api/v2/overview/libraries-daily?from=2020-01-01&to=2020-01-05", &got)
	if got.Libraries == nil || got.Points == nil || len(got.Libraries) != 0 || len(got.Points) != 0 {
		t.Errorf("empty period: %+v", got)
	}
}

// Days and weeks are local (Europe/Paris): 2026-03-28 23:30Z is already the
// 29th; long ranges are bucketed by ISO week (labelled by Monday).
func TestV2OverviewScatterParis(t *testing.T) {
	t.Setenv("TZ", "Europe/Paris")
	db := testutil.NewDB(t)
	if err := db.Create(&[]models.JFLibrary{{Id: "lib-a", Name: sp("Films")}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]models.JFLibraryItem{{Id: "m1", Name: sp("One"), Type: sp("Movie"), ParentId: sp("lib-a"), RunTimeTicks: i64(100 * secTicks), Genres: js(`[]`)}}).Error; err != nil {
		t.Fatal(err)
	}
	rows := []models.JFPlaybackActivity{
		trackedRow("a", "u1", "m1", "Movie", mustUTC(t, "2026-03-28T22:30:00Z"), 60, 100, nil),
		trackedRow("b", "u1", "m1", "Movie", mustUTC(t, "2026-03-28T23:30:00Z"), 60, 100, nil),
		trackedRow("c", "u1", "m1", "Movie", mustUTC(t, "2026-03-29T01:30:00Z"), 60, 100, nil),
		trackedRow("old", "u1", "m1", "Movie", mustUTC(t, "2024-01-01T12:00:00Z"), 60, 100, nil),
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	var got scatterLibraries
	v2JSON(t, db, "/api/v2/overview/libraries-daily?from=2026-03-28&to=2026-03-30", &got)
	if got.Bucket != "day" || fmt.Sprint(got.Points) != "[{2026-03-28 lib-a 1} {2026-03-29 lib-a 2}]" {
		t.Errorf("paris days: %s %v", got.Bucket, got.Points)
	}
	var plays scatterPlays
	v2JSON(t, db, "/api/v2/overview/plays?from=2026-03-28&to=2026-03-30", &plays)
	if len(plays.Points) != 3 || plays.Points[0].StartedAt != "2026-03-29T03:30:00+02:00" || plays.Points[2].StartedAt != "2026-03-28T23:30:00+01:00" {
		t.Errorf("paris plays: %+v", plays.Points)
	}
	// from=03-29 local starts at 2026-03-28 23:00Z: "a" (22:30Z) is out.
	v2JSON(t, db, "/api/v2/overview/plays?from=2026-03-29&to=2026-03-30", &plays)
	if want := []string{"c", "b"}; !reflect.DeepEqual(playIDs(plays), want) {
		t.Errorf("paris 03-29 ids = %v", playIDs(plays))
	}

	v2JSON(t, db, "/api/v2/overview/libraries-daily?from=2024-01-01&to=2026-03-31", &got)
	if got.Bucket != "week" || fmt.Sprint(got.Points) != "[{2024-01-01 lib-a 1} {2026-03-23 lib-a 3}]" {
		t.Errorf("weekly: %s %v", got.Bucket, got.Points)
	}
	if fmt.Sprint(got.Libraries) != "[{lib-a Films 4}]" {
		t.Errorf("weekly libraries: %v", got.Libraries)
	}
}
