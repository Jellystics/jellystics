package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/Jellystics/Jellystics/internal/ws"
	"gorm.io/gorm"
)

// mockJF is a scripted Jellyfin mock. Each field is a handler keyed by the
// endpoint the sync service hits. GetSessions consumes sessionScript one entry
// per call, so a test can drive SessionTick across a sequence of ticks.
type mockJF struct {
	server        *httptest.Server
	sessionScript [][]jellyfin.SessionInfo
	sessionIdx    int

	users     []jellyfin.User
	libraries []jellyfin.Library
	// items maps IncludeItemTypes (comma-joined) -> items returned for /Items.
	items map[string][]jellyfin.Item
	// userData maps item id -> user data returned by GET /Items/{id}.
	userData map[string]jellyfin.UserItemData

	// Plugins: jellysticsPlugin answers /Jellystics/Info (else 404);
	// pbrRows is what the Playback Reporting custom query returns (the
	// official plugin is listed in /Plugins when non-nil).
	jellysticsPlugin bool
	jellysticsSchema int // 0 = 1
	pbrRows          [][]any
	pbrQueries       int
}

func newMockJF(t *testing.T) *mockJF {
	t.Helper()
	m := &mockJF{items: map[string][]jellyfin.Item{}, userData: map[string]jellyfin.UserItemData{}}
	mux := http.NewServeMux()

	mux.HandleFunc("/Sessions", func(w http.ResponseWriter, r *http.Request) {
		var out []jellyfin.SessionInfo
		if m.sessionIdx < len(m.sessionScript) {
			out = m.sessionScript[m.sessionIdx]
			m.sessionIdx++
		}
		writeJSON(w, out)
	})

	mux.HandleFunc("/Users", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, m.users)
	})

	// /Users/{id}/Views for GetLibraries.
	mux.HandleFunc("/Users/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/Views") {
			writeJSON(w, jellyfin.LibrariesResponse{Items: m.libraries})
			return
		}
		writeJSON(w, jellyfin.ItemsResponse{})
	})

	mux.HandleFunc("/Items", func(w http.ResponseWriter, r *http.Request) {
		types := r.URL.Query().Get("IncludeItemTypes")
		items := m.items[types]
		writeJSON(w, jellyfin.ItemsResponse{
			Items:            items,
			TotalRecordCount: len(items),
			StartIndex:       0,
		})
	})

	mux.HandleFunc("/Items/", func(w http.ResponseWriter, r *http.Request) {
		ud, ok := m.userData[strings.TrimPrefix(r.URL.Path, "/Items/")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]any{"UserData": ud})
	})

	mux.HandleFunc("/Jellystics/Info", func(w http.ResponseWriter, r *http.Request) {
		if !m.jellysticsPlugin {
			http.NotFound(w, r)
			return
		}
		schema := m.jellysticsSchema
		if schema == 0 {
			schema = 1
		}
		writeJSON(w, map[string]any{"pluginVersion": "1.0.0.0", "schemaVersion": schema})
	})

	mux.HandleFunc("/Plugins", func(w http.ResponseWriter, r *http.Request) {
		out := []jellyfin.Plugin{}
		if m.pbrRows != nil {
			out = append(out, jellyfin.Plugin{Name: "Playback Reporting", ConfigurationFileName: sp("Jellyfin.Plugin.PlaybackReporting.xml")})
		}
		writeJSON(w, out)
	})

	mux.HandleFunc("/user_usage_stats/submit_custom_query", func(w http.ResponseWriter, r *http.Request) {
		m.pbrQueries++
		writeJSON(w, map[string]any{"results": m.pbrRows})
	})

	m.server = httptest.NewServer(mux)
	t.Cleanup(m.server.Close)
	return m
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// newTestService wires a real repository container against a fresh throwaway DB,
// seeds app_config.JF_HOST to the mock server URL (as production does), and
// returns a sync.Service that will talk to the mock via refreshClient.
func newTestService(t *testing.T, mock *mockJF) (*Service, *gorm.DB) {
	t.Helper()
	db := testutil.NewDB(t)
	repos := repository.New(db)

	// Point the DB config at the mock server so refreshClient targets it.
	host := mock.server.URL
	if err := db.Exec(`UPDATE app_config SET "JF_HOST" = ? WHERE "ID" = 1`, host).Error; err != nil {
		t.Fatalf("seed JF_HOST: %v", err)
	}

	// A client is still required by New(); refreshClient overwrites it, but pass a
	// correct one anyway to mirror production bootstrap.
	jf := jellyfin.NewClient(host, "")
	svc := New(repos, jf, ws.NewHub())
	return svc, db
}

