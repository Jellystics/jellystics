package handler_test

import (
	"net/http"
	"testing"
)

func approx(a *float64, want float64) bool {
	return a != nil && *a > want-1e-6 && *a < want+1e-6
}

func TestV2Movie(t *testing.T) {
	db := v2Seeded(t)
	var got struct {
		Item struct {
			ID, Name       string
			Year           int
			RuntimeSeconds int64
			Genres         []string
			SizeBytes      int64
			Resolution     *string
		}
		Totals struct {
			Views, Clicks, Viewers         int
			CompletedRatio, TranscodeRatio *float64
			WatchSeconds                   int64
		}
		Funnel        struct{ Clicks, Views, Half, Completed int }
		StopPositions []int
		Viewers       []struct {
			User           struct{ ID, Name string }
			Views          int
			BestCompletion *float64
		}
		History []v2HistoryRow
	}
	v2JSON(t, db, "/api/v2/movies/m1", &got)
	if got.Item.Name != "Inception" || got.Item.Year != 2010 || got.Item.RuntimeSeconds != 100 || len(got.Item.Genres) != 1 ||
		got.Item.SizeBytes != 4_000_000_000 || got.Item.Resolution == nil || *got.Item.Resolution != "1080p" {
		t.Errorf("item = %+v", got.Item)
	}
	tt := got.Totals
	if tt.Views != 2 || tt.Clicks != 3 || tt.Viewers != 2 || tt.WatchSeconds != 140 || !approx(tt.TranscodeRatio, 0.5) || !approx(tt.CompletedRatio, 0.5) {
		t.Errorf("totals = %+v", tt)
	}
	if got.Funnel != (struct{ Clicks, Views, Half, Completed int }{3, 2, 1, 1}) {
		t.Errorf("funnel = %+v", got.Funnel)
	}
	if len(got.StopPositions) != 24 || got.StopPositions[23] != 1 || got.StopPositions[7] != 1 {
		t.Errorf("stops = %v", got.StopPositions)
	}
	if len(got.Viewers) != 2 || len(got.History) != 3 || got.History[0].ID != "r3" {
		t.Errorf("viewers = %+v history = %d", got.Viewers, len(got.History))
	}
	// Range and user apply to stats, never to the catalog item.
	v2JSON(t, db, "/api/v2/movies/m1?userId=u2", &got)
	if got.Totals.Views != 1 || got.Totals.Clicks != 2 || got.Item.Name != "Inception" {
		t.Errorf("user filter: %+v", got.Totals)
	}
	for _, id := range []string{"s1", "nope"} {
		if w := v2Get(t, db, "/api/v2/movies/"+id); w.Code != http.StatusNotFound {
			t.Errorf("movie %s: %d", id, w.Code)
		}
	}
}

