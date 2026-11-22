package handler

import (
	"net/http"
	"time"

	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
)

// tzName is the validated display time zone name (safe to embed in SQL).
func tzName() string {
	n, _ := statsql.Timezone()
	return n
}

// The discover endpoints work on the top-level catalog: movies and series
// (albums instead when type=music), not archived, optionally limited to one
// library (libraryId = jf_library_items.ParentId).

// catalogTypes returns the jf_library_items types covered by the request.
func catalogTypes(p V2Params) []interface{} {
	switch p.Type {
	case "movie":
		return []interface{}{"Movie"}
	case "series":
		return []interface{}{"Series"}
	case "music":
		return []interface{}{"MusicAlbum"}
	}
	return []interface{}{"Movie", "Series"}
}

// catalogCTE (append after the acts CTE) defines `cat`: one row per title
// with kind, name, add date, production year and size on disk (movie file,
// or the sum of the episode / track files, pre-aggregated once per series /
// album). Args: catalogArgs(p).
func catalogCTE(p V2Params) string {
	lib := ""
	if p.LibraryID != "" {
		lib = ` AND i."ParentId" = ?`
	}
	var series, albums bool
	for _, t := range catalogTypes(p) {
		series = series || t == "Series"
		albums = albums || t == "MusicAlbum"
	}
	pre, joins, seriesSize, albumSize := "", "", "0", "0"
	if series {
		pre += `,
	ep_size AS (
	  SELECT e."SeriesId" AS id, SUM(ei."Size") AS size
	  FROM jf_library_episodes e JOIN jf_item_info ei ON ei."Id" = e."Id"
	  WHERE NOT e.archived GROUP BY e."SeriesId"
	)`
		joins += `
	  LEFT JOIN ep_size es ON es.id = i."Id"`
		seriesSize = `COALESCE(es.size, 0)`
	}
	if albums {
		pre += `,
	trk_size AS (
	  SELECT t."AlbumId" AS id, SUM(ti."Size") AS size
	  FROM jf_music_tracks t JOIN jf_item_info ti ON ti."Id" = t."Id"
	  GROUP BY t."AlbumId"
	)`
		joins += `
	  LEFT JOIN trk_size ts ON ts.id = i."Id"`
		albumSize = `COALESCE(ts.size, 0)`
	}
	return pre + `,
	cat AS (
	  SELECT i."Id" AS id,
	         CASE i."Type" WHEN 'Movie' THEN 'movie' WHEN 'Series' THEN 'series' ELSE 'album' END AS kind,
	         COALESCE(i."Name", '') AS name,
	         NULLIF(i."DateCreated", '')::timestamptz AS added,
	         i."ProductionYear" AS year,
	         (CASE i."Type"
	            WHEN 'Movie' THEN COALESCE(ii."Size", 0)
	            WHEN 'Series' THEN ` + seriesSize + `
	            ELSE ` + albumSize + `
	          END)::bigint AS size_bytes
	  FROM jf_library_items i
	  LEFT JOIN jf_item_info ii ON ii."Id" = i."Id"` + joins + `
	  WHERE i."Type" IN (` + inPlaceholders(len(catalogTypes(p))) + `) AND NOT i.archived` + lib + `
	)`
}

func catalogArgs(p V2Params) []interface{} {
	args := catalogTypes(p)
	if p.LibraryID != "" {
		args = append(args, p.LibraryID)
	}
	return args
}

