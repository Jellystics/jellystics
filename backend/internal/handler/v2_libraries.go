package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type libraryCard struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // movies | tvshows | music
	Items     int    `json:"items"`
	SizeBytes int64  `json:"sizeBytes"`
	Totals    Totals `json:"totals" gorm:"-"`
}

// libraryCardsSQL lists the movie / tv / music libraries with their catalog
// size: items = movies, series or albums; sizeBytes = file sizes of the
// movies, episodes or tracks (jf_item_info).
const libraryCardsSQL = `
	SELECT l."Id" AS id, COALESCE(l."Name", '') AS name, l."CollectionType" AS type,
	       CASE l."CollectionType"
	         WHEN 'movies'  THEN (SELECT COUNT(*) FROM jf_library_items i WHERE i."ParentId" = l."Id" AND i."Type" = 'Movie' AND NOT i.archived)
	         WHEN 'tvshows' THEN (SELECT COUNT(*) FROM jf_library_items i WHERE i."ParentId" = l."Id" AND i."Type" = 'Series' AND NOT i.archived)
	         ELSE (SELECT COUNT(*) FROM jf_library_items i WHERE i."ParentId" = l."Id" AND i."Type" = 'MusicAlbum' AND NOT i.archived)
	       END::int AS items,
	       COALESCE((SELECT SUM(ii."Size") FROM jf_item_info ii WHERE ii."Id" IN (
	         SELECT i."Id" FROM jf_library_items i WHERE i."ParentId" = l."Id" AND i."Type" = 'Movie' AND NOT i.archived
	         UNION ALL
	         SELECT e."Id" FROM jf_library_episodes e JOIN jf_library_items s ON s."Id" = e."SeriesId"
	         WHERE s."ParentId" = l."Id" AND NOT e.archived
	         UNION ALL
	         SELECT t."Id" FROM jf_music_tracks t WHERE t."LibraryId" = l."Id" AND NOT t.archived
	       )), 0)::bigint AS size_bytes
	FROM jf_libraries l
	WHERE NOT l.archived AND l."CollectionType" IN ('movies', 'tvshows', 'music')`

// libraryTotals returns Totals per library id over f.
func (h *V2Handler) libraryTotals(c *gin.Context, f *actFilter) (map[string]Totals, error) {
	var rows []struct {
		LibraryID string
		Totals
	}
	q, args := f.sql(`,
		lu AS (
		  SELECT library_id, ` + statsqlCompletion + ` AS r
		  FROM acts WHERE is_view AND runtime_ticks > 0 GROUP BY library_id, user_id, item_id, local_ts::date
		),
		lc AS (SELECT library_id, AVG(r)::float8 AS completion FROM lu GROUP BY library_id)
		SELECT acts.library_id, ` + aggSeconds + ` AS watch_seconds, ` + aggPlays + ` AS views, ` + aggClicks + ` AS clicks,
		       COUNT(DISTINCT user_id)::int AS active_users, MAX(lc.completion) AS completion
		FROM acts LEFT JOIN lc ON lc.library_id = acts.library_id
		WHERE acts.library_id IS NOT NULL GROUP BY acts.library_id`)
	if err := h.raw(c, &rows, q, args...); err != nil {
		return nil, err
	}
	m := make(map[string]Totals, len(rows))
	for _, r := range rows {
		m[r.LibraryID] = r.Totals
	}
	return m, nil
}

// GET /api/v2/libraries
func (h *V2Handler) Libraries(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	cards := []libraryCard{}
	if v2Fail(c, h.raw(c, &cards, libraryCardsSQL+` ORDER BY l."Name"`)) {
		return
	}
	tot, err := h.libraryTotals(c, acts().inRange(p.Range).forUser(p.UserID))
	if v2Fail(c, err) {
		return
	}
	for i := range cards {
		cards[i].Totals = tot[cards[i].ID]
	}
	c.JSON(http.StatusOK, cards)
}

type libraryItem struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Views        int      `json:"views"`
	Viewers      int      `json:"viewers"`
	WatchSeconds int64    `json:"watchSeconds"`
	Retention    *float64 `json:"retention,omitempty"`
	Trend        []int64  `json:"trend"`
}

var librarySorts = map[string]string{
	"name":         "name",
	"views":        "views",
	"viewers":      "viewers",
	"watchSeconds": "watch_seconds",
}