func TestV2Series(t *testing.T) {
	db := v2Seeded(t)
	var got struct {
		Item struct {
			Name              string
			Years             struct{ From, To *int }
			Seasons, Episodes int
			Genres            []string
		}
		Totals struct {
			WatchSeconds                   int64
			Views, Clicks, Viewers, Binges int
			Retention                      *float64
		}
		Retention [][]int
		Seasons   []struct {
			Number   int
			Episodes []struct {
				ID             string
				Number         int
				Views, Viewers int
				AvgCompletion  *float64
				Abandons       int
			}
		}
		Viewers []struct {
			User            struct{ Name string }
			EpisodesWatched int
		}
		Cadence []struct {
			Bucket string
			Count  int
		}
		ReleaseLag []struct {
			Bucket string
			Count  int
		}
	}
	v2JSON(t, db, "/api/v2/series/s1", &got)
	if got.Item.Name != "Les Rivages" || got.Item.Seasons != 2 || got.Item.Episodes != 3 || got.Item.Years.From == nil || *got.Item.Years.From != 2020 {
		t.Errorf("item = %+v", got.Item)
	}
	if got.Totals.Views != 3 || got.Totals.Clicks != 3 || got.Totals.Viewers != 1 || got.Totals.WatchSeconds != 240 || got.Totals.Binges != 0 || !approx(got.Totals.Retention, 1) {
		t.Errorf("totals = %+v", got.Totals)
	}
	if len(got.Retention) != 2 || len(got.Retention[0]) != 2 || len(got.Retention[1]) != 1 {
		t.Errorf("retention = %v", got.Retention)
	}
	if len(got.Seasons) != 2 || got.Seasons[0].Number != 1 || got.Seasons[1].Episodes[0].ID != "e3" {
		t.Fatalf("seasons = %+v", got.Seasons)
	}
	e2 := got.Seasons[0].Episodes[1]
	e3 := got.Seasons[1].Episodes[0]
	if e2.ID != "e2" || !approx(e2.AvgCompletion, 0.95) || e2.Abandons != 0 || !approx(e3.AvgCompletion, 0.5) || e3.Abandons != 1 {
		t.Errorf("episodes e2 = %+v e3 = %+v", e2, e3)
	}
	if len(got.Viewers) != 1 || got.Viewers[0].EpisodesWatched != 3 {
		t.Errorf("viewers = %+v", got.Viewers)
	}
	counts := map[string]int{}
	for _, b := range got.Cadence {
		counts[b.Bucket] = b.Count
	}
	if len(got.Cadence) != 7 || counts["1-15min"] != 1 || counts["nextDay"] != 1 {
		t.Errorf("cadence = %+v", got.Cadence)
	}
	if len(got.ReleaseLag) != 6 || got.ReleaseLag[1].Bucket != "1-7d" || got.ReleaseLag[1].Count != 3 {
		t.Errorf("releaseLag = %+v", got.ReleaseLag)
	}
	if w := v2Get(t, db, "/api/v2/series/m1"); w.Code != http.StatusNotFound {
		t.Errorf("series m1: %d", w.Code)
	}
}

func TestV2Episode(t *testing.T) {
	db := v2Seeded(t)
	var got struct {
		Item struct {
			ID, Name, SeriesID, SeriesName string
			Season, Episode                int
			RuntimeSeconds                 int64
			PremiereDate                   *string
			Prev, Next                     *struct {
				ID              string
				Season, Episode int
			}
			Library *struct{ ID, Name string }
		}
		Totals struct {
			Views, Viewers                  int
			AvgCompletion, PausedSecondsAvg *float64
		}
		Funnel        struct{ Clicks, Views, Half, Completed int }
		StopPositions []int
		Viewers       []struct {
			User         struct{ Name string }
			WatchSeconds int64
			Completion   *float64
			Device       string
		}
	}
	v2JSON(t, db, "/api/v2/episodes/e2", &got)
	it := got.Item
	if it.SeriesID != "s1" || it.SeriesName != "Les Rivages" || it.Season != 1 || it.Episode != 2 || it.RuntimeSeconds != 100 ||
		it.Prev == nil || it.Prev.ID != "e1" || it.Prev.Season != 1 || it.Prev.Episode != 1 ||
		it.Next == nil || it.Next.ID != "e3" || it.Next.Season != 2 || it.Next.Episode != 1 ||
		it.Library == nil || it.Library.ID != "lib-s" || it.Library.Name != "Séries" || it.PremiereDate == nil || len(*it.PremiereDate) != 10 {
		t.Errorf("item = %+v", it)
	}
	if got.Totals.Views != 1 || got.Totals.Viewers != 1 || !approx(got.Totals.AvgCompletion, 0.95) || !approx(got.Totals.PausedSecondsAvg, 42) {
		t.Errorf("totals = %+v", got.Totals)
	}
	if len(got.StopPositions) != 20 || got.StopPositions[19] != 1 || got.Funnel.Completed != 1 {
		t.Errorf("stops = %v funnel = %+v", got.StopPositions, got.Funnel)
	}
	if len(got.Viewers) != 1 || got.Viewers[0].Device != "Chrome" || got.Viewers[0].User.Name != "Alice" {
		t.Errorf("viewers = %+v", got.Viewers)
	}
	v2JSON(t, db, "/api/v2/episodes/e1", &got)
	if got.Item.Prev != nil || got.Item.Next == nil || got.Item.Next.ID != "e2" {
		t.Errorf("e1 prev/next = %+v %+v", got.Item.Prev, got.Item.Next)
	}
	if w := v2Get(t, db, "/api/v2/episodes/nope"); w.Code != http.StatusNotFound {
		t.Errorf("episode nope: %d", w.Code)
	}
}