// withArgs concatenates argument lists.
func withArgs(lists ...[]interface{}) []interface{} {
	out := []interface{}{}
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

type catalogItem struct {
	ID        string  `json:"id"`
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	DateAdded *string `json:"dateAdded"`
	SizeBytes int64   `json:"sizeBytes"`
}

// GET /api/v2/discover/unwatched → titles never viewed (all time; by the
// user when userId is given). Oldest additions first.
func (h *V2Handler) DiscoverUnwatched(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	f := acts().forUser(p.UserID)
	cte := catalogCTE(p) + `,
	seen AS (SELECT DISTINCT group_id FROM acts WHERE is_view AND group_id IS NOT NULL),
	un AS (SELECT * FROM cat WHERE id NOT IN (SELECT group_id FROM seen))`
	// One query (the all-time acts scan behind `seen` runs once): the summary
	// row, joined to the 20 oldest additions (no item row when none).
	var rows []struct {
		Count, Catalog int
		TotalSize      int64
		ID, Kind, Name *string
		DateAdded      *string
		SizeBytes      *int64
	}
	q, args := f.sql(cte+`,
	sm AS (SELECT (SELECT COUNT(*) FROM un)::int AS count, (SELECT COUNT(*) FROM cat)::int AS catalog,
	              (SELECT COALESCE(SUM(size_bytes), 0) FROM un)::bigint AS total_size)
	SELECT sm.count, sm.catalog, sm.total_size, top.id, top.kind, top.name, top.date_added, top.size_bytes
	FROM sm LEFT JOIN LATERAL (
	  SELECT id, kind, name, added, to_char(added AT TIME ZONE '`+tzName()+`', 'YYYY-MM-DD') AS date_added, size_bytes
	  FROM un ORDER BY added ASC NULLS LAST, name, id LIMIT 20
	) top ON true
	ORDER BY top.added ASC NULLS LAST, top.name, top.id`, catalogArgs(p)...)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	items := []catalogItem{}
	var count, catalog int
	var total int64
	for _, r := range rows {
		count, catalog, total = r.Count, r.Catalog, r.TotalSize
		if r.ID != nil {
			items = append(items, catalogItem{ID: *r.ID, Kind: deref(r.Kind), Name: deref(r.Name), DateAdded: r.DateAdded, SizeBytes: derefInt64(r.SizeBytes)})
		}
	}
	c.JSON(http.StatusOK, gin.H{"count": count, "catalog": catalog, "sizeBytes": total, "items": items})
}

func derefInt64(n *int64) int64 {
	if n == nil {
		return 0
	}
	return *n
}

// abandonDays: a title is abandoned when its last play is older than this.
const abandonDays = 60

type abandonedRow struct {
	ID              string      `json:"id"`
	Kind            string      `json:"kind"` // movie | series
	Name            string      `json:"name"`
	User            RefUserJSON `json:"user"`
	Progress        float64     `json:"progress"`
	LastPlayed      time.Time   `json:"lastPlayed"`
	PositionSeconds *int64      `json:"positionSeconds,omitempty"`
	EpisodesViewed  *int        `json:"episodesViewed,omitempty"`
	EpisodesTotal   *int        `json:"episodesTotal,omitempty"`
}

// GET /api/v2/discover/abandoned → (user, title) pairs left unfinished whose
// last play is older than 60 days (all time; userId / libraryId apply):
// movies whose best completion is in [10 %, 80 %), series with 10–80 % of the
// catalog episodes viewed. Most recently played first, 20 max.
func (h *V2Handler) DiscoverAbandoned(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -abandonDays)
	pp := p
	pp.Type, pp.MediaType = "", ""
	f := pp.filter(TimeRange{AllTime: true}).where(`a."IsView"`)
	switch p.Type {
	case "movie":
		f.where(`a."MediaType" = 'Movie'`)
	case "series":
		f.where(`a."MediaType" = 'Episode'`)
	case "music":
		c.JSON(http.StatusOK, gin.H{"count": 0, "items": []abandonedRow{}})
		return
	default:
		f.where(`a."MediaType" IN ('Movie', 'Episode')`)
	}
	cte := `,
	mv AS (
	  SELECT 'movie' AS kind, group_id AS id, MAX(group_name) AS name, user_id, MAX(user_name) AS user_name,
	         MAX(` + ratioExpr + `) AS progress, MAX(started_at) AS last_played,
	         (MAX(pos_end) / 10000000)::bigint AS position_seconds, NULL::int AS episodes_viewed, NULL::int AS episodes_total
	  FROM acts WHERE media_type = 'Movie' AND group_id IS NOT NULL AND user_id IS NOT NULL
	  GROUP BY group_id, user_id
	  HAVING MAX(` + ratioExpr + `) >= 0.1 AND MAX(` + ratioExpr + `) < 0.8 AND MAX(started_at) < ?
	),
	eps AS (SELECT "SeriesId" AS series_id, COUNT(*)::int AS total FROM jf_library_episodes WHERE NOT archived GROUP BY 1),
	sr AS (
	  SELECT 'series' AS kind, group_id AS id, MAX(group_name) AS name, user_id, MAX(user_name) AS user_name,
	         COUNT(DISTINCT item_id)::float8 / MAX(eps.total) AS progress, MAX(started_at) AS last_played,
	         NULL::bigint AS position_seconds, COUNT(DISTINCT item_id)::int AS episodes_viewed, MAX(eps.total) AS episodes_total
	  FROM acts JOIN eps ON eps.series_id = acts.group_id
	  WHERE media_type = 'Episode' AND user_id IS NOT NULL AND eps.total > 0
	  GROUP BY group_id, user_id
	  HAVING COUNT(DISTINCT item_id)::float8 / MAX(eps.total) >= 0.1 AND COUNT(DISTINCT item_id)::float8 / MAX(eps.total) < 0.8
	     AND MAX(started_at) < ?
	),
	ab AS (SELECT * FROM mv UNION ALL SELECT * FROM sr)`
	var rows []struct {
		Count                            int
		Kind, ID, Name, UserID, UserName string
		Progress                         float64
		LastPlayed                       time.Time
		PositionSeconds                  *int64
		EpisodesViewed, EpisodesTotal    *int
	}
	q, args := f.sql(cte+`
		SELECT COUNT(*) OVER ()::int AS count, kind, id, COALESCE(name, '') AS name, user_id, COALESCE(user_name, user_id) AS user_name, progress,
		       last_played, position_seconds, episodes_viewed, episodes_total
		FROM ab ORDER BY last_played DESC, id, user_id LIMIT 20`, cutoff, cutoff)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	items := make([]abandonedRow, len(rows))
	for i, r := range rows {
		items[i] = abandonedRow{ID: r.ID, Kind: r.Kind, Name: r.Name, User: RefUserJSON{ID: r.UserID, Name: r.UserName},
			Progress: r.Progress, LastPlayed: localTime(r.LastPlayed), PositionSeconds: r.PositionSeconds,
			EpisodesViewed: r.EpisodesViewed, EpisodesTotal: r.EpisodesTotal}
	}
	count := 0
	if len(rows) > 0 {
		count = rows[0].Count
	}
	c.JSON(http.StatusOK, gin.H{"count": count, "items": items})
}

