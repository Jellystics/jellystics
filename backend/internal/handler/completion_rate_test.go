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
	"gorm.io/gorm"
)

func sp(s string) *string { return &s }
func i64(v int64) *int64  { return &v }
func tp(t time.Time) *time.Time {
	return &t
}

const secTicks = 10_000_000 // ticks per second

// trackedRow builds a playback row with the accurate-tracking columns set.
func trackedRow(id, user, itemId, mediaType string, start time.Time, durSec, runtimeSec int64, posEndSec *int64) models.JFPlaybackActivity {
	r := models.JFPlaybackActivity{
		Id: id, UserId: sp(user), UserName: sp(user),
		NowPlayingItemId: sp(itemId), NowPlayingItemName: sp(itemId),
		ItemId: sp(itemId), MediaType: sp(mediaType),
		PlaybackDuration: i64(durSec), RunTimeTicks: i64(runtimeSec * secTicks),
		StartedAt: tp(start), EndedAt: tp(start.Add(time.Duration(durSec) * time.Second)),
		ActivityDateInserted: sp(testutil.TS(start)),
		IsView:               durSec >= 30,
		Source:               "watchdog",
	}
	if posEndSec != nil {
		r.PositionTicksEnd = i64(*posEndSec * secTicks)
	}
	return r
}

// seedCompletion inserts views with exact completion ratios (all on the same
// day, user u1):
//
//	movie-1  100/100 s                         → 1.00 completed
//	movie-2   30/100 s                         → 0.30 abandoned
//	ep-1     two sittings: stop at 40 s, then resumed and stopped at 96 s
//	                                           → one unit, 0.96 completed
//	ep-2      80/100 s                         → 0.80 abandoned (< 0.9)
//	track-1  200/200 s (Audio)                 → 1.00 completed
//
// plus a 10 s click on movie-1 by u2, which is not a view and must be ignored.
func seedCompletion(t *testing.T, db *gorm.DB) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	genres := datatypes.JSON([]byte("[]"))
	movie := "Movie"
	must(db.Create(&[]models.JFLibraryItem{
		{Id: "movie-1", Name: sp("Inception"), Type: &movie, ParentId: sp("lib-movies"), RunTimeTicks: i64(100 * secTicks), Genres: genres},
		{Id: "movie-2", Name: sp("Tenet"), Type: &movie, ParentId: sp("lib-movies"), RunTimeTicks: i64(100 * secTicks), Genres: genres},
	}).Error)
	must(db.Create(&[]models.JFLibraryEpisode{
		{Id: "ep-1", Name: sp("Pilot"), RunTimeTicks: i64(100 * secTicks), SeriesId: sp("series-1")},
		{Id: "ep-2", Name: sp("Second"), RunTimeTicks: i64(100 * secTicks), SeriesId: sp("series-1")},
	}).Error)
	must(db.Create(&models.JFMusicTrack{
		Id: "track-1", Name: sp("One More Time"), AlbumId: sp("album-1"),
		RunTimeTicks: i64(200 * secTicks), LibraryId: sp("lib-music"), Genres: genres,
	}).Error)

	day := testutil.RecentDay(2).Add(10 * time.Hour)
	p40, p96, p80 := int64(40), int64(96), int64(80)
	must(db.Create(&[]models.JFPlaybackActivity{
		trackedRow("c1", "u1", "movie-1", "Movie", day, 100, 100, nil),
		trackedRow("c2", "u1", "movie-2", "Movie", day.Add(time.Hour), 30, 100, nil),
		trackedRow("c3a", "u1", "ep-1", "Episode", day.Add(2*time.Hour), 40, 100, &p40),
		trackedRow("c3b", "u1", "ep-1", "Episode", day.Add(3*time.Hour), 56, 100, &p96),
		trackedRow("c5", "u1", "ep-2", "Episode", day.Add(4*time.Hour), 80, 100, &p80),
		trackedRow("c4", "u1", "track-1", "Audio", day.Add(5*time.Hour), 200, 200, nil),
		trackedRow("click", "u2", "movie-1", "Movie", day.Add(6*time.Hour), 10, 100, nil),
	}).Error)
}

type completionResponse struct {
	Overall struct {
		AvgCompletionRate float64 `json:"avgCompletionRate"`
		TotalPlays        int     `json:"totalPlays"`
		CompletedPlays    int     `json:"completedPlays"`
		AbandonedPlays    int     `json:"abandonedPlays"`
	} `json:"overall"`
	ByType []struct {
		Type              string  `json:"type"`
		AvgCompletionRate float64 `json:"avgCompletionRate"`
		TotalPlays        int     `json:"totalPlays"`
	} `json:"byType"`
	Threshold float64 `json:"threshold"`
}

func TestGetCompletionRate_IncludesAudio(t *testing.T) {
	db := testutil.NewDB(t)
	seedCompletion(t, db)

	gin.SetMode(gin.TestMode)
	h := handler.NewStatsFrontendHandler(db, repository.New(db))
	r := gin.New()
	r.GET("/completion", h.GetCompletionRate)

	req := httptest.NewRequest(http.MethodGet, "/completion?from=all", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp completionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}

	if resp.Threshold != 0.9 {
		t.Errorf("threshold = %v, want 0.9", resp.Threshold)
	}
	// 5 (user, item, day) units; the click is not a view; ep-1's two sittings
	// are one unit.
	if resp.Overall.TotalPlays != 5 {
		t.Errorf("overall totalPlays = %d, want 5", resp.Overall.TotalPlays)
	}
	// Completed (>= 0.9): movie-1 1.0, ep-1 0.96 (resumed), track 1.0.
	if resp.Overall.CompletedPlays != 3 {
		t.Errorf("completedPlays = %d, want 3", resp.Overall.CompletedPlays)
	}
	// Abandoned: movie-2 0.30, ep-2 0.8. ep-1's first sitting is NOT an abandon.
	if resp.Overall.AbandonedPlays != 2 {
		t.Errorf("abandonedPlays = %d, want 2", resp.Overall.AbandonedPlays)
	}

	byType := map[string]struct {
		avg   float64
		plays int
	}{}
	for _, r := range resp.ByType {
		byType[r.Type] = struct {
			avg   float64
			plays int
		}{r.AvgCompletionRate, r.TotalPlays}
	}
	audio, ok := byType["Audio"]
	if !ok {
		t.Fatalf("byType missing 'Audio' — music completion is not counted; got %+v", resp.ByType)
	}
	if audio.plays != 1 || audio.avg != 1.0 {
		t.Errorf("Audio = {plays:%d avg:%v}, want {1, 1.0}", audio.plays, audio.avg)
	}
	if m := byType["Movie"]; m.plays != 2 || m.avg != 0.65 { // (1.0+0.30)/2
		t.Errorf("Movie = {plays:%d avg:%v}, want {2, 0.65}", m.plays, m.avg)
	}
	if e := byType["Episode"]; e.plays != 2 || e.avg != 0.88 { // (0.96+0.8)/2 = 0.88
		t.Errorf("Episode = {plays:%d avg:%v}, want {2, 0.88}", e.plays, e.avg)
	}
}
