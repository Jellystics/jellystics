package handler_test

import (
	"net/http"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
)

func TestV2Search(t *testing.T) {
	db := v2Seeded(t)
	type res struct {
		Items []struct {
			ID, Kind, Name       string
			Year                 *int
			ParentID, ParentName *string
		}
		Users []struct{ ID, Name string }
	}
	var got res
	v2JSON(t, db, "/api/v2/search?q=INC", &got)
	if len(got.Items) != 1 || got.Items[0].ID != "m1" || got.Items[0].Kind != "movie" || got.Items[0].Year == nil || got.Items[0].ParentID != nil {
		t.Errorf("q=inc: %+v", got)
	}
	v2JSON(t, db, "/api/v2/search?q=open", &got)
	if len(got.Items) != 1 || got.Items[0].Kind != "track" || *got.Items[0].ParentID != "al1" || *got.Items[0].ParentName != "Blue Album" {
		t.Errorf("q=open: %+v", got.Items)
	}
	v2JSON(t, db, "/api/v2/search?q=band", &got)
	if len(got.Items) != 1 || got.Items[0].Kind != "artist" {
		t.Errorf("q=band: %+v", got.Items)
	}
	v2JSON(t, db, "/api/v2/search?q=ali", &got)
	if len(got.Users) != 1 || got.Users[0].Name != "Alice" || len(got.Items) != 0 {
		t.Errorf("q=ali: %+v", got)
	}
	v2JSON(t, db, "/api/v2/search?q=pi", &got)
	if len(got.Items) != 1 || got.Items[0].ID != "e1" || *got.Items[0].ParentID != "s1" {
		t.Errorf("q=pi: %+v", got.Items)
	}
	// Prefix matches first ("Closing"), then top-level kinds ("Inception" before the episode "Second").
	v2JSON(t, db, "/api/v2/search?q=c", &got)
	if len(got.Items) != 3 || got.Items[0].ID != "t2" || got.Items[1].ID != "m1" || got.Items[2].ID != "e2" {
		t.Errorf("q=c: %+v", got.Items)
	}
	v2JSON(t, db, "/api/v2/search?q=_", &got)
	if len(got.Items) != 0 || len(got.Users) != 0 {
		t.Errorf("q=_ must be literal: %+v", got)
	}
	v2JSON(t, db, "/api/v2/search", &got)
	if got.Items == nil || len(got.Items) != 0 {
		t.Errorf("empty q: %+v", got)
	}
}

func TestV2Users(t *testing.T) {
	db := v2Seeded(t)
	var got []struct {
		ID, Name     string
		WatchSeconds int64
		Views        int
		ByType       struct{ Movie, Series, Music int64 }
		LastActivity *string
		Trend        []int64
	}
	v2JSON(t, db, "/api/v2/users", &got)
	if len(got) != 2 || got[0].Name != "Alice" || got[0].WatchSeconds != 340 || got[0].Views != 4 ||
		got[0].ByType.Movie != 100 || got[0].ByType.Series != 240 || got[1].ByType.Music != 210 || got[0].LastActivity == nil {
		t.Fatalf("users = %+v", got)
	}
	var sum int64
	for _, v := range got[0].Trend {
		sum += v
	}
	if len(got[0].Trend) != 14 || sum != 340 {
		t.Errorf("trend = %v", got[0].Trend)
	}
	v2JSON(t, db, "/api/v2/users?type=music", &got)
	if got[0].Name != "Bob" || got[0].WatchSeconds != 210 || got[1].WatchSeconds != 0 {
		t.Errorf("type=music: %+v", got)
	}
}

