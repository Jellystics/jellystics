package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// A skip is a click that did not become a view (< 30 s listened);
// skipRate = skips / clicks (NULL when no click).
const aggSkipRate = `(COUNT(*) FILTER (WHERE NOT is_view)::float8 / NULLIF(COUNT(*), 0))`

type trackStat struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	AlbumID   string   `json:"albumId"`
	AlbumName string   `json:"albumName"`
	Views     int      `json:"views"`
	Clicks    int      `json:"clicks"`
	SkipRate  *float64 `json:"skipRate"`
	Listeners int      `json:"listeners"`
}

// ---------------------------------------------------------------------------
// GET /api/v2/artists/:id
// ---------------------------------------------------------------------------

func (h *V2Handler) Artist(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var item struct {
		ID        string
		Name      *string
		Genres    jsonOf[[]string]
		Albums    int
		Tracks    int
		LibraryID *string
	}
	if v2Fail(c, h.raw(c, &item, `
		SELECT ar."Id" AS id, ar."Name" AS name, ar."Genres"::text AS genres,
		       COALESCE(ar."LibraryId", (SELECT t."LibraryId" FROM jf_music_tracks t WHERE t."ArtistId" = ar."Id" AND t."LibraryId" IS NOT NULL
		                                 GROUP BY 1 ORDER BY COUNT(*) DESC, 1 LIMIT 1)) AS library_id,
		       (SELECT COUNT(*) FROM (
		          SELECT i."Id" FROM jf_library_items i WHERE i."Type" = 'MusicAlbum' AND i."ArtistId" = ar."Id" AND NOT i.archived
		          UNION SELECT t."AlbumId" FROM jf_music_tracks t WHERE t."ArtistId" = ar."Id" AND t."AlbumId" IS NOT NULL AND NOT t.archived
		        ) al)::int AS albums,
		       (SELECT COUNT(*) FROM jf_music_tracks t WHERE t."ArtistId" = ar."Id" AND NOT t.archived)::int AS tracks
		FROM jf_music_artists ar WHERE ar."Id" = ?`, id)) {
		return
	}
	if item.ID == "" {
		v2NotFound(c)
		return
	}
	lib, err := h.libraryRefOf(c, item.LibraryID)
	if v2Fail(c, err) {
		return
	}
	f := p.entityFilter().where(`a."MediaType" = 'Audio' AND (ct."ArtistId" = ? OR (ct."ArtistId" IS NULL AND cal."ArtistId" = ?))`, id, id)

	var totals struct {
		Listens      int      `json:"listens"`
		WatchSeconds int64    `json:"watchSeconds"`
		Listeners    int      `json:"listeners"`
		SkipRate     *float64 `json:"skipRate"`
	}
	q, args := f.sql(`
		SELECT ` + aggPlays + ` AS listens, ` + aggSeconds + ` AS watch_seconds,
		       COUNT(DISTINCT user_id) FILTER (WHERE is_view)::int AS listeners, ` + aggSkipRate + ` AS skip_rate
		FROM acts`)
	if v2Fail(c, h.raw(c, &totals, q, args...)) {
		return
	}
	topTracks := []trackStat{}
	q, args = f.sql(`
		SELECT item_id AS id, COALESCE(MAX(item_name), '') AS name, COALESCE(MAX(group_id), '') AS album_id,
		       COALESCE(MAX(group_name), '') AS album_name, ` + aggPlays + ` AS views, ` + aggClicks + ` AS clicks,
		       ` + aggSkipRate + ` AS skip_rate, COUNT(DISTINCT user_id) FILTER (WHERE is_view)::int AS listeners
		FROM acts WHERE item_id IS NOT NULL
		GROUP BY item_id ORDER BY views DESC, clicks DESC, item_id LIMIT 20`)
	if v2Fail(c, h.raw(c, &topTracks, q, args...)) {
		return
	}
	type albumRow struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Year    *int   `json:"year"`
		Tracks  int    `json:"tracks"`
		Listens int    `json:"listens"`
	}
	albums := []albumRow{}
	q, args = f.sql(`,
		listens AS (SELECT group_id, `+aggPlays+` AS listens FROM acts GROUP BY group_id)
		SELECT i."Id" AS id, COALESCE(i."Name", '') AS name, i."ProductionYear" AS year,
		       (SELECT COUNT(*) FROM jf_music_tracks t WHERE t."AlbumId" = i."Id" AND NOT t.archived)::int AS tracks,
		       COALESCE(l.listens, 0) AS listens
		FROM jf_library_items i LEFT JOIN listens l ON l.group_id = i."Id"
		WHERE i."Type" = 'MusicAlbum' AND NOT i.archived
		  AND (i."ArtistId" = ? OR i."Id" IN (SELECT t."AlbumId" FROM jf_music_tracks t WHERE t."ArtistId" = ?))
		ORDER BY i."ProductionYear" DESC NULLS LAST, i."Name"`, id, id)
	if v2Fail(c, h.raw(c, &albums, q, args...)) {
		return
	}
	var hours []struct {
		Hour     int
		SkipRate float64
	}
	q, args = f.sql(`SELECT EXTRACT(HOUR FROM local_ts)::int AS hour, ` + aggSkipRate + ` AS skip_rate FROM acts GROUP BY 1`)
	if v2Fail(c, h.raw(c, &hours, q, args...)) {
		return
	}
	skipByHour := make([]float64, 24)
	for _, r := range hours {
		if r.Hour >= 0 && r.Hour < 24 {
			skipByHour[r.Hour] = r.SkipRate
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"item":       gin.H{"id": item.ID, "name": strOrEmpty(item.Name), "genres": strs(item.Genres), "albums": item.Albums, "tracks": item.Tracks, "library": lib},
		"totals":     totals,
		"topTracks":  topTracks,
		"albums":     albums,
		"skipByHour": skipByHour,
	})
}

