package handler_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// seedRichStats builds a deterministic multi-library dataset (movie, TV and
// music libraries). Playback rows (PlaybackDuration is SECONDS):
//
//	Movie "Inception" (m1): p1, p2 Alice 600 s; p3 Bob 300 s
//	Episode "Pilot" (ep1 of series-1): p4 Alice 1200 s
//	Track "Song" (tr1 of album alb1): p5 Bob 180 s
func seedRichStats(t *testing.T, db *gorm.DB) {
	t.Helper()
	repos := repository.New(db)
	ctx := t.Context()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	action := datatypes.JSON([]byte(`["Action"]`))
	drama := datatypes.JSON([]byte(`["Drama"]`))
	rock := datatypes.JSON([]byte(`["Rock"]`))

	must(repos.User.Upsert(ctx, []models.JFUser{
		{Id: "u1", Name: sp("Alice")},
		{Id: "u2", Name: sp("Bob")},
	}))

	must(repos.Library.Upsert(ctx, []models.JFLibrary{
		{Id: "lib-movies", Name: sp("Movies"), CollectionType: sp("movies")},
		{Id: "lib-tv", Name: sp("Shows"), CollectionType: sp("tvshows")},
		{Id: "lib-music", Name: sp("Music"), CollectionType: sp("music")},
	}))

	movie := "Movie"
	series := "Series"
	album := "MusicAlbum"
	must(repos.Item.Upsert(ctx, []models.JFLibraryItem{
		{Id: "m1", Name: sp("Inception"), Type: &movie, ParentId: sp("lib-movies"),
			Genres: action, RunTimeTicks: i64(1200 * secTicks), ProductionYear: ip(2010)},
		{Id: "series-1", Name: sp("Lost"), Type: &series, ParentId: sp("lib-tv"), Genres: drama},
		{Id: "alb1", Name: sp("Greatest Hits"), Type: &album, ParentId: sp("lib-music"),
			AlbumArtist: sp("The Band"), ArtistId: sp("art1"), Genres: rock},
	}))
	must(repos.Season.Upsert(ctx, []models.JFLibrarySeason{
		{Id: "s1", Name: sp("Season 1"), SeriesId: sp("series-1")},
	}))
	must(repos.Episode.Upsert(ctx, []models.JFLibraryEpisode{
		{Id: "ep1", Name: sp("Pilot"), SeriesId: sp("series-1"), SeasonId: sp("s1"), RunTimeTicks: i64(1500 * secTicks)},
	}))

	must(repos.MusicArtist.Upsert(ctx, []models.JFMusicArtist{
		{Id: "art1", Name: sp("The Band"), LibraryId: sp("lib-music")},
	}))
	must(repos.MusicTrack.Upsert(ctx, []models.JFMusicTrack{
		{Id: "tr1", Name: sp("Song"), AlbumId: sp("alb1"), AlbumName: sp("Greatest Hits"),
			AlbumArtist: sp("The Band"), ArtistId: sp("art1"),
			LibraryId: sp("lib-music"), Genres: rock, RunTimeTicks: i64(240 * secTicks)},
	}))

	must(db.Create(&[]models.JFItemInfo{
		{Id: "m1", Size: i64(1000)},
		{Id: "ep1", Size: i64(500)},
		{Id: "tr1", Size: i64(50)},
	}).Error)

	d := testutil.TS(richSeedDay().Add(14 * time.Hour))
	dEp := testutil.TS(richSeedDay().Add(9 * time.Hour))
	must(repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{
		{Id: "p1", UserId: sp("u1"), UserName: sp("Alice"), NowPlayingItemId: sp("m1"), NowPlayingItemName: sp("Inception"),
			Client: sp("Web"), DeviceName: sp("Firefox"), PlayMethod: sp("DirectPlay"), PlaybackDuration: i64(600), ActivityDateInserted: &d, Source: "watchdog"},
		{Id: "p2", UserId: sp("u1"), UserName: sp("Alice"), NowPlayingItemId: sp("m1"), NowPlayingItemName: sp("Inception"),
			Client: sp("Web"), DeviceName: sp("Firefox"), PlayMethod: sp("DirectPlay"), PlaybackDuration: i64(600), ActivityDateInserted: &d, Source: "watchdog"},
		{Id: "p3", UserId: sp("u2"), UserName: sp("Bob"), NowPlayingItemId: sp("m1"), NowPlayingItemName: sp("Inception"),
			Client: sp("Android"), DeviceName: sp("Pixel"), PlayMethod: sp("Transcode"), PlaybackDuration: i64(300), ActivityDateInserted: &dEp, Source: "watchdog"},
		{Id: "p4", UserId: sp("u1"), UserName: sp("Alice"), NowPlayingItemId: sp("series-1"), NowPlayingItemName: sp("Lost"), SeriesName: sp("Lost"),
			EpisodeId: sp("ep1"), Client: sp("Web"), PlayMethod: sp("DirectPlay"), PlaybackDuration: i64(1200), ActivityDateInserted: &d, Source: "watchdog"},
		{Id: "p5", UserId: sp("u2"), UserName: sp("Bob"), NowPlayingItemId: sp("alb1"), NowPlayingItemName: sp("Greatest Hits"),
			EpisodeId: sp("tr1"), Client: sp("Android"), PlayMethod: sp("DirectPlay"), PlaybackDuration: i64(180), ActivityDateInserted: &dEp, Source: "watchdog"},
	}))
	testutil.BackfillTracking(t, db)
}

// richSeedDay is the (UTC) calendar day of every seedRichStats playback row.
func richSeedDay() time.Time { return testutil.RecentDay(3) }