func TestV2Music(t *testing.T) {
	db := v2Seeded(t)
	var artist struct {
		Item struct {
			Name           string
			Albums, Tracks int
			Genres         []string
		}
		Totals struct {
			Listens, Listeners int
			WatchSeconds       int64
			SkipRate           *float64
		}
		TopTracks []struct {
			ID, AlbumID   string
			Views, Clicks int
			SkipRate      *float64
		}
		Albums []struct {
			ID              string
			Tracks, Listens int
		}
		SkipByHour []float64
	}
	v2JSON(t, db, "/api/v2/artists/ar1", &artist)
	if artist.Item.Name != "The Band" || artist.Item.Albums != 1 || artist.Item.Tracks != 2 {
		t.Errorf("artist item = %+v", artist.Item)
	}
	if artist.Totals.Listens != 1 || artist.Totals.Listeners != 1 || artist.Totals.WatchSeconds != 210 || !approx(artist.Totals.SkipRate, 0.5) {
		t.Errorf("artist totals = %+v", artist.Totals)
	}
	if len(artist.TopTracks) != 2 || artist.TopTracks[0].ID != "t1" || artist.TopTracks[0].AlbumID != "al1" || !approx(artist.TopTracks[1].SkipRate, 1) {
		t.Errorf("topTracks = %+v", artist.TopTracks)
	}
	if len(artist.Albums) != 1 || artist.Albums[0].Tracks != 2 || artist.Albums[0].Listens != 1 {
		t.Errorf("albums = %+v", artist.Albums)
	}
	if len(artist.SkipByHour) != 24 || artist.SkipByHour[8] != 0.5 {
		t.Errorf("skipByHour = %v", artist.SkipByHour)
	}

	var album struct {
		Item struct {
			ArtistID, ArtistName string
			Tracks               int
			RuntimeSeconds       int64
		}
		Totals struct {
			Listens, FullAlbumListens, Listeners int
			FullListenRatio, SkipRate            *float64
		}
		Tracks []struct {
			ID            string
			Number        int
			Views, Clicks int
			SkipRate      *float64
		}
	}
	v2JSON(t, db, "/api/v2/albums/al1", &album)
	if album.Item.ArtistName != "The Band" || album.Item.Tracks != 2 || album.Item.RuntimeSeconds != 400 {
		t.Errorf("album item = %+v", album.Item)
	}
	if album.Totals.Listens != 1 || album.Totals.FullAlbumListens != 0 || !approx(album.Totals.FullListenRatio, 1) || !approx(album.Totals.SkipRate, 0.5) {
		t.Errorf("album totals = %+v", album.Totals)
	}
	if len(album.Tracks) != 2 || album.Tracks[0].ID != "t1" || album.Tracks[1].Clicks != 1 || !approx(album.Tracks[1].SkipRate, 1) {
		t.Errorf("album tracks = %+v", album.Tracks)
	}

	var track struct {
		Item   struct{ AlbumID string }
		Totals struct {
			Views, Clicks int
			SkipRate      *float64
		}
		Recent []struct {
			User         struct{ Name string }
			WatchSeconds int64
		}
	}
	v2JSON(t, db, "/api/v2/tracks/t2", &track)
	if track.Item.AlbumID != "al1" || track.Totals.Views != 0 || track.Totals.Clicks != 1 || !approx(track.Totals.SkipRate, 1) || len(track.Recent) != 1 || track.Recent[0].User.Name != "Bob" {
		t.Errorf("track = %+v", track)
	}
	for _, path := range []string{"/api/v2/artists/x", "/api/v2/albums/m1", "/api/v2/tracks/x"} {
		if w := v2Get(t, db, path); w.Code != http.StatusNotFound {
			t.Errorf("%s: %d", path, w.Code)
		}
	}
}