// ---------------------------------------------------------------------------
// GET /api/v2/albums/:id
// ---------------------------------------------------------------------------

func (h *V2Handler) Album(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var item struct {
		ID           string
		Name         *string
		ArtistID     *string
		ArtistName   *string
		Year         *int
		Tracks       int
		RuntimeTicks int64
		LibraryID    *string
	}
	if v2Fail(c, h.raw(c, &item, `
		SELECT i."Id" AS id, i."Name" AS name, i."ArtistId" AS artist_id, i."ParentId" AS library_id,
		       COALESCE(ar."Name", i."AlbumArtist") AS artist_name, i."ProductionYear" AS year,
		       (SELECT COUNT(*) FROM jf_music_tracks t WHERE t."AlbumId" = i."Id" AND NOT t.archived)::int AS tracks,
		       (SELECT COALESCE(SUM(t."RunTimeTicks"), 0) FROM jf_music_tracks t WHERE t."AlbumId" = i."Id" AND NOT t.archived)::bigint AS runtime_ticks
		FROM jf_library_items i LEFT JOIN jf_music_artists ar ON ar."Id" = i."ArtistId"
		WHERE i."Id" = ? AND i."Type" = 'MusicAlbum'`, id)) {
		return
	}
	if item.ID == "" {
		v2NotFound(c)
		return
	}
	lib, err := h.libraryRefOf(c, item.LibraryID)
	if v2Fail(c, err) {
		return
	}
	f := p.entityFilter().where(`a."MediaType" = 'Audio' AND COALESCE(ct."AlbumId", a."NowPlayingItemId") = ?`, id)

	var totals struct {
		Listens          int      `json:"listens"`
		FullListenRatio  *float64 `json:"fullListenRatio"`
		FullAlbumListens int      `json:"fullAlbumListens"`
		Listeners        int      `json:"listeners"`
		SkipRate         *float64 `json:"skipRate"`
	}
	// fullAlbumListens: (user, local day) on which every catalog track of the
	// album (non-archived catalog tracks only) was listened to.
	q, args := f.sql(`,
		ud AS (
		  SELECT user_id, local_ts::date AS day, COUNT(DISTINCT item_id) AS n FROM acts
		  WHERE is_view AND item_id IN (SELECT t."Id" FROM jf_music_tracks t WHERE t."AlbumId" = ? AND NOT t.archived)
		  GROUP BY 1, 2
		)
		SELECT `+aggPlays+` AS listens,
		       (COUNT(*) FILTER (WHERE is_view AND `+ratioExpr+` >= ?)::float8 / NULLIF(COUNT(*) FILTER (WHERE is_view), 0)) AS full_listen_ratio,
		       (SELECT COUNT(*) FROM ud WHERE ? > 0 AND n >= ?)::int AS full_album_listens,
		       COUNT(DISTINCT user_id) FILTER (WHERE is_view)::int AS listeners, `+aggSkipRate+` AS skip_rate
		FROM acts`, id, completionThreshold(), item.Tracks, item.Tracks)
	if v2Fail(c, h.raw(c, &totals, q, args...)) {
		return
	}
	type albumTrack struct {
		ID             string   `json:"id"`
		Number         *int     `json:"number"`
		Name           string   `json:"name"`
		RuntimeSeconds int64    `json:"runtimeSeconds"`
		Views          int      `json:"views"`
		Clicks         int      `json:"clicks"`
		SkipRate       *float64 `json:"skipRate"`
		Listeners      int      `json:"listeners"`
	}
	tracks := []albumTrack{}
	q, args = f.sql(`,
		ts AS (
		  SELECT item_id, `+aggPlays+` AS views, `+aggClicks+` AS clicks, `+aggSkipRate+` AS skip_rate,
		         COUNT(DISTINCT user_id) FILTER (WHERE is_view)::int AS listeners
		  FROM acts GROUP BY item_id
		)
		SELECT t."Id" AS id, t."IndexNumber" AS number, COALESCE(t."Name", '') AS name,
		       (COALESCE(t."RunTimeTicks", 0) / 10000000)::bigint AS runtime_seconds,
		       COALESCE(ts.views, 0) AS views, COALESCE(ts.clicks, 0) AS clicks, ts.skip_rate, COALESCE(ts.listeners, 0) AS listeners
		FROM jf_music_tracks t LEFT JOIN ts ON ts.item_id = t."Id"
		WHERE t."AlbumId" = ? AND NOT t.archived
		ORDER BY t."DiscNumber" NULLS FIRST, t."IndexNumber" NULLS LAST, t."Name"`, id)
	if v2Fail(c, h.raw(c, &tracks, q, args...)) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"item": gin.H{
			"id": item.ID, "name": strOrEmpty(item.Name), "artistId": item.ArtistID, "artistName": item.ArtistName,
			"year": item.Year, "tracks": item.Tracks, "runtimeSeconds": item.RuntimeTicks / 10_000_000,
			"library": lib,
		},
		"totals": totals,
		"tracks": tracks,
	})
}