func TestV2User(t *testing.T) {
	db := v2Seeded(t)
	if err := db.Create(&models.JFLibraryEpisode{Id: "e4", Name: sp("Finale"), SeriesId: sp("s1"), ParentIndexNumber: ip(2), IndexNumber: ip(2), RunTimeTicks: i64(100 * secTicks)}).Error; err != nil {
		t.Fatal(err)
	}
	var got struct {
		ID, Name       string
		MemberSince    *string
		FavoriteDevice *string
		Totals         v2Totals
		Previous       *v2Totals
		Calendar       []struct {
			Date         string
			WatchSeconds int64
		}
		InProgress []struct {
			SeriesID, NextID        string
			NextSeason, NextEpisode int
			Progress                float64
		}
		Hours   []int
		Profile struct{ Regularity, Discovery, Concentration, Completion *float64 }
		Genres  struct {
			Movie, Series, Music []struct {
				Name         string
				WatchSeconds int64
			}
		}
		Devices []struct {
			Device, Client, Version string
			Views                   int
		}
		Streak struct{ Current, Best int }
	}
	v2JSON(t, db, "/api/v2/users/u1?compare=1", &got)
	if got.Name != "Alice" || got.MemberSince == nil || got.FavoriteDevice == nil || *got.FavoriteDevice != "Chrome" {
		t.Errorf("header = %+v", got)
	}
	if got.Totals.Views != 4 || got.Totals.WatchSeconds != 340 || got.Previous == nil {
		t.Errorf("totals = %+v prev = %+v", got.Totals, got.Previous)
	}
	var calSum int64
	for _, d := range got.Calendar {
		calSum += d.WatchSeconds
	}
	if len(got.Calendar) != 365 || calSum != 340 {
		t.Errorf("calendar = %d days, sum %d", len(got.Calendar), calSum)
	}
	if len(got.InProgress) != 1 || got.InProgress[0].NextID != "e4" || got.InProgress[0].NextSeason != 2 || got.InProgress[0].NextEpisode != 2 || got.InProgress[0].Progress != 0.75 {
		t.Errorf("inProgress = %+v", got.InProgress)
	}
	if len(got.Hours) != 24 || got.Hours[21] != 3 || got.Hours[20] != 1 {
		t.Errorf("hours = %v", got.Hours)
	}
	pr := got.Profile
	if !approx(pr.Regularity, 0.1) || !approx(pr.Discovery, 0.75) || !approx(pr.Concentration, 1) || !approx(pr.Completion, 0.75) {
		t.Errorf("profile = %v %v %v %v", *pr.Regularity, *pr.Discovery, *pr.Concentration, *pr.Completion)
	}
	if len(got.Genres.Movie) != 1 || got.Genres.Movie[0].Name != "Sci-Fi" || got.Genres.Series[0].WatchSeconds != 240 || len(got.Genres.Music) != 0 {
		t.Errorf("genres = %+v", got.Genres)
	}
	if len(got.Devices) != 1 || got.Devices[0].Version != "10.9" || got.Devices[0].Views != 4 {
		t.Errorf("devices = %+v", got.Devices)
	}
	if got.Streak.Best != 3 || got.Streak.Current != 3 {
		t.Errorf("streak = %+v", got.Streak)
	}
	if w := v2Get(t, db, "/api/v2/users/nope"); w.Code != http.StatusNotFound {
		t.Errorf("user nope: %d", w.Code)
	}
}

func TestV2Libraries(t *testing.T) {
	db := v2Seeded(t)
	type card struct {
		ID, Name, Type string
		Items          int
		SizeBytes      int64
		Totals         v2Totals
	}
	var cards []card
	v2JSON(t, db, "/api/v2/libraries", &cards)
	if len(cards) != 3 || cards[0].ID != "lib-m" || cards[0].Type != "movies" || cards[0].Items != 2 || cards[0].SizeBytes != 4_000_000_000 ||
		cards[0].Totals.Views != 2 || cards[2].ID != "lib-s" || cards[2].Totals.Views != 3 {
		t.Fatalf("cards = %+v", cards)
	}

	var lib struct {
		Library card
		Items   struct {
			Rows []struct {
				ID, Kind       string
				Views, Viewers int
				Retention      *float64
				Trend          []int64
			}
			Total int
		}
		Genres    []struct{ Name string }
		Methods   struct{ Direct, Stream, Transcode int }
		Unwatched struct {
			Count int
			List  []struct{ ID string }
		}
		TimeToWatch struct{ Count int }
	}
	v2JSON(t, db, "/api/v2/libraries/lib-m", &lib)
	if lib.Library.Name != "Films" || lib.Items.Total != 2 || lib.Items.Rows[0].ID != "m1" || lib.Items.Rows[0].Views != 2 || lib.Items.Rows[0].Viewers != 2 ||
		len(lib.Items.Rows[0].Trend) != 14 || lib.Items.Rows[0].Retention != nil {
		t.Errorf("lib-m items = %+v", lib.Items)
	}
	if len(lib.Genres) != 1 || lib.Genres[0].Name != "Sci-Fi" || lib.Methods.Direct != 1 || lib.Methods.Transcode != 1 {
		t.Errorf("genres = %+v methods = %+v", lib.Genres, lib.Methods)
	}
	if lib.Unwatched.Count != 1 || len(lib.Unwatched.List) != 1 || lib.Unwatched.List[0].ID != "m2" {
		t.Errorf("unwatched = %+v", lib.Unwatched)
	}
	v2JSON(t, db, "/api/v2/libraries/lib-m?sort=name&q=un", &lib)
	if lib.Items.Total != 1 || lib.Items.Rows[0].ID != "m2" {
		t.Errorf("q=un: %+v", lib.Items)
	}
	v2JSON(t, db, "/api/v2/libraries/lib-s", &lib)
	if len(lib.Items.Rows) != 1 || lib.Items.Rows[0].Kind != "series" || !approx(lib.Items.Rows[0].Retention, 1) {
		t.Errorf("lib-s items = %+v", lib.Items)
	}
	if w := v2Get(t, db, "/api/v2/libraries/nope"); w.Code != http.StatusNotFound {
		t.Errorf("library nope: %d", w.Code)
	}
}