// rewindTick moves the watchdog LastTickAt for a session back by `secs` seconds
// so the next SessionTick sees a wall-clock gap. In production this gap is real
// (~10s between ticks); in a test the two ticks happen microseconds apart, so we
// simulate elapsed time explicitly. This does NOT touch the position/pause
// accounting — it only supplies the wall-clock the increment is clamped to.
func rewindTick(t *testing.T, db *gorm.DB, sessionID string, secs float64) {
	t.Helper()
	err := db.Exec(
		`UPDATE jf_activity_watchdog SET "LastTickAt" = "LastTickAt" - (? * interval '1 second'), "StartedAt" = "StartedAt" - (? * interval '1 second') WHERE "Id" = ?`,
		secs,
		secs, sessionID,
	).Error
	if err != nil {
		t.Fatalf("rewind tick: %v", err)
	}
}

// --- test data builders ---

func movieSession(sessionID, itemID, name string, posSeconds int64, paused bool) jellyfin.SessionInfo {
	pos := posSeconds * ticksPerSecond
	return jellyfin.SessionInfo{
		Id:       sessionID,
		UserId:   "user-1",
		UserName: "alice",
		Client:   "Jellyfin Web",
		NowPlayingItem: &jellyfin.SessionItem{
			Id:   itemID,
			Name: name,
			Type: "Movie",
		},
		PlayState: &jellyfin.PlayState{PositionTicks: &pos, IsPaused: paused},
	}
}

func musicSession(sessionID, albumID, trackID, trackName, albumName string, posSeconds int64) jellyfin.SessionInfo {
	pos := posSeconds * ticksPerSecond
	album := albumName
	return jellyfin.SessionInfo{
		Id:       sessionID,
		UserId:   "user-1",
		UserName: "alice",
		Client:   "Jellyfin Web",
		NowPlayingItem: &jellyfin.SessionItem{
			Id:      trackID,
			Name:    trackName,
			Type:    "Audio",
			AlbumId: &albumID,
			Album:   &album,
		},
		PlayState: &jellyfin.PlayState{PositionTicks: &pos, IsPaused: false},
	}
}

