package handler_test

import (
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/testutil"
)

func TestV2DiscoverCatalog(t *testing.T) {
	db := v2Seeded(t)

	type item struct {
		ID, Kind, Name string
		DateAdded      *string
		SizeBytes      int64
		Seen           bool
	}
	var un struct {
		Count, Catalog int
		SizeBytes      int64
		Items          []item
	}
	v2JSON(t, db, "/api/v2/discover/unwatched", &un)
	if un.Count != 1 || un.Catalog != 3 || un.SizeBytes != 0 || len(un.Items) != 1 || un.Items[0].ID != "m2" || un.Items[0].Kind != "movie" {
		t.Errorf("unwatched = %+v", un)
	}
	v2JSON(t, db, "/api/v2/discover/unwatched?type=movie", &un)
	if un.Count != 1 || un.Catalog != 2 {
		t.Errorf("unwatched movies = %+v", un)
	}
	v2JSON(t, db, "/api/v2/discover/unwatched?userId=u2", &un) // Bob never watched the series
	if un.Count != 2 || un.Items[0].ID != "m2" && un.Items[1].ID != "m2" {
		t.Errorf("unwatched by u2 = %+v", un)
	}
	v2JSON(t, db, "/api/v2/discover/unwatched?libraryId=lib-s", &un)
	if un.Count != 0 || un.Catalog != 1 {
		t.Errorf("unwatched lib-s = %+v", un)
	}

	var added struct {
		Count, Seen int
		Items       []item
	}
	v2JSON(t, db, "/api/v2/discover/added", &added)
	if added.Count != 1 || added.Seen != 1 || len(added.Items) != 1 || added.Items[0].ID != "m1" || !added.Items[0].Seen ||
		added.Items[0].DateAdded == nil || *added.Items[0].DateAdded != v2Day3.AddDate(0, 0, -10).Format("2006-01-02") {
		t.Errorf("added = %+v", added)
	}

	var storage []struct {
		ID, Kind  string
		SizeBytes int64
		Views     int
	}
	v2JSON(t, db, "/api/v2/discover/storage", &storage)
	if len(storage) != 1 || storage[0].ID != "m1" || storage[0].SizeBytes != 4_000_000_000 || storage[0].Views != 2 {
		t.Errorf("storage = %+v", storage)
	}

	// Episode and track sizes are pre-aggregated per series / album.
	mustExec(t, db, `INSERT INTO jf_item_info ("Id", "Size", "Type") VALUES ('e1', 1000, 'Episode'), ('e2', 2000, 'Episode'), ('t1', 300, 'Audio')`)
	v2JSON(t, db, "/api/v2/discover/storage", &storage)
	if len(storage) != 2 || storage[1].ID != "s1" || storage[1].SizeBytes != 3000 {
		t.Errorf("storage with episodes = %+v", storage)
	}
	v2JSON(t, db, "/api/v2/discover/storage?type=music", &storage)
	if len(storage) != 1 || storage[0].ID != "al1" || storage[0].SizeBytes != 300 {
		t.Errorf("storage music = %+v", storage)
	}
	v2JSON(t, db, "/api/v2/discover/added?from=2000-01-01&to=2000-01-02", &added)
	if added.Count != 0 || added.Seen != 0 || len(added.Items) != 0 {
		t.Errorf("added empty = %+v", added)
	}
	if w := v2Get(t, db, "/api/v2/discover/decades"); w.Code != 404 {
		t.Errorf("/discover/decades should be gone, status %d", w.Code)
	}
}

func TestV2DiscoverAbandoned(t *testing.T) {
	db := v2Seeded(t)
	var ab struct {
		Count int
		Items []struct {
			ID, Kind, Name                string
			User                          struct{ ID, Name string }
			Progress                      float64
			LastPlayed                    time.Time
			PositionSeconds               *int64
			EpisodesViewed, EpisodesTotal *int
		}
	}
	v2JSON(t, db, "/api/v2/discover/abandoned", &ab)
	if ab.Count != 0 || len(ab.Items) != 0 {
		t.Fatalf("abandoned before = %+v", ab)
	}
	pos50, pos100 := int64(50), int64(100)
	rows := []models.JFPlaybackActivity{
		trackedRow("ab1", "u2", "m2", "Movie", testutil.RecentDay(90).Add(20*time.Hour), 50, 100, &pos50),
		trackedRow("ab2", "u2", "e1", "Episode", testutil.RecentDay(100).Add(20*time.Hour), 100, 100, &pos100),
		trackedRow("ab3", "u1", "m2", "Movie", testutil.RecentDay(90).Add(21*time.Hour), 95, 100, &pos100), // finished: not abandoned
	}
	rows[0].UserName, rows[1].UserName, rows[2].UserName = sp("Bob"), sp("Bob"), sp("Alice")
	rows[1].EpisodeId = sp("e1")
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	v2JSON(t, db, "/api/v2/discover/abandoned", &ab)
	if ab.Count != 2 || len(ab.Items) != 2 {
		t.Fatalf("abandoned = %+v", ab)
	}
	m, s := ab.Items[0], ab.Items[1]
	if m.ID != "m2" || m.Kind != "movie" || m.User.ID != "u2" || !near(m.Progress, 0.5) || m.PositionSeconds == nil || *m.PositionSeconds != 50 {
		t.Errorf("abandoned movie = %+v", m)
	}
	if s.ID != "s1" || s.Kind != "series" || s.User.Name != "Bob" || !near(s.Progress, 1.0/3) ||
		s.EpisodesViewed == nil || *s.EpisodesViewed != 1 || s.EpisodesTotal == nil || *s.EpisodesTotal != 3 {
		t.Errorf("abandoned series = %+v", s)
	}
	v2JSON(t, db, "/api/v2/discover/abandoned?type=series", &ab)
	if ab.Count != 1 || ab.Items[0].ID != "s1" {
		t.Errorf("abandoned series only = %+v", ab)
	}
	v2JSON(t, db, "/api/v2/discover/abandoned?userId=u1", &ab)
	if ab.Count != 0 {
		t.Errorf("abandoned u1 = %+v", ab)
	}

	// Under 10 % of the episodes (1 of 11) is a sampling, not an abandon.
	eps := make([]models.JFLibraryEpisode, 8)
	for i := range eps {
		id := "ex" + string(rune('a'+i))
		eps[i] = models.JFLibraryEpisode{Id: id, EpisodeId: sp(id), Name: sp(id), SeriesId: sp("s1"), SeasonId: sp("se2"), SeriesName: sp("Les Rivages")}
	}
	if err := db.Create(&eps).Error; err != nil {
		t.Fatal(err)
	}
	v2JSON(t, db, "/api/v2/discover/abandoned?type=series", &ab)
	if ab.Count != 0 || len(ab.Items) != 0 {
		t.Errorf("abandoned series under 10 %% = %+v", ab)
	}
}
