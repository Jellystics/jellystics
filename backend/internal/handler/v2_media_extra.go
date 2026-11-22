package handler

// Extra entity-page widgets (phase 4): sittings to finish a movie, decades
// watched vs catalog and growth vs consumption of a library, albums listened
// to in full, paged history of a series. Mounted by registerMediaExtra on the /api/v2 group.

import (
	"net/http"
	"sort"
	"time"

	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
)

func (h *V2Handler) registerMediaExtra(v2 *gin.RouterGroup) {
	v2.GET("/movies/:id/sittings", h.MovieSittings)
	v2.GET("/libraries/:id/decades", h.LibraryDecades)
	v2.GET("/libraries/:id/growth", h.LibraryGrowth)
	v2.GET("/albums/:id/full-listens", h.AlbumFullListens)
	v2.GET("/series/:id/history", h.SeriesHistory)
}

// ---------------------------------------------------------------------------
// GET /api/v2/movies/:id/sittings
// ---------------------------------------------------------------------------

type sittingsBucket struct {
	Sittings int `json:"sittings"` // 1, 2, 3, 4 (= 4 or more)
	Count    int `json:"count"`
}

// MovieSittings counts, for every completion of the movie in the range, the
// number of viewing sittings (view rows of that user) it took since the user's
// previous completion: 1 = watched in one go. Earlier sittings may fall
// before the range; only the completing sitting must be inside it.
func (h *V2Handler) MovieSittings(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var exists int64
	if v2Fail(c, h.raw(c, &exists, `SELECT COUNT(*) FROM jf_library_items WHERE "Id" = ? AND "Type" = 'Movie'`, id)) {
		return
	}
	if exists == 0 {
		v2NotFound(c)
		return
	}
	var rows []struct{ Sittings, N int }
	q, args := acts().forUser(p.UserID).where(`a."MediaType" = 'Movie' AND a."ItemId" = ?`, id).sql(`,
		seq AS (
		  SELECT user_id, started_at, COALESCE(`+ratioExpr+` >= ?, false) AS done FROM acts WHERE is_view
		),
		seg AS (
		  SELECT user_id, started_at, done,
		         COALESCE(SUM(CASE WHEN done THEN 1 ELSE 0 END) OVER (
		           PARTITION BY user_id ORDER BY started_at ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING), 0) AS k
		  FROM seq
		),
		fin AS (
		  SELECT COUNT(*) AS sittings, MAX(started_at) FILTER (WHERE done) AS done_at
		  FROM seg GROUP BY user_id, k HAVING BOOL_OR(done)
		)
		SELECT LEAST(sittings, 4)::int AS sittings, COUNT(*)::int AS n
		FROM fin WHERE done_at >= ? AND done_at < ? GROUP BY 1`, completionThreshold(), p.Range.From, p.Range.To)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	out := []sittingsBucket{{1, 0}, {2, 0}, {3, 0}, {4, 0}}
	for _, r := range rows {
		if r.Sittings >= 1 && r.Sittings <= 4 {
			out[r.Sittings-1].Count = r.N
		}
	}
	c.JSON(http.StatusOK, out)
}

// libraryOf loads the library card fields needed by the library widgets;
// writes a 404 and returns false when the id is unknown.
func (h *V2Handler) libraryOf(c *gin.Context, id string) (itemType string, ok bool) {
	var card libraryCard
	if v2Fail(c, h.raw(c, &card, libraryCardsSQL+` AND l."Id" = ?`, id)) {
		return "", false
	}
	if card.ID == "" {
		v2NotFound(c)
		return "", false
	}
	return map[string]string{"movies": "Movie", "tvshows": "Series", "music": "MusicAlbum"}[card.Type], true
}

// ---------------------------------------------------------------------------
// GET /api/v2/libraries/:id/decades
// ---------------------------------------------------------------------------

type libDecadeRow struct {
	Decade  int `json:"decade"`
	Views   int `json:"views"`
	Catalog int `json:"catalog"`
}