// GET /api/v2/libraries/:id — card + paged items (movies / series / albums,
// q filters by name, sort=name|views|viewers|watchSeconds, default views
// desc), genres, methods, unwatched titles and time to first watch.
func (h *V2Handler) Library(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var card libraryCard
	if v2Fail(c, h.raw(c, &card, libraryCardsSQL+` AND l."Id" = ?`, id)) {
		return
	}
	if card.ID == "" {
		v2NotFound(c)
		return
	}
	f := acts().inRange(p.Range).forUser(p.UserID).where(`(`+libraryIdExpr+`) = ?`, id)
	tot, err := h.libraryTotals(c, f)
	if v2Fail(c, err) {
		return
	}
	card.Totals = tot[id]

	itemType := map[string]string{"movies": "Movie", "tvshows": "Series", "music": "MusicAlbum"}[card.Type]
	kind := map[string]string{"movies": "movie", "tvshows": "series", "music": "album"}[card.Type]
	nameCond, nameArgs := "", []interface{}{}
	if p.Q != "" {
		nameCond = ` AND i."Name" ILIKE ?`
		nameArgs = append(nameArgs, likePattern(p.Q))
	}
	var total int64
	cntArgs := append([]interface{}{id, itemType}, nameArgs...)
	if v2Fail(c, h.raw(c, &total, `SELECT COUNT(*) FROM jf_library_items i WHERE i."ParentId" = ? AND i."Type" = ? AND NOT i.archived`+nameCond, cntArgs...)) {
		return
	}
	sortExpr, ok := librarySorts[p.Sort]
	if !ok {
		sortExpr = "views"
	}
	dir := " DESC"
	if p.Sort == "name" {
		dir = " ASC"
	}
	if c.Query("desc") != "" {
		dir = " ASC"
		if p.Desc {
			dir = " DESC"
		}
	}
	var items []struct {
		ID, Name     string
		Views        int
		Viewers      int
		WatchSeconds int64
		First, Last  *int
	}
	outerArgs := append([]interface{}{id, itemType}, nameArgs...)
	outerArgs = append(outerArgs, p.PageSize, p.Offset())
	q, args := f.sql(`,
		st AS (
		  SELECT group_id, `+aggPlays+` AS views, COUNT(DISTINCT user_id) FILTER (WHERE is_view)::int AS viewers, `+aggSeconds+` AS watch_seconds
		  FROM acts GROUP BY group_id
		),
		epv AS (
		  SELECT item_id, COUNT(DISTINCT user_id) FILTER (WHERE is_view)::int AS v FROM acts WHERE media_type = 'Episode' GROUP BY item_id
		),
		eps AS (
		  SELECT e."SeriesId" AS series_id, e."Id",
		         ROW_NUMBER() OVER (PARTITION BY e."SeriesId" ORDER BY `+episodeOrder+`) AS rn,
		         COUNT(*) OVER (PARTITION BY e."SeriesId") AS n
		  FROM jf_library_episodes e WHERE NOT e.archived AND `+notSpecial+`
		),
		ret AS (
		  SELECT series_id, MAX(COALESCE(epv.v, 0)) FILTER (WHERE rn = 1) AS first, MAX(COALESCE(epv.v, 0)) FILTER (WHERE rn = n) AS last
		  FROM eps LEFT JOIN epv ON epv.item_id = eps."Id" GROUP BY series_id
		)
		SELECT i."Id" AS id, COALESCE(i."Name", '') AS name, COALESCE(st.views, 0) AS views, COALESCE(st.viewers, 0) AS viewers,
		       COALESCE(st.watch_seconds, 0) AS watch_seconds, ret.first, ret.last
		FROM jf_library_items i
		LEFT JOIN st ON st.group_id = i."Id"
		LEFT JOIN ret ON ret.series_id = i."Id"
		WHERE i."ParentId" = ? AND i."Type" = ? AND NOT i.archived`+nameCond+`
		ORDER BY `+sortExpr+dir+`, i."Id"
		LIMIT ? OFFSET ?`, outerArgs...)
	if v2Fail(c, h.raw(c, &items, q, args...)) {
		return
	}
	tr := trendRange(p.Range)
	ids := make([]interface{}, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	trends := map[string]map[string]int64{}
	if len(ids) > 0 {
		var rows []struct {
			GroupID, Date string
			Views         int64
		}
		q, args = acts().inRange(tr).forUser(p.UserID).where(`(`+libraryIdExpr+`) = ?`, id).
			sql(`SELECT group_id, `+localDay+` AS date, `+aggPlays+`::bigint AS views FROM acts WHERE group_id IN (`+inPlaceholders(len(ids))+`) GROUP BY 1, 2`, ids...)
		if v2Fail(c, h.raw(c, &rows, q, args...)) {
			return
		}
		for _, r := range rows {
			if trends[r.GroupID] == nil {
				trends[r.GroupID] = map[string]int64{}
			}
			trends[r.GroupID][r.Date] = r.Views
		}
	}
	list := make([]libraryItem, len(items))
	for i, it := range items {
		li := libraryItem{ID: it.ID, Kind: kind, Name: it.Name, Views: it.Views, Viewers: it.Viewers, WatchSeconds: it.WatchSeconds, Trend: trendSeries(p.Range, trends[it.ID])}
		if kind == "series" && it.First != nil && *it.First > 0 && it.Last != nil {
			r := float64(*it.Last) / float64(*it.First)
			li.Retention = &r
		}
		list[i] = li
	}

	type genreRow struct {
		Name         string `json:"name"`
		WatchSeconds int64  `json:"watchSeconds"`
		Views        int    `json:"views"`
	}
	genres := []genreRow{}
	q, args = f.sql(`SELECT genre AS name, ` + aggSeconds + ` AS watch_seconds, ` + aggPlays + ` AS views
		FROM acts ` + genreLateral + ` WHERE acts.genres IS NOT NULL
		GROUP BY genre ORDER BY watch_seconds DESC, genre LIMIT 15`)
	if v2Fail(c, h.raw(c, &genres, q, args...)) {
		return
	}
	var methods methodSplit
	q, args = f.sql(`SELECT COUNT(*) FILTER (WHERE play_method = 'DirectPlay')::int AS direct,
		COUNT(*) FILTER (WHERE play_method = 'DirectStream')::int AS stream,
		COUNT(*) FILTER (WHERE play_method = 'Transcode')::int AS transcode FROM acts WHERE is_view`)
	if v2Fail(c, h.raw(c, &methods, q, args...)) {
		return
	}

	// Unwatched: titles of the library never viewed (all time, all users).
	type unwatchedItem struct {
		ID        string  `json:"id"`
		Kind      string  `json:"kind"`
		Name      string  `json:"name"`
		DateAdded *string `json:"dateAdded"`
		Total     int64   `json:"-"`
	}
	// One pass over the watched set: the window count gives the total.
	var unCount int64
	unList := []unwatchedItem{}
	watched := acts().where(`a."IsView"`).where(`(`+libraryIdExpr+`) = ?`, id)
	q, args = watched.sql(`SELECT i."Id" AS id, ? AS kind, COALESCE(i."Name", '') AS name,
		       to_char(NULLIF(i."DateCreated", '')::timestamptz, 'YYYY-MM-DD') AS date_added,
		       COUNT(*) OVER () AS total
		FROM jf_library_items i
		WHERE i."ParentId" = ? AND i."Type" = ? AND NOT i.archived AND i."Id" NOT IN (SELECT group_id FROM acts WHERE group_id IS NOT NULL)
		ORDER BY NULLIF(i."DateCreated", '')::timestamptz DESC NULLS LAST, i."Name" LIMIT 20`, kind, id, itemType)
	if v2Fail(c, h.raw(c, &unList, q, args...)) {
		return
	}
	if len(unList) > 0 {
		unCount = unList[0].Total
	}

	// Time to watch: days between catalog add and the first view (all users)
	// of movies / episodes / tracks first viewed within the range.
	var ttw struct {
		Count      int      `json:"count"`
		AvgDays    *float64 `json:"avgDays"`
		MedianDays *float64 `json:"medianDays"`
	}
	q, args = watched.sql(`,
		fw AS (SELECT item_id, MIN(started_at) AS first_view FROM acts GROUP BY item_id),
		added AS (
		  SELECT "Id", NULLIF("DateCreated", '')::timestamptz AS d FROM jf_library_items WHERE "Type" = 'Movie'
		  UNION ALL SELECT "Id", NULLIF("DateCreated", '')::timestamptz FROM jf_library_episodes
		  UNION ALL SELECT "Id", NULLIF("DateCreated", '')::timestamptz FROM jf_music_tracks
		),
		lag AS (
		  SELECT EXTRACT(EPOCH FROM (fw.first_view - added.d)) / 86400.0 AS days
		  FROM fw JOIN added ON added."Id" = fw.item_id
		  WHERE added.d IS NOT NULL AND fw.first_view >= added.d AND fw.first_view >= ? AND fw.first_view < ?
		)
		SELECT COUNT(*)::int AS count, AVG(days)::float8 AS avg_days,
		       (percentile_cont(0.5) WITHIN GROUP (ORDER BY days))::float8 AS median_days
		FROM lag`, p.Range.From, p.Range.To)
	if v2Fail(c, h.raw(c, &ttw, q, args...)) {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"library":     card,
		"items":       gin.H{"rows": list, "total": total, "page": p.Page, "pageSize": p.PageSize},
		"genres":      genres,
		"methods":     methods,
		"unwatched":   gin.H{"count": unCount, "list": unList},
		"timeToWatch": ttw,
	})
}