// GET /api/v2/discover/added → titles added to the catalog within the range,
// and how many of them have been viewed since (all users / userId).
func (h *V2Handler) DiscoverAdded(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	f := acts().forUser(p.UserID)
	cte := catalogCTE(p) + `,
	seen AS (SELECT DISTINCT group_id FROM acts WHERE is_view AND group_id IS NOT NULL),
	ad AS (SELECT cat.*, (cat.id IN (SELECT group_id FROM seen)) AS seen FROM cat WHERE added >= ? AND added < ?)`
	cargs := withArgs(catalogArgs(p), []interface{}{p.Range.From, p.Range.To})
	type addedItem struct {
		catalogItem
		Seen bool `json:"seen"`
	}
	// One query: the window aggregates carry the totals on every listed row.
	var rows []struct {
		Count, SeenCount int
		ID, Kind, Name   string
		DateAdded        *string
		SizeBytes        int64
		Seen             bool
	}
	q, args := f.sql(cte+`
		SELECT count, seen_count, id, kind, name, date_added, size_bytes, seen FROM (
		  SELECT COUNT(*) OVER ()::int AS count, (COUNT(*) FILTER (WHERE seen) OVER ())::int AS seen_count,
		         id, kind, name, added, to_char(added AT TIME ZONE '`+tzName()+`', 'YYYY-MM-DD') AS date_added, size_bytes, seen
		  FROM ad
		) x ORDER BY added DESC, name, id LIMIT 20`, cargs...)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	items := make([]addedItem, len(rows))
	count, seen := 0, 0
	for i, r := range rows {
		count, seen = r.Count, r.SeenCount
		items[i] = addedItem{catalogItem{r.ID, r.Kind, r.Name, r.DateAdded, r.SizeBytes}, r.Seen}
	}
	c.JSON(http.StatusOK, gin.H{"count": count, "seen": seen, "items": items})
}

type storagePoint struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	SizeBytes int64  `json:"sizeBytes"`
	Views     int    `json:"views"`
}

// GET /api/v2/discover/storage → size on disk vs views in the range of the
// 300 largest titles with a known size.
func (h *V2Handler) DiscoverStorage(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	out := []storagePoint{}
	q, args := p.filter(p.Range).sql(catalogCTE(p)+`,
		v AS (SELECT group_id, `+aggPlays+` AS views FROM acts GROUP BY group_id)
		SELECT cat.id, cat.kind, cat.name, cat.size_bytes, COALESCE(v.views, 0) AS views
		FROM cat LEFT JOIN v ON v.group_id = cat.id
		WHERE cat.size_bytes > 0
		ORDER BY cat.size_bytes DESC, cat.id LIMIT 300`, catalogArgs(p)...)
	if v2Fail(c, h.raw(c, &out, q, args...)) {
		return
	}
	c.JSON(http.StatusOK, out)
}