// LibraryDecades compares, per production decade of the library titles
// (movies, series or albums), the views in the range with the catalog size.
// Titles without a production year are left out.
func (h *V2Handler) LibraryDecades(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	itemType, ok := h.libraryOf(c, id)
	if !ok {
		return
	}
	var catalog []struct{ Decade, N int }
	if v2Fail(c, h.raw(c, &catalog, `
		SELECT (i."ProductionYear" / 10 * 10)::int AS decade, COUNT(*)::int AS n
		FROM jf_library_items i
		WHERE i."ParentId" = ? AND i."Type" = ? AND NOT i.archived AND i."ProductionYear" > 0
		GROUP BY 1`, id, itemType)) {
		return
	}
	var views []struct{ Decade, N int }
	q, args := acts().inRange(p.Range).forUser(p.UserID).where(`(`+libraryIdExpr+`) = ?`, id).sql(`
		SELECT (i."ProductionYear" / 10 * 10)::int AS decade, ` + aggPlays + ` AS n
		FROM acts JOIN jf_library_items i ON i."Id" = acts.group_id
		WHERE i."ProductionYear" > 0 GROUP BY 1`)
	if v2Fail(c, h.raw(c, &views, q, args...)) {
		return
	}
	byDecade := map[int]*libDecadeRow{}
	get := func(d int) *libDecadeRow {
		if byDecade[d] == nil {
			byDecade[d] = &libDecadeRow{Decade: d}
		}
		return byDecade[d]
	}
	for _, r := range catalog {
		get(r.Decade).Catalog = r.N
	}
	for _, r := range views {
		get(r.Decade).Views = r.N
	}
	out := make([]libDecadeRow, 0, len(byDecade))
	for _, r := range byDecade {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Decade < out[j].Decade })
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/v2/libraries/:id/growth
// ---------------------------------------------------------------------------

type libGrowthPoint struct {
	Date       string `json:"date"`
	Added      int    `json:"added"`
	FirstViews int    `json:"firstViews"`
}

// growthBucket picks the bucket size for a range: day up to a month, week up
// to six months, month up to three years, year beyond.
func growthBucket(tr TimeRange) string {
	days := calendarDays(tr.From, tr.To)
	switch {
	case days <= 31:
		return "day"
	case days <= 186:
		return "week"
	case days <= 1100:
		return "month"
	}
	return "year"
}

// bucketStart truncates a local date to the start of its bucket (weeks start
// on Monday, like Postgres date_trunc).
func bucketStart(d time.Time, unit string) time.Time {
	switch unit {
	case "week":
		return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
	case "month":
		return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
	case "year":
		return time.Date(d.Year(), 1, 1, 0, 0, 0, 0, d.Location())
	}
	return d
}

func bucketNext(d time.Time, unit string) time.Time {
	switch unit {
	case "week":
		return d.AddDate(0, 0, 7)
	case "month":
		return d.AddDate(0, 1, 0)
	case "year":
		return d.AddDate(1, 0, 0)
	}
	return d.AddDate(0, 0, 1)
}

// libraryLeavesSQL lists (Id, DateCreated, kind = playback MediaType) of the
// movies, episodes and tracks of library ? (three placeholders, all the
// library id).
const libraryLeavesSQL = `
	SELECT i."Id", NULLIF(i."DateCreated", '')::timestamptz AS d, 'Movie' AS kind FROM jf_library_items i
	WHERE i."ParentId" = ? AND i."Type" = 'Movie' AND NOT i.archived
	UNION ALL
	SELECT e."Id", NULLIF(e."DateCreated", '')::timestamptz, 'Episode' FROM jf_library_episodes e JOIN jf_library_items s ON s."Id" = e."SeriesId"
	WHERE s."ParentId" = ? AND NOT e.archived
	UNION ALL
	SELECT t."Id", NULLIF(t."DateCreated", '')::timestamptz, 'Audio' FROM jf_music_tracks t WHERE t."LibraryId" = ? AND NOT t.archived`

