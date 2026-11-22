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

func TestV2EntityLibraryRef(t *testing.T) {
	db := v2Seeded(t)
	type ref struct{ ID, Name string }
	var got struct{ Item struct{ Library *ref } }
	for path, want := range map[string]ref{
		"/api/v2/movies/m1":   {"lib-m", "Films"},
		"/api/v2/series/s1":   {"lib-s", "Séries"},
		"/api/v2/episodes/e2": {"lib-s", "Séries"},
		"/api/v2/artists/ar1": {"lib-mu", "Musique"},
		"/api/v2/albums/al1":  {"lib-mu", "Musique"},
	} {
		got.Item.Library = nil
		v2JSON(t, db, path, &got)
		if got.Item.Library == nil || *got.Item.Library != want {
			t.Errorf("%s library = %+v, want %+v", path, got.Item.Library, want)
		}
	}
	// Unknown parent library: null.
	if err := db.Create(&models.JFLibraryItem{Id: "m9", Name: sp("Orphan"), Type: sp("Movie"), ParentId: sp("gone")}).Error; err != nil {
		t.Fatal(err)
	}
	got.Item.Library = &ref{}
	v2JSON(t, db, "/api/v2/movies/m9", &got)
	if got.Item.Library != nil {
		t.Errorf("orphan library = %+v", got.Item.Library)
	}
}

func TestV2TrackNames(t *testing.T) {
	db := v2Seeded(t)
	var got struct {
		Item struct{ AlbumName, ArtistName *string }
	}
	v2JSON(t, db, "/api/v2/tracks/t1", &got)
	if got.Item.AlbumName == nil || *got.Item.AlbumName != "Blue Album" || got.Item.ArtistName == nil || *got.Item.ArtistName != "The Band" {
		t.Errorf("track item = %+v", got.Item)
	}
}

func TestV2LibraryDecadesUserFilter(t *testing.T) {
	db := v2Seeded(t)
	var got []struct{ Decade, Views, Catalog int }
	v2JSON(t, db, "/api/v2/libraries/lib-m/decades?userId=u1", &got)
	if len(got) != 2 || got[1].Decade != 2010 || got[1].Views != 1 || got[1].Catalog != 1 {
		t.Errorf("decades u1 = %+v", got)
	}
}

func TestV2LibraryGrowthFilters(t *testing.T) {
	db := v2Seeded(t)
	type point struct {
		Date              string
		Added, FirstViews int
	}
	var got struct{ Points []point }
	sum := func() (added, first int) {
		for _, p := range got.Points {
			added += p.Added
			first += p.FirstViews
		}
		return
	}
	v2JSON(t, db, "/api/v2/libraries/lib-s/growth?userId=u2", &got)
	if _, first := sum(); first != 0 {
		t.Errorf("u2 first views = %d", first)
	}
	v2JSON(t, db, "/api/v2/libraries/lib-s/growth?userId=u1", &got)
	if _, first := sum(); first != 3 {
		t.Errorf("u1 first views = %d", first)
	}
	v2JSON(t, db, "/api/v2/libraries/lib-m/growth?type=series", &got)
	if added, first := sum(); added != 0 || first != 0 {
		t.Errorf("type=series on movies = %d %d", added, first)
	}
	v2JSON(t, db, "/api/v2/libraries/lib-m/growth?type=movie", &got)
	if added, _ := sum(); added != 1 {
		t.Errorf("type=movie added = %d", added)
	}
}

func TestV2AlbumFullListensIgnoresArchivedTracks(t *testing.T) {
	db := v2Seeded(t)
	if err := db.Create(&models.JFMusicTrack{Id: "t3", Name: sp("Hidden"), AlbumId: sp("al1"), ArtistId: sp("ar1"), Archived: true, RunTimeTicks: i64(200 * secTicks), Genres: js(`[]`)}).Error; err != nil {
		t.Fatal(err)
	}
	// Bob listened to t1 (r7) and now to the archived t3: two distinct tracks
	// on the day, but only one of the two catalog tracks.
	pos := int64(200)
	r := trackedRow("r12", "u2", "t3", "Audio", v2Day1.Add(9*time.Hour), 200, 200, &pos)
	r.NowPlayingItemId = sp("al1")
	if err := db.Create(&r).Error; err != nil {
		t.Fatal(err)
	}
	var got []struct {
		ID          string
		FullListens int
	}
	v2JSON(t, db, "/api/v2/albums/al1/full-listens", &got)
	if len(got) == 0 || got[0].ID != "al1" || got[0].FullListens != 0 {
		t.Errorf("albums = %+v", got)
	}
}
