package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)


func js(s string) datatypes.JSON { return datatypes.JSON([]byte(s)) }

// v2Days are the UTC days used by seedV2 (relative to today).
var (
	v2Day1 = testutil.RecentDay(1)
	v2Day2 = testutil.RecentDay(2)
	v2Day3 = testutil.RecentDay(3)
)

// seedV2 builds a small catalog (1 movie, 1 series with 3 episodes over 2
// seasons, 1 artist with 1 album of 2 tracks) and 8 playback rows inside the
// default 30-day window plus 1 row exactly one year ago:
//
//	r1 u1 m1 Movie  day3 20:00 100/100 s DirectPlay   Web/Chrome      view, completed
//	r2 u2 m1 Movie  day3 20:01  30/100 s Transcode    Android/Pixel   view
//	r3 u2 m1 Movie  day2 10:00  10/100 s DirectPlay   Android/Pixel   click
//	r4 u1 e1 Ep     day2 21:00 100/100 s DirectStream Web/Chrome      view
//	r5 u1 e2 Ep     day2 21:05  90 s, stop 95/100     DirectStream    view
//	r6 u1 e3 Ep     day1 21:00  50/100 s DirectStream Web/Chrome      view
//	r7 u2 t1 Audio  day1 08:00 200/200 s DirectPlay   Android/Pixel   view
//	r8 u2 t2 Audio  day1 08:04  10/200 s DirectPlay   Android/Pixel   click (skip)
//	r9 u1 m1 Movie  one year ago 12:00 100 s          (on this day)
func seedV2(t *testing.T, db *gorm.DB) {
	t.Helper()
	t.Setenv("TZ", "UTC")
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	must(db.Create(&[]models.JFUser{{Id: "u1", Name: sp("Alice")}, {Id: "u2", Name: sp("Bob")}}).Error)
	must(db.Create(&[]models.JFLibrary{
		{Id: "lib-m", Name: sp("Films"), CollectionType: sp("movies")},
		{Id: "lib-s", Name: sp("Séries"), CollectionType: sp("tvshows")},
		{Id: "lib-mu", Name: sp("Musique"), CollectionType: sp("music")},
	}).Error)
	must(db.Create(&[]models.JFLibraryItem{
		{Id: "m1", Name: sp("Inception"), Type: sp("Movie"), ParentId: sp("lib-m"), ProductionYear: ip(2010), RunTimeTicks: i64(100 * secTicks), Genres: js(`["Sci-Fi"]`), DateCreated: sp(testutil.TS(v2Day3.AddDate(0, 0, -10)))},
		{Id: "m2", Name: sp("Unseen"), Type: sp("Movie"), ParentId: sp("lib-m"), ProductionYear: ip(2001), RunTimeTicks: i64(100 * secTicks), Genres: js(`["Drama"]`)},
		{Id: "s1", Name: sp("Les Rivages"), Type: sp("Series"), ParentId: sp("lib-s"), ProductionYear: ip(2020), Genres: js(`["Drama"]`)},
		{Id: "al1", Name: sp("Blue Album"), Type: sp("MusicAlbum"), ParentId: sp("lib-mu"), ArtistId: sp("ar1"), AlbumArtist: sp("The Band"), ProductionYear: ip(1999), Genres: js(`["Rock"]`)},
	}).Error)
	must(db.Create(&[]models.JFItemInfo{{Id: "m1", Size: i64(4_000_000_000), Type: sp("Movie"), MediaStreams: js(`[{"Type":"Video","Width":1920,"Height":1080}]`)}}).Error)
	must(db.Create(&[]models.JFLibrarySeason{
		{Id: "se1", Name: sp("Season 1"), IndexNumber: ip(1), SeriesId: sp("s1")},
		{Id: "se2", Name: sp("Season 2"), IndexNumber: ip(2), SeriesId: sp("s1")},
	}).Error)
	premiere := testutil.TS(v2Day3.AddDate(0, 0, -3))
	must(db.Create(&[]models.JFLibraryEpisode{
		{Id: "e1", EpisodeId: sp("e1"), Name: sp("Pilot"), SeriesId: sp("s1"), SeasonId: sp("se1"), SeriesName: sp("Les Rivages"), ParentIndexNumber: ip(1), IndexNumber: ip(1), RunTimeTicks: i64(100 * secTicks), PremiereDate: &premiere},
		{Id: "e2", EpisodeId: sp("e2"), Name: sp("Second"), SeriesId: sp("s1"), SeasonId: sp("se1"), SeriesName: sp("Les Rivages"), ParentIndexNumber: ip(1), IndexNumber: ip(2), RunTimeTicks: i64(100 * secTicks), PremiereDate: &premiere},
		{Id: "e3", EpisodeId: sp("e3"), Name: sp("Return"), SeriesId: sp("s1"), SeasonId: sp("se2"), SeriesName: sp("Les Rivages"), ParentIndexNumber: ip(2), IndexNumber: ip(1), RunTimeTicks: i64(100 * secTicks), PremiereDate: &premiere},
	}).Error)
	must(db.Create(&[]models.JFMusicArtist{{Id: "ar1", Name: sp("The Band"), LibraryId: sp("lib-mu"), Genres: js(`["Rock"]`)}}).Error)
	must(db.Create(&[]models.JFMusicTrack{
		{Id: "t1", Name: sp("Opening"), AlbumId: sp("al1"), AlbumName: sp("Blue Album"), ArtistId: sp("ar1"), LibraryId: sp("lib-mu"), IndexNumber: ip(1), RunTimeTicks: i64(200 * secTicks), Genres: js(`["Rock"]`)},
		{Id: "t2", Name: sp("Closing"), AlbumId: sp("al1"), AlbumName: sp("Blue Album"), ArtistId: sp("ar1"), LibraryId: sp("lib-mu"), IndexNumber: ip(2), RunTimeTicks: i64(200 * secTicks), Genres: js(`["Rock"]`)},
	}).Error)

	at := func(day time.Time, h, m int) time.Time {
		return day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
	}
	row := func(id, user, name, item, mt string, start time.Time, dur, runtime int64, pos int64, method, client, device string) models.JFPlaybackActivity {
		r := trackedRow(id, user, item, mt, start, dur, runtime, &pos)
		r.UserName = sp(name)
		r.PlayMethod, r.Client, r.DeviceName = sp(method), sp(client), sp(device)
		r.ApplicationVersion = sp("10.9")
		switch mt {
		case "Episode":
			r.NowPlayingItemId, r.EpisodeId, r.SeriesName = sp("s1"), sp(item), sp("Les Rivages")
		case "Audio":
			r.NowPlayingItemId = sp("al1")
		}
		return r
	}
	rows := []models.JFPlaybackActivity{
		row("r1", "u1", "Alice", "m1", "Movie", at(v2Day3, 20, 0), 100, 100, 100, "DirectPlay", "Web", "Chrome"),
		row("r2", "u2", "Bob", "m1", "Movie", at(v2Day3, 20, 1), 30, 100, 30, "Transcode", "Android", "Pixel"),
		row("r3", "u2", "Bob", "m1", "Movie", at(v2Day2, 10, 0), 10, 100, 10, "DirectPlay", "Android", "Pixel"),
		row("r4", "u1", "Alice", "e1", "Episode", at(v2Day2, 21, 0), 100, 100, 100, "DirectStream", "Web", "Chrome"),
		row("r5", "u1", "Alice", "e2", "Episode", at(v2Day2, 21, 5), 90, 100, 95, "DirectStream", "Web", "Chrome"),
		row("r6", "u1", "Alice", "e3", "Episode", at(v2Day1, 21, 0), 50, 100, 50, "DirectStream", "Web", "Chrome"),
		row("r7", "u2", "Bob", "t1", "Audio", at(v2Day1, 8, 0), 200, 200, 200, "DirectPlay", "Android", "Pixel"),
		row("r8", "u2", "Bob", "t2", "Audio", at(v2Day1, 8, 4), 10, 200, 10, "DirectPlay", "Android", "Pixel"),
		row("r9", "u1", "Alice", "m1", "Movie", at(testutil.RecentDay(0).AddDate(-1, 0, 0), 12, 0), 100, 100, 100, "DirectPlay", "Web", "Chrome"),
	}
	rows[4].PausedSeconds = i64(42)
	must(db.Create(&rows).Error)
}

// v2Get serves one GET through the full v2 route table on a seeded database.
func v2Get(t *testing.T, db *gorm.DB, target string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler.NewV2Handler(db).Register(&r.RouterGroup)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	return w
}

// v2JSON GETs target, expects 200 and decodes the body into out.
func v2JSON(t *testing.T, db *gorm.DB, target string, out interface{}) {
	t.Helper()
	w := v2Get(t, db, target)
	if w.Code != http.StatusOK {
		t.Fatalf("%s: status %d, body %s", target, w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
		t.Fatalf("%s: decode: %v (%s)", target, err, w.Body.String())
	}
}

func v2Seeded(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.NewDB(t)
	seedV2(t, db)
	return db
}