func insertRows(t *testing.T, db *gorm.DB, rows ...models.JFPlaybackActivity) {
	t.Helper()
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("insert rows: %v", err)
	}
}

// seedBinge: binge = >= 3 distinct episodes, same series, same user, same
// local day. Only u1 qualifies.
func seedBinge(t *testing.T, db *gorm.DB) {
	t.Helper()
	series := "Series"
	if err := db.Create(&models.JFLibraryItem{Id: "show", Name: sp("The Show"), Type: &series, ParentId: sp("lib-tv"),
		Genres: datatypes.JSON([]byte(`["Drama"]`))}).Error; err != nil {
		t.Fatal(err)
	}
	var eps []models.JFLibraryEpisode
	for i := 1; i <= 4; i++ {
		eps = append(eps, models.JFLibraryEpisode{Id: fmt.Sprintf("e%d", i), Name: sp(fmt.Sprintf("Ep %d", i)),
			SeriesId: sp("show"), RunTimeTicks: i64(1800 * secTicks)})
	}
	if err := db.Create(&eps).Error; err != nil {
		t.Fatal(err)
	}
	ep := func(id, user, item string, start time.Time) models.JFPlaybackActivity {
		r := trackedRow(id, user, item, "Episode", start, 1500, 1800, nil)
		r.NowPlayingItemId = sp("show")
		r.SeriesName = sp("The Show")
		return r
	}
	day := testutil.RecentDay(4).Add(18 * time.Hour)
	insertRows(t, db,
		// u1: 3 distinct episodes on one day → 1 binge.
		ep("b1", "u1", "e1", day), ep("b2", "u1", "e2", day.Add(time.Hour)), ep("b3", "u1", "e3", day.Add(2*time.Hour)),
		// u2: the same episode 3 times → not a binge.
		ep("r1", "u2", "e1", day), ep("r2", "u2", "e1", day.Add(time.Hour)), ep("r3", "u2", "e1", day.Add(2*time.Hour)),
		// u3: 3 distinct episodes, but one is a 10 s click → not a binge.
		ep("c1", "u3", "e1", day), ep("c2", "u3", "e2", day.Add(time.Hour)),
		func() models.JFPlaybackActivity {
			r := trackedRow("c3", "u3", "e3", "Episode", day.Add(2*time.Hour), 10, 1800, nil)
			r.NowPlayingItemId = sp("show")
			return r
		}(),
	)
}

func gatherMetrics(t *testing.T, db *gorm.DB) map[string][]string {
	t.Helper()
	reg := prometheus.NewRegistry()
	reg.MustRegister(handler.NewMetricsCollector(repository.New(db), db))
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err) // duplicate label sets fail here
	}
	out := map[string][]string{}
	for _, mf := range mfs {
		for _, m := range mf.GetMetric() {
			var labels []string
			for _, l := range m.GetLabel() {
				labels = append(labels, l.GetName()+"="+l.GetValue())
			}
			v := m.GetCounter().GetValue() + m.GetGauge().GetValue()
			out[mf.GetName()] = append(out[mf.GetName()], fmt.Sprintf("%s %g", strings.Join(labels, ","), v))
		}
	}
	return out
}

func hasSample(samples []string, want string) bool {
	for _, s := range samples {
		if s == want {
			return true
		}
	}
	return false
}

func TestMetrics_RenameMusicCompletionBinge(t *testing.T) {
	db := testutil.NewDB(t)
	seedRichStats(t, db)
	// Alice was renamed: old rows carry the old name, jf_users the new one.
	if err := db.Exec(`UPDATE jf_playback_activity SET "UserName" = 'Alice (old)' WHERE "Id" = 'p1'`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`UPDATE jf_users SET "Name" = 'Alicia' WHERE "Id" = 'u1'`).Error; err != nil {
		t.Fatal(err)
	}
	// A different user id that shares the new name.
	insertRows(t, db, trackedRow("dup", "u9", "m1", "Movie", richSeedDay().Add(21*time.Hour), 600, 1200, nil))
	if err := db.Exec(`INSERT INTO jf_users ("Id", "Name") VALUES ('u9', 'Alicia')`).Error; err != nil {
		t.Fatal(err)
	}
	seedBinge(t, db)

	m := gatherMetrics(t, db)
	// u1: 3 rich-seed views + 3 binge episodes, under its current name only.
	if !hasSample(m["jellyfin_plays_by_user_total"], "user=Alicia 6") {
		t.Errorf("plays_by_user = %v, want user=Alicia 6 (one series per id)", m["jellyfin_plays_by_user_total"])
	}
	if !hasSample(m["jellyfin_plays_by_user_total"], "user=Alicia (2) 1") {
		t.Errorf("plays_by_user = %v, want deduped label for the second Alicia", m["jellyfin_plays_by_user_total"])
	}
	if !hasSample(m["jellyfin_plays_by_media_type_total"], "type=Audio 1") {
		t.Errorf("plays_by_media_type = %v, want type=Audio 1", m["jellyfin_plays_by_media_type_total"])
	}
	if !hasSample(m["jellyfin_binge_sessions_total"], " 1") {
		t.Errorf("binge_sessions = %v, want 1", m["jellyfin_binge_sessions_total"])
	}
	// Completion uses the played item's runtime: Inception (1200 s) watched
	// 600/600/300/600 s → abandoned; Pilot 1200/1500 s = 0.8 → abandoned
	// (threshold 0.9); Song 180/240 = 0.75 → abandoned; binge episodes
	// 1500/1800 = 0.83 → abandoned. Nothing reaches 90 %.
	if !hasSample(m["jellyfin_completed_plays_total"], " 0") {
		t.Errorf("completed = %v, want 0", m["jellyfin_completed_plays_total"])
	}
}
