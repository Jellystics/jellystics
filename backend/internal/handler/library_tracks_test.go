package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// TestGetLibraryTracks_PlayCount verifies each track's PlayCount comes from
// views whose ItemId (the track actually played) matches — NOT the parent
// NowPlayingItemId (the album). A decoy row with NowPlayingItemId=track-1 must
// not inflate track-1, and a short click is not a play.
func TestGetLibraryTracks_PlayCount(t *testing.T) {
	db := testutil.NewDB(t)
	genres := datatypes.JSON([]byte("[]"))

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	must(db.Create(&[]models.JFMusicTrack{
		{Id: "track-1", Name: sp("One More Time"), AlbumId: sp("album-1"), AlbumName: sp("Discovery"), AlbumArtist: sp("Daft Punk"), LibraryId: sp("lib-music"), Genres: genres},
		{Id: "track-2", Name: sp("Aerodynamic"), AlbumId: sp("album-1"), AlbumName: sp("Discovery"), AlbumArtist: sp("Daft Punk"), LibraryId: sp("lib-music"), Genres: genres},
	}).Error)

	start := testutil.RecentDay(1).Add(10 * time.Hour)
	play := func(id string, dur int64) models.JFPlaybackActivity {
		r := trackedRow(id, "u1", "track-1", "Audio", start, dur, 180, nil)
		r.NowPlayingItemId = sp("album-1") // parent (album) id
		return r
	}
	decoy := trackedRow("decoy", "u1", "track-2", "Audio", start, 180, 180, nil)
	decoy.NowPlayingItemId = sp("track-1")
	must(db.Create(&[]models.JFPlaybackActivity{
		// track-1 viewed 3× (ItemId = track id, NowPlayingItemId = album).
		play("p1", 180), play("p2", 180), play("p3", 180),
		// A 5 s click on track-1 is not a play.
		play("click", 5),
		// Decoy: NowPlayingItemId equals track-1 but the item played is track-2 —
		// must count for track-2, not track-1 (keyed by ItemId).
		decoy,
	}).Error)

	gin.SetMode(gin.TestMode)
	h := handler.NewStatsFrontendHandler(db, repository.New(db))
	r := gin.New()
	r.GET("/tracks", h.GetLibraryTracks)

	req := httptest.NewRequest(http.MethodGet, "/tracks?libraryId=lib-music", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var rows []struct {
		Id        string `json:"Id"`
		PlayCount int    `json:"PlayCount"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}
	counts := map[string]int{}
	for _, r := range rows {
		counts[r.Id] = r.PlayCount
	}
	if counts["track-1"] != 3 {
		t.Errorf("track-1 PlayCount = %d, want 3 (views by ItemId; click and decoy excluded)", counts["track-1"])
	}
	if counts["track-2"] != 1 {
		t.Errorf("track-2 PlayCount = %d, want 1 (decoy)", counts["track-2"])
	}
}
