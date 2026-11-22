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
// or the sum of the episode / track files). Args: catalogArgs(p).
func catalogCTE(p V2Params) string {
	lib := ""
	if p.LibraryID != "" {
		lib = ` AND i."ParentId" = ?`
	}
	return `,
	cat AS (
	  SELECT i."Id" AS id,
	         CASE i."Type" WHEN 'Movie' THEN 'movie' WHEN 'Series' THEN 'series' ELSE 'album' END AS kind,
	         COALESCE(i."Name", '') AS name,
	         NULLIF(i."DateCreated", '')::timestamptz AS added,
	         i."ProductionYear" AS year,
	         (CASE i."Type"
	            WHEN 'Movie' THEN COALESCE(ii."Size", 0)
	            WHEN 'Series' THEN COALESCE((SELECT SUM(ei."Size") FROM jf_library_episodes e JOIN jf_item_info ei ON ei."Id" = e."Id"
	                                         WHERE e."SeriesId" = i."Id" AND NOT e.archived), 0)
	            ELSE COALESCE((SELECT SUM(ti."Size") FROM jf_music_tracks t JOIN jf_item_info ti ON ti."Id" = t."Id"
	                           WHERE t."AlbumId" = i."Id"), 0)
	          END)::bigint AS size_bytes
	  FROM jf_library_items i
	  LEFT JOIN jf_item_info ii ON ii."Id" = i."Id"
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
	p, ok := v2Params(c)
	if !ok {
		return
	}
	f := acts().forUser(p.UserID)
	cte := catalogCTE(p) + `,
	seen AS (SELECT DISTINCT group_id FROM acts WHERE is_view AND group_id IS NOT NULL),
	un AS (SELECT * FROM cat WHERE id NOT IN (SELECT group_id FROM seen))`
	var sum struct {
		Count, Catalog int
		SizeBytes      int64
	}
	q, args := f.sql(cte+`
		SELECT (SELECT COUNT(*) FROM un)::int AS count, (SELECT COUNT(*) FROM cat)::int AS catalog,
		       (SELECT COALESCE(SUM(size_bytes), 0) FROM un)::bigint AS size_bytes`, catalogArgs(p)...)
	if v2Fail(c, h.raw(c, &sum, q, args...)) {
		return
	}
	items := []catalogItem{}
	q, args = f.sql(cte+`
		SELECT id, kind, name, to_char(added AT TIME ZONE '`+tzName()+`', 'YYYY-MM-DD') AS date_added, size_bytes
		FROM un ORDER BY added ASC NULLS LAST, name, id LIMIT 20`, catalogArgs(p)...)
	if v2Fail(c, h.raw(c, &items, q, args...)) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": sum.Count, "catalog": sum.Catalog, "sizeBytes": sum.SizeBytes, "items": items})
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
// movies whose best completion is in [10 %, 80 %), series with < 80 % of the
// catalog episodes viewed. Most recently played first, 20 max.
func (h *V2Handler) DiscoverAbandoned(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -abandonDays)
	pp := p
	pp.Type, pp.MediaType = "", ""
	f := pp.analyticsFilter(TimeRange{AllTime: true}).where(`a."IsView"`)
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
	  HAVING COUNT(DISTINCT item_id)::float8 / MAX(eps.total) < 0.8 AND MAX(started_at) < ?
	),
	ab AS (SELECT * FROM mv UNION ALL SELECT * FROM sr)`
	var count int
	q, args := f.sql(cte+` SELECT COUNT(*)::int FROM ab`, cutoff, cutoff)
	if v2Fail(c, h.raw(c, &count, q, args...)) {
		return
	}
	var rows []struct {
		Kind, ID, Name, UserID, UserName string
		Progress                         float64
		LastPlayed                       time.Time
		PositionSeconds                  *int64
		EpisodesViewed, EpisodesTotal    *int
	}
	q, args = f.sql(cte+`
		SELECT kind, id, COALESCE(name, '') AS name, user_id, COALESCE(user_name, user_id) AS user_name, progress,
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
	c.JSON(http.StatusOK, gin.H{"count": count, "items": items})
}

// GET /api/v2/discover/added → titles added to the catalog within the range,
// and how many of them have been viewed since (all users / userId).
func (h *V2Handler) DiscoverAdded(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	f := acts().forUser(p.UserID)
	cte := catalogCTE(p) + `,
	seen AS (SELECT DISTINCT group_id FROM acts WHERE is_view AND group_id IS NOT NULL),
	ad AS (SELECT cat.*, (cat.id IN (SELECT group_id FROM seen)) AS seen FROM cat WHERE added >= ? AND added < ?)`
	cargs := withArgs(catalogArgs(p), []interface{}{p.Range.From, p.Range.To})
	var sum struct{ Count, Seen int }
	q, args := f.sql(cte+` SELECT COUNT(*)::int AS count, COUNT(*) FILTER (WHERE seen)::int AS seen FROM ad`, cargs...)
	if v2Fail(c, h.raw(c, &sum, q, args...)) {
		return
	}
	type addedItem struct {
		catalogItem
		Seen bool `json:"seen"`
	}
	var rows []struct {
		ID, Kind, Name string
		DateAdded      *string
		SizeBytes      int64
		Seen           bool
	}
	q, args = f.sql(cte+`
		SELECT id, kind, name, to_char(added AT TIME ZONE '`+tzName()+`', 'YYYY-MM-DD') AS date_added, size_bytes, seen
		FROM ad ORDER BY added DESC, name, id LIMIT 20`, cargs...)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	items := make([]addedItem, len(rows))
	for i, r := range rows {
		items[i] = addedItem{catalogItem{r.ID, r.Kind, r.Name, r.DateAdded, r.SizeBytes}, r.Seen}
	}
	c.JSON(http.StatusOK, gin.H{"count": sum.Count, "seen": sum.Seen, "items": items})
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
	p, ok := v2Params(c)
	if !ok {
		return
	}
	out := []storagePoint{}
	q, args := p.analyticsFilter(p.Range).sql(catalogCTE(p)+`,
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

type decadeRow struct {
	Decade int `json:"decade"`
	Items  int `json:"items"`
	Views  int `json:"views"`
}

// GET /api/v2/discover/decades → catalog titles and views in the range per
// production decade (titles without a year are skipped).
func (h *V2Handler) DiscoverDecades(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	out := []decadeRow{}
	q, args := p.analyticsFilter(p.Range).sql(catalogCTE(p)+`,
		v AS (SELECT group_id, `+aggPlays+` AS views FROM acts GROUP BY group_id)
		SELECT (cat.year / 10 * 10)::int AS decade, COUNT(*)::int AS items, COALESCE(SUM(v.views), 0)::int AS views
		FROM cat LEFT JOIN v ON v.group_id = cat.id
		WHERE cat.year IS NOT NULL AND cat.year > 0
		GROUP BY 1 ORDER BY 1`, catalogArgs(p)...)
	if v2Fail(c, h.raw(c, &out, q, args...)) {
		return
	}
	c.JSON(http.StatusOK, out)
}

type growthPoint struct {
	Month      string `json:"month"`
	Added      int    `json:"added"`
	FirstViews int    `json:"firstViews"`
}

// GET /api/v2/discover/growth → per local month of the range: titles added
// to the catalog, and titles viewed for the first time ever (by anyone, or by
// userId).
func (h *V2Handler) DiscoverGrowth(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	var rows []struct {
		Month             string
		Added, FirstViews int
	}
	f := acts().forUser(p.UserID).where(`a."IsView"`)
	q, args := f.sql(catalogCTE(p)+`,
		fv AS (
		  SELECT group_id, MIN(started_at) AS first FROM acts
		  WHERE group_id IN (SELECT id FROM cat) GROUP BY group_id
		),
		ev AS (
		  SELECT to_char(added AT TIME ZONE '`+tzName()+`', 'YYYY-MM') AS month, 1 AS a, 0 AS v FROM cat WHERE added >= ? AND added < ?
		  UNION ALL
		  SELECT to_char(first AT TIME ZONE '`+tzName()+`', 'YYYY-MM'), 0, 1 FROM fv WHERE first >= ? AND first < ?
		)
		SELECT month, SUM(a)::int AS added, SUM(v)::int AS first_views FROM ev GROUP BY month`,
		withArgs(catalogArgs(p), []interface{}{p.Range.From, p.Range.To, p.Range.From, p.Range.To})...)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	by := map[string]growthPoint{}
	for _, r := range rows {
		by[r.Month] = growthPoint{Month: r.Month, Added: r.Added, FirstViews: r.FirstViews}
	}
	out := []growthPoint{}
	for _, m := range monthsOf(p.Range) {
		g := by[m]
		g.Month = m
		out = append(out, g)
	}
	c.JSON(http.StatusOK, out)
}