// ---------------------------------------------------------------------------
// GET /api/v2/tracks/:id  (side panel)
// ---------------------------------------------------------------------------

func (h *V2Handler) Track(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var item struct {
		ID         string
		Name       *string
		AlbumID    *string
		AlbumName  *string
		ArtistID   *string
		ArtistName *string
	}
	if v2Fail(c, h.raw(c, &item, `
		SELECT t."Id" AS id, t."Name" AS name, t."AlbumId" AS album_id, COALESCE(al."Name", t."AlbumName") AS album_name,
		       t."ArtistId" AS artist_id, ar."Name" AS artist_name
		FROM jf_music_tracks t
		LEFT JOIN jf_library_items al ON al."Id" = t."AlbumId"
		LEFT JOIN jf_music_artists ar ON ar."Id" = t."ArtistId"
		WHERE t."Id" = ?`, id)) {
		return
	}
	if item.ID == "" {
		v2NotFound(c)
		return
	}
	f := p.entityFilter().where(`a."MediaType" = 'Audio' AND a."ItemId" = ?`, id)
	var totals struct {
		Views     int      `json:"views"`
		Clicks    int      `json:"clicks"`
		SkipRate  *float64 `json:"skipRate"`
		Listeners int      `json:"listeners"`
	}
	q, args := f.sql(`SELECT ` + aggPlays + ` AS views, ` + aggClicks + ` AS clicks, ` + aggSkipRate + ` AS skip_rate,
		COUNT(DISTINCT user_id) FILTER (WHERE is_view)::int AS listeners FROM acts`)
	if v2Fail(c, h.raw(c, &totals, q, args...)) {
		return
	}
	var rows []struct {
		UserID, UserName string
		StartedAt        time.Time
		WatchSeconds     int64
	}
	q, args = f.sql(`SELECT COALESCE(user_id, '') AS user_id, COALESCE(user_name, '') AS user_name, started_at, duration AS watch_seconds FROM acts ORDER BY started_at DESC LIMIT 20`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	type recent struct {
		User         refUser   `json:"user"`
		StartedAt    time.Time `json:"startedAt"`
		WatchSeconds int64     `json:"watchSeconds"`
	}
	out := make([]recent, len(rows))
	for i, r := range rows {
		out[i] = recent{refUser{r.UserID, r.UserName}, localTime(r.StartedAt), r.WatchSeconds}
	}
	c.JSON(http.StatusOK, gin.H{
		"item": gin.H{
			"id": item.ID, "name": strOrEmpty(item.Name), "albumId": item.AlbumID, "albumName": item.AlbumName,
			"artistId": item.ArtistID, "artistName": item.ArtistName,
		},
		"totals": totals,
		"recent": out,
	})
}