// TestSyncMusicLibrary_Mapping proves the item-sync path upserts artists, albums,
// and tracks with the correct table mapping: MusicAlbum → jf_library_items,
// Audio → jf_music_tracks (with AlbumId/ArtistId linkage).
func TestSyncMusicLibrary_Mapping(t *testing.T) {
	mock := newMockJF(t)
	mock.items["MusicArtist"] = []jellyfin.Item{
		{Id: "artist-1", Name: "The Band", Type: "MusicArtist", Genres: []string{"rock"}},
	}
	mock.items["MusicAlbum"] = []jellyfin.Item{
		{Id: "album-1", Name: "Debut", Type: "MusicAlbum"},
	}
	mock.items["Audio"] = []jellyfin.Item{
		{
			Id: "track-1", Name: "Opener", Type: "Audio",
			AlbumId:      sp("album-1"),
			Album:        sp("Debut"),
			AlbumArtist:  sp("The Band"),
			AlbumArtists: []jellyfin.NamedItem{{Id: "artist-1", Name: "The Band"}},
			IndexNumber:  intp(1),
			RunTimeTicks: i64p(2000000000),
			MediaSources: []jellyfin.MediaSource{{Path: "/music/opener.flac", Size: i64p(9000)}},
		},
	}
	svc, db := newTestService(t, mock)
	ctx := context.Background()

	if err := svc.SyncMusicLibrary(ctx, "lib-music"); err != nil {
		t.Fatalf("SyncMusicLibrary: %v", err)
	}

	// Artist upserted.
	var artist models.JFMusicArtist
	if err := db.Where(`"Id" = ?`, "artist-1").First(&artist).Error; err != nil {
		t.Fatalf("artist not upserted: %v", err)
	}

	// Album upserted into jf_library_items.
	var album models.JFLibraryItem
	if err := db.Where(`"Id" = ?`, "album-1").First(&album).Error; err != nil {
		t.Fatalf("album not upserted into jf_library_items: %v", err)
	}
	if album.Type == nil || *album.Type != "MusicAlbum" {
		t.Fatalf("album Type = %v, want MusicAlbum", album.Type)
	}

	// Track upserted into jf_music_tracks with album/artist linkage.
	var track models.JFMusicTrack
	if err := db.Where(`"Id" = ?`, "track-1").First(&track).Error; err != nil {
		t.Fatalf("track not upserted: %v", err)
	}
	if track.AlbumId == nil || *track.AlbumId != "album-1" {
		t.Fatalf("track AlbumId = %v, want album-1", track.AlbumId)
	}
	if track.ArtistId == nil || *track.ArtistId != "artist-1" {
		t.Fatalf("track ArtistId = %v, want artist-1 (first album artist)", track.ArtistId)
	}
}

// TestSyncTVLibrary_Mapping proves series/seasons/episodes are upserted into their
// respective tables with correct series/season linkage.
func TestSyncTVLibrary_Mapping(t *testing.T) {
	mock := newMockJF(t)
	mock.items["Series"] = []jellyfin.Item{
		{Id: "series-1", Name: "Lost", Type: "Series"},
	}
	mock.items["Season"] = []jellyfin.Item{
		{Id: "season-1", Name: "Season 1", Type: "Season", SeriesId: sp("series-1"), IndexNumber: intp(1)},
	}
	mock.items["Episode"] = []jellyfin.Item{
		{
			Id: "ep-1", Name: "Pilot", Type: "Episode",
			SeriesId: sp("series-1"), SeasonId: sp("season-1"),
			SeriesName: sp("Lost"), IndexNumber: intp(1),
			MediaSources: []jellyfin.MediaSource{{Path: "/tv/pilot.mkv", Size: i64p(500)}},
		},
	}
	svc, db := newTestService(t, mock)
	ctx := context.Background()

	if err := svc.SyncTVLibrary(ctx, "lib-tv"); err != nil {
		t.Fatalf("SyncTVLibrary: %v", err)
	}

	var series models.JFLibraryItem
	if err := db.Where(`"Id" = ?`, "series-1").First(&series).Error; err != nil {
		t.Fatalf("series not upserted: %v", err)
	}
	var season models.JFLibrarySeason
	if err := db.Where(`"Id" = ?`, "season-1").First(&season).Error; err != nil {
		t.Fatalf("season not upserted: %v", err)
	}
	if season.SeriesId == nil || *season.SeriesId != "series-1" {
		t.Fatalf("season SeriesId = %v, want series-1", season.SeriesId)
	}
	var ep models.JFLibraryEpisode
	if err := db.Where(`"Id" = ?`, "ep-1").First(&ep).Error; err != nil {
		t.Fatalf("episode not upserted: %v", err)
	}
	if ep.SeriesId == nil || *ep.SeriesId != "series-1" {
		t.Fatalf("episode SeriesId = %v, want series-1", ep.SeriesId)
	}
	if ep.SeasonId == nil || *ep.SeasonId != "season-1" {
		t.Fatalf("episode SeasonId = %v, want season-1", ep.SeasonId)
	}
}
