package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
)

func TestV2MovieSittings(t *testing.T) {
	db := v2Seeded(t)
	// Bob viewed 30 % on day 3 (r2) and finishes on day 1: two sittings.
	pos := int64(100)
	r := trackedRow("r10", "u2", "m1", "Movie", v2Day1.Add(12*time.Hour), 80, 100, &pos)
	if err := db.Create(&r).Error; err != nil {
		t.Fatal(err)
	}
	var got []struct{ Sittings, Count int }
	v2JSON(t, db, "/api/v2/movies/m1/sittings", &got)
	// Alice: r1 completes in one sitting (her previous completion r9 a year
	// ago closes the earlier segment and is outside the range).
	want := []struct{ Sittings, Count int }{{1, 1}, {2, 1}, {3, 0}, {4, 0}}
	if len(got) != 4 {
		t.Fatalf("sittings = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sittings = %+v, want %+v", got, want)
			break
		}
	}
	v2JSON(t, db, "/api/v2/movies/m1/sittings?userId=u1", &got)
	if got[0].Count != 1 || got[1].Count != 0 {
		t.Errorf("user filter: %+v", got)
	}
	for _, id := range []string{"s1", "nope"} {
		if w := v2Get(t, db, "/api/v2/movies/"+id+"/sittings"); w.Code != http.StatusNotFound {
			t.Errorf("%s: status %d", id, w.Code)
		}
	}
}

func TestV2LibraryDecades(t *testing.T) {
	db := v2Seeded(t)
	var got []struct{ Decade, Views, Catalog int }
	v2JSON(t, db, "/api/v2/libraries/lib-m/decades", &got)
	if len(got) != 2 || got[0] != (struct{ Decade, Views, Catalog int }{2000, 0, 1}) || got[1] != (struct{ Decade, Views, Catalog int }{2010, 2, 1}) {
		t.Errorf("decades = %+v", got)
	}
	if w := v2Get(t, db, "/api/v2/libraries/nope/decades"); w.Code != http.StatusNotFound {
		t.Errorf("unknown library: status %d", w.Code)
	}
}

func TestV2LibraryGrowth(t *testing.T) {
	db := v2Seeded(t)
	type point struct {
		Date              string
		Added, FirstViews int
	}
	var got struct {
		Bucket string
		Points []point
	}
	find := func(date string) point {
		for _, p := range got.Points {
			if p.Date == date {
				return p
			}
		}
		t.Fatalf("no point %s in %+v", date, got.Points)
		return point{}
	}
	v2JSON(t, db, "/api/v2/libraries/lib-m/growth", &got)
	if got.Bucket != "day" || len(got.Points) != 30 {
		t.Fatalf("bucket = %s, %d points", got.Bucket, len(got.Points))
	}
	// m1 was added 13 days ago; its first view ever is a year old.
	if p := find(v2Day3.AddDate(0, 0, -10).Format("2006-01-02")); p.Added != 1 || p.FirstViews != 0 {
		t.Errorf("added point = %+v", p)
	}
	v2JSON(t, db, "/api/v2/libraries/lib-s/growth", &got)
	if p := find(v2Day2.Format("2006-01-02")); p.FirstViews != 2 {
		t.Errorf("day2 = %+v", p)
	}
	if p := find(v2Day1.Format("2006-01-02")); p.FirstViews != 1 {
		t.Errorf("day1 = %+v", p)
	}
	v2JSON(t, db, "/api/v2/libraries/lib-s/growth?from=2025-01-01&to=2026-01-01", &got)
	if got.Bucket != "month" || len(got.Points) != 12 || got.Points[0].Date != "2025-01-01" || got.Points[11].Date != "2025-12-01" {
		t.Errorf("monthly = %s %+v", got.Bucket, got.Points)
	}
	v2JSON(t, db, "/api/v2/libraries/lib-s/growth?from=2026-03-04&to=2026-05-04", &got)
	if got.Bucket != "week" || got.Points[0].Date != "2026-03-02" {
		t.Errorf("weekly = %s %+v", got.Bucket, got.Points)
	}
	if w := v2Get(t, db, "/api/v2/libraries/nope/growth"); w.Code != http.StatusNotFound {
		t.Errorf("unknown library: status %d", w.Code)
	}
}

func TestV2AlbumFullListens(t *testing.T) {
	db := v2Seeded(t)
	if err := db.Create(&models.JFLibraryItem{Id: "al2", Name: sp("Red Album"), Type: sp("MusicAlbum"), ParentId: sp("lib-mu"), ArtistId: sp("ar1"), ProductionYear: ip(2005)}).Error; err != nil {
		t.Fatal(err)
	}
	type album struct {
		ID, Name                     string
		Year                         *int
		Tracks, Listens, FullListens int
	}
	var got []album
	v2JSON(t, db, "/api/v2/albums/al1/full-listens", &got)
	// Bob skipped t2 (r8): one listen, no full listen yet.
	if len(got) != 2 || got[0].ID != "al1" || got[0].Listens != 1 || got[0].FullListens != 0 || got[0].Tracks != 2 || got[1].ID != "al2" {
		t.Fatalf("albums = %+v", got)
	}
	pos := int64(200)
	r := trackedRow("r11", "u2", "t2", "Audio", v2Day1.Add(8*time.Hour+10*time.Minute), 200, 200, &pos)
	r.NowPlayingItemId = sp("al1")
	if err := db.Create(&r).Error; err != nil {
		t.Fatal(err)
	}
	v2JSON(t, db, "/api/v2/albums/al2/full-listens", &got)
	if len(got) != 2 || got[0].ID != "al1" || got[0].Listens != 2 || got[0].FullListens != 1 || got[1].FullListens != 0 {
		t.Errorf("albums = %+v", got)
	}
	if w := v2Get(t, db, "/api/v2/albums/nope/full-listens"); w.Code != http.StatusNotFound {
		t.Errorf("unknown album: status %d", w.Code)
	}
}

func TestV2SeriesHistory(t *testing.T) {
	db := v2Seeded(t)
	var got struct {
		Rows                  []v2HistoryRow
		Total, Page, PageSize int
	}
	v2JSON(t, db, "/api/v2/series/s1/history?pageSize=2", &got)
	if got.Total != 3 || len(got.Rows) != 2 || got.Rows[0].ID != "r6" || got.Rows[0].Item.Name != "Return" ||
		got.Rows[0].Item.Season == nil || *got.Rows[0].Item.Season != 2 {
		t.Fatalf("history = %+v", got)
	}
	v2JSON(t, db, "/api/v2/series/s1/history?page=2&pageSize=2", &got)
	if len(got.Rows) != 1 || got.Rows[0].ID != "r4" {
		t.Errorf("page 2 = %+v", got.Rows)
	}
	v2JSON(t, db, "/api/v2/series/s1/history?userId=u2", &got)
	if got.Total != 0 || len(got.Rows) != 0 {
		t.Errorf("user filter = %+v", got)
	}
	for _, id := range []string{"m1", "nope"} {
		if w := v2Get(t, db, "/api/v2/series/"+id+"/history"); w.Code != http.StatusNotFound {
			t.Errorf("%s: status %d", id, w.Code)
		}
	}
}
