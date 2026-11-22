package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
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
	must(db.Create(&models.JFMusicTrack{Id: "t1", AlbumId: sp("a1"), Genres: datatypes.JSON([]byte("[]"))}).Error)

	gin.SetMode(gin.TestMode)
	h := handler.NewItemRouteHandler(db)
	r := gin.New()
	r.GET("/api/v2/items/:id/route", h.Route)

	cases := []struct {
		id, typ, series, album string
		code                   int
	}{
		{"m1", "movie", "", "", 200},
		{"s1", "series", "", "", 200},
		{"a1", "album", "", "", 200},
		{"x1", "other", "", "", 200},
		{"e1", "episode", "s1", "", 200},
		{"jf-e2", "episode", "s1", "", 200},
		{"t1", "track", "", "a1", 200},
		{"nope", "", "", "", 404},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v2/items/"+tc.id+"/route", nil))
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