// LibraryGrowth compares, per bucket of the range, the movies / episodes /
// tracks added to the library with the ones viewed for the first time (all
// time; by the userId user when given). `type` restricts both series to that
// media type. Returns {bucket, points}.
func (h *V2Handler) LibraryGrowth(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if _, ok := h.libraryOf(c, id); !ok {
		return
	}
	unit := growthBucket(p.Range) // whitelisted: safe to interpolate
	trunc := func(expr string) string {
		return `to_char(date_trunc('` + unit + `', ` + statsql.AtLocal(expr) + `)::date, 'YYYY-MM-DD')`
	}
	var added []struct {
		Date string
		N    int
	}
	if v2Fail(c, h.raw(c, &added, `
		SELECT `+trunc("l.d")+` AS date, COUNT(*)::int AS n FROM (`+libraryLeavesSQL+`) l
		WHERE l.d >= ? AND l.d < ? AND (? = '' OR l.kind = ?) GROUP BY 1`, id, id, id, p.Range.From, p.Range.To, p.MediaType, p.MediaType)) {
		return
	}
	var first []struct {
		Date string
		N    int
	}
	ff := acts().forUser(p.UserID).where(`a."IsView"`).where(`(`+libraryIdExpr+`) = ?`, id)
	if p.MediaType != "" {
		ff.where(`a."MediaType" = ?`, p.MediaType)
	}
	q, args := ff.sql(`,
		fw AS (SELECT item_id, MIN(started_at) AS first_view FROM acts GROUP BY item_id)
		SELECT `+trunc("fw.first_view")+` AS date, COUNT(*)::int AS n FROM fw
		WHERE fw.first_view >= ? AND fw.first_view < ? GROUP BY 1`, p.Range.From, p.Range.To)
	if v2Fail(c, h.raw(c, &first, q, args...)) {
		return
	}
	idx := map[string]int{}
	points := []libGrowthPoint{}
	loc := statsql.Location()
	for d := bucketStart(p.Range.From.In(loc), unit); d.Before(p.Range.To); d = bucketNext(d, unit) {
		key := d.Format(v2DateLayout)
		idx[key] = len(points)
		points = append(points, libGrowthPoint{Date: key})
	}
	for _, r := range added {
		if i, ok := idx[r.Date]; ok {
			points[i].Added = r.N
		}
	}
	for _, r := range first {
		if i, ok := idx[r.Date]; ok {
			points[i].FirstViews = r.N
		}
	}
	c.JSON(http.StatusOK, gin.H{"bucket": unit, "points": points})
}

// ---------------------------------------------------------------------------
// GET /api/v2/albums/:id/full-listens
// ---------------------------------------------------------------------------

type fullListenAlbum struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Year        *int   `json:"year"`
	Tracks      int    `json:"tracks"`
	Listens     int    `json:"listens"`
	FullListens int    `json:"fullListens"`
}

