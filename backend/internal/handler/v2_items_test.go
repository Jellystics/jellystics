package handler_test

import (
	"encoding/json"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"gorm.io/datatypes"
)

func TestItemRoute(t *testing.T) {
	db := testutil.NewDB(t)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	must(db.Create(&[]models.JFLibraryItem{
		{Id: "m1", Type: sp("Movie")},
		{Id: "s1", Type: sp("Series")},
		{Id: "a1", Type: sp("MusicAlbum")},
		{Id: "x1", Type: sp("BoxSet")},
	}).Error)
	must(db.Create(&models.JFLibraryEpisode{Id: "e1", SeriesId: sp("s1")}).Error)
	must(db.Create(&models.JFLibraryEpisode{Id: "e2", EpisodeId: sp("jf-e2"), SeriesId: sp("s1")}).Error)
	must(db.Create(&models.JFLibraryEpisode{Id: "e3"}).Error)
	must(db.Create(&models.JFMusicTrack{Id: "t1", AlbumId: sp("a1"), Genres: datatypes.JSON([]byte("[]"))}).Error)
	must(db.Create(&models.JFMusicTrack{Id: "t2", Genres: datatypes.JSON([]byte("[]"))}).Error)
	must(db.Create(&models.JFMusicArtist{Id: "ar1", Name: sp("The Band"), Genres: datatypes.JSON([]byte("[]"))}).Error)

	cases := []struct {
		id, typ, series, album string
		code                   int
	}{
		{"m1", "movie", "", "", 200},
		{"s1", "series", "", "", 200},
		{"a1", "album", "", "", 200},
		{"ar1", "artist", "", "", 200},
		{"e1", "episode", "s1", "", 200},
		{"jf-e2", "episode", "s1", "", 200},
		{"t1", "track", "", "a1", 200},
		{"x1", "", "", "", 404}, // unsupported library item type
		{"e3", "", "", "", 404}, // episode without series
		{"t2", "", "", "", 404}, // track without album
		{"nope", "", "", "", 404},
	}
	for _, tc := range cases {
		w := v2Get(t, db, "/api/v2/items/"+tc.id+"/route")
		if w.Code != tc.code {
			t.Fatalf("%s: status %d, body %s", tc.id, w.Code, w.Body.String())
		}
		if tc.code != 200 {
			continue
		}
		var got struct{ Type, Id, SeriesId, AlbumId string }
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Type != tc.typ || got.Id != tc.id || got.SeriesId != tc.series || got.AlbumId != tc.album {
			t.Errorf("%s: got %+v", tc.id, got)
		}
	}
}