// AlbumFullListens lists the albums of the album's artist (the album itself
// when it has no artist) with their full listens in the range: (user, local
// day) pairs on which every catalog track of the album was viewed. Sorted by
// full listens, then listens, then name.
func (h *V2Handler) AlbumFullListens(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var album struct {
		ID       string
		ArtistID *string
	}
	if v2Fail(c, h.raw(c, &album, `SELECT "Id" AS id, "ArtistId" AS artist_id FROM jf_library_items WHERE "Id" = ? AND "Type" = 'MusicAlbum'`, id)) {
		return
	}
	if album.ID == "" {
		v2NotFound(c)
		return
	}
	artist := ""
	if album.ArtistID != nil {
		artist = *album.ArtistID
	}
	albums := []fullListenAlbum{}
	if v2Fail(c, h.raw(c, &albums, `
		SELECT i."Id" AS id, COALESCE(i."Name", '') AS name, i."ProductionYear" AS year,
		       (SELECT COUNT(*) FROM jf_music_tracks t WHERE t."AlbumId" = i."Id" AND NOT t.archived)::int AS tracks
		FROM jf_library_items i
		WHERE i."Type" = 'MusicAlbum' AND (i."Id" = ? OR (? <> '' AND i."ArtistId" = ? AND NOT i.archived))`, id, artist, artist)) {
		return
	}
	ids := make([]interface{}, len(albums))
	for i, a := range albums {
		ids[i] = a.ID
	}
	var stats []struct {
		GroupID     string
		Listens     int
		FullListens int
	}
	q, args := p.entityFilter().where(`a."MediaType" = 'Audio'`).sql(`,
		ud AS (
		  SELECT acts.group_id, acts.user_id, acts.local_ts::date AS day, COUNT(DISTINCT acts.item_id) AS n
		  FROM acts JOIN jf_music_tracks t ON t."Id" = acts.item_id AND t."AlbumId" = acts.group_id AND NOT t.archived
		  WHERE acts.is_view GROUP BY 1, 2, 3
		),
		tc AS (
		  SELECT t."AlbumId" AS album_id, COUNT(*) AS n FROM jf_music_tracks t WHERE NOT t.archived GROUP BY 1
		)
		SELECT g.group_id, COALESCE(l.listens, 0)::int AS listens, COUNT(ud.n) FILTER (WHERE tc.n > 0 AND ud.n >= tc.n)::int AS full_listens
		FROM (SELECT unnest(ARRAY[`+inPlaceholders(len(ids))+`]::text[]) AS group_id) g
		LEFT JOIN (SELECT group_id, `+aggPlays+` AS listens FROM acts GROUP BY 1) l ON l.group_id = g.group_id
		LEFT JOIN ud ON ud.group_id = g.group_id
		LEFT JOIN tc ON tc.album_id = g.group_id
		GROUP BY g.group_id, l.listens`, ids...)
	if v2Fail(c, h.raw(c, &stats, q, args...)) {
		return
	}
	byID := map[string]int{}
	for i, a := range albums {
		byID[a.ID] = i
	}
	for _, s := range stats {
		if i, ok := byID[s.GroupID]; ok {
			albums[i].Listens, albums[i].FullListens = s.Listens, s.FullListens
		}
	}
	sort.SliceStable(albums, func(i, j int) bool {
		a, b := albums[i], albums[j]
		if a.FullListens != b.FullListens {
			return a.FullListens > b.FullListens
		}
		if a.Listens != b.Listens {
			return a.Listens > b.Listens
		}
		return a.Name < b.Name
	})
	c.JSON(http.StatusOK, albums)
}

// ---------------------------------------------------------------------------
// GET /api/v2/series/:id/history
// ---------------------------------------------------------------------------

// SeriesHistory pages the playback rows of the series' episodes (range and
// userId apply; sort/desc like /history). Rows carry the episode name, season
// and number. Returns {rows, total, page, pageSize}.
func (h *V2Handler) SeriesHistory(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var exists int64
	if v2Fail(c, h.raw(c, &exists, `SELECT COUNT(*) FROM jf_library_items WHERE "Id" = ? AND "Type" = 'Series'`, id)) {
		return
	}
	if exists == 0 {
		v2NotFound(c)
		return
	}
	f := p.entityFilter().where(`a."MediaType" = 'Episode' AND COALESCE(ce."SeriesId", a."NowPlayingItemId") = ?`, id)
	var total int64
	q, args := f.sql(`SELECT COUNT(*) FROM acts`)
	if v2Fail(c, h.raw(c, &total, q, args...)) {
		return
	}
	rows, err := h.queryHistory(c, f, historyOrder(c, p), p.PageSize, p.Offset())
	if v2Fail(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"rows": rows, "total": total, "page": p.Page, "pageSize": p.PageSize})
}
