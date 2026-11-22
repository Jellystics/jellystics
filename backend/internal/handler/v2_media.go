package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// entityFilter: range + user only (type / library filters are meaningless on
// an entity page).
func (p V2Params) entityFilter() *actFilter {
	return acts().inRange(p.Range).forUser(p.UserID)
}

// libraryRef is the library an entity belongs to (kicker link / breadcrumb).
type libraryRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// libraryRefOf resolves a library id to {id, name}; nil when empty or unknown.
func (h *V2Handler) libraryRefOf(c *gin.Context, id *string) (*libraryRef, error) {
	if id == nil || *id == "" {
		return nil, nil
	}
	var r libraryRef
	if err := h.raw(c, &r, `SELECT "Id" AS id, COALESCE("Name", '') AS name FROM jf_libraries WHERE "Id" = ?`, *id); err != nil {
		return nil, err
	}
	if r.ID == "" {
		return nil, nil
	}
	return &r, nil
}

type refUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type funnelStats struct {
	Clicks    int `json:"clicks"`
	Views     int `json:"views"`
	Half      int `json:"half"`
	Completed int `json:"completed"`
}

// funnel: clicks/views are rows; half/completed count the (user, item, day)
// view units reaching 50 % (or finished, so the funnel stays monotone) /
// finished (completionCTE).
func (h *V2Handler) funnel(c *gin.Context, f *actFilter) (funnelStats, error) {
	var out funnelStats
	q, args := f.sql(completionCTE + `
		SELECT ` + aggClicks + ` AS clicks, ` + aggPlays + ` AS views,
		       (SELECT COUNT(*) FROM completion WHERE completion_ratio >= 0.5 OR finished)::int AS half,
		       (SELECT COUNT(*) FROM completion WHERE ` + completedCond + `)::int AS completed
		FROM acts`)
	err := h.raw(c, &out, q, args...)
	return out, err
}

// stopPositions histograms, over `bins` equal slices of the runtime, the
// furthest stop position of each (user, item, local day) view unit (same unit
// as the funnel). The stop position is PositionTicksEnd, or the start
// position plus the watched duration when the end position is unknown.
func (h *V2Handler) stopPositions(c *gin.Context, f *actFilter, bins int) ([]int, error) {
	var rows []struct{ Bin, N int }
	q, args := f.sql(fmt.Sprintf(`,
		units AS (
		  SELECT LEAST(GREATEST(MAX(COALESCE(pos_end, COALESCE(pos_start, 0) + duration * 10000000))::float8
		                        / NULLIF(MAX(runtime_ticks), 0)::float8, 0), 1) AS r
		  FROM acts WHERE is_view AND runtime_ticks > 0 AND NOT plugin_open
		  GROUP BY user_id, item_id, local_ts::date
		)
		SELECT LEAST(FLOOR(r * %d), %d)::int AS bin, COUNT(*)::int AS n
		FROM units WHERE r IS NOT NULL GROUP BY 1`, bins, bins-1))
	if err := h.raw(c, &rows, q, args...); err != nil {
		return nil, err
	}
	out := make([]int, bins)
	for _, r := range rows {
		if r.Bin >= 0 && r.Bin < bins {
			out[r.Bin] = r.N
		}
	}
	return out, nil
}

func ticksToSeconds(t *int64) int64 {
	if t == nil {
		return 0
	}
	return *t / 10_000_000
}

func strs(j jsonOf[[]string]) []string {
	if j.V == nil {
		return []string{}
	}
	return j.V
}

// resolutionLabel maps the first video stream width to a label.
func resolutionLabel(width *int) *string {
	if width == nil || *width <= 0 {
		return nil
	}
	var s string
	switch w := *width; {
	case w >= 3800:
		s = "4K"
	case w >= 1900:
		s = "1080p"
	case w >= 1200:
		s = "720p"
	default:
		s = "SD"
	}
	return &s
}

// ---------------------------------------------------------------------------
// GET /api/v2/movies/:id
// ---------------------------------------------------------------------------

type movieViewer struct {
	User           refUser   `json:"user"`
	Views          int       `json:"views"`
	LastWatched    time.Time `json:"lastWatched"`
	BestCompletion *float64  `json:"bestCompletion"`
}

func (h *V2Handler) Movie(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var item struct {
		ID           string
		Name         *string
		Year         *int
		RunTimeTicks *int64
		Genres       jsonOf[[]string]
		SizeBytes    *int64
		Width        *int
		LibraryID    *string
	}
	if v2Fail(c, h.raw(c, &item, `
		SELECT i."Id" AS id, i."Name" AS name, i."ProductionYear" AS year, i."RunTimeTicks" AS run_time_ticks,
		       i."Genres"::text AS genres, ii."Size" AS size_bytes,
		       (SELECT (s->>'Width')::int FROM jsonb_array_elements(CASE WHEN jsonb_typeof(ii."MediaStreams") = 'array' THEN ii."MediaStreams" ELSE '[]'::jsonb END) s
		         WHERE s->>'Type' = 'Video' LIMIT 1) AS width,
		       i."ParentId" AS library_id
		FROM jf_library_items i LEFT JOIN jf_item_info ii ON ii."Id" = i."Id"
		WHERE i."Id" = ? AND i."Type" = 'Movie'`, id)) {
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
	f := p.entityFilter().where(`a."MediaType" = 'Movie' AND a."ItemId" = ?`, id)

	var totals struct {
		Views          int      `json:"views"`
		Clicks         int      `json:"clicks"`
		Viewers        int      `json:"viewers"`
		CompletedRatio *float64 `json:"completedRatio"`
		TranscodeRatio *float64 `json:"transcodeRatio"`
		WatchSeconds   int64    `json:"watchSeconds"`
	}
	q, args := f.sql(completionCTE + `
		SELECT ` + aggPlays + ` AS views, ` + aggClicks + ` AS clicks,
		       COUNT(DISTINCT user_id) FILTER (WHERE is_view)::int AS viewers,
		       (SELECT AVG(CASE WHEN ` + completedCond + ` THEN 1.0 ELSE 0.0 END)::float8 FROM completion) AS completed_ratio,
		       (COUNT(*) FILTER (WHERE is_view AND play_method = 'Transcode')::float8 / NULLIF(COUNT(*) FILTER (WHERE is_view), 0)) AS transcode_ratio,
		       ` + aggSeconds + ` AS watch_seconds
		FROM acts`)
	if v2Fail(c, h.raw(c, &totals, q, args...)) {
		return
	}
	fun, err := h.funnel(c, f)
	if v2Fail(c, err) {
		return
	}
	stops, err := h.stopPositions(c, f, 24)
	if v2Fail(c, err) {
		return
	}
	var vrows []struct {
		UserID, UserName string
		Views            int
		LastWatched      time.Time
		BestCompletion   *float64
	}
	q, args = f.sql(`
		SELECT user_id, COALESCE(MAX(user_name), '') AS user_name, COUNT(*)::int AS views,
		       MAX(started_at) AS last_watched, MAX(` + ratioExpr + `)::float8 AS best_completion
		FROM acts WHERE is_view AND user_id IS NOT NULL
		GROUP BY user_id ORDER BY last_watched DESC`)
	if v2Fail(c, h.raw(c, &vrows, q, args...)) {
		return
	}
	viewers := make([]movieViewer, len(vrows))
	for i, v := range vrows {
		viewers[i] = movieViewer{User: refUser{v.UserID, v.UserName}, Views: v.Views, LastWatched: localTime(v.LastWatched), BestCompletion: v.BestCompletion}
	}
	hist, err := h.queryHistory(c, f, ` ORDER BY started_at DESC, acts.id DESC`, 50, 0)
	if v2Fail(c, err) {
		return
	}
	pte, err := h.playedToEnd(c, f)
	if v2Fail(c, err) {
		return
	}
	since, err := h.pluginSince(c, "plays")
	if v2Fail(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"item": gin.H{
			"id": item.ID, "name": strOrEmpty(item.Name), "year": item.Year, "runtimeSeconds": ticksToSeconds(item.RunTimeTicks),
			"genres": strs(item.Genres), "sizeBytes": item.SizeBytes, "resolution": resolutionLabel(item.Width),
			"library": lib,
		},
		"totals":        totals,
		"funnel":        fun,
		"stopPositions": stops,
		"viewers":       viewers,
		"history":       hist,
		"playedToEnd":   pte,
		"pluginSince":   since,
	})
}

// ---------------------------------------------------------------------------
// GET /api/v2/series/:id
// ---------------------------------------------------------------------------

type seriesEpisode struct {
	ID            string   `json:"id"`
	Number        *int     `json:"number"`
	Name          string   `json:"name"`
	Views         int      `json:"views"`
	Viewers       int      `json:"viewers"`
	AvgCompletion *float64 `json:"avgCompletion"`
	Abandons      int      `json:"abandons"`
	// Skips of the closed plugin views (SkipPlays = 0: not measured).
	SkipPlays         int      `json:"skipPlays"`
	IntroRate         *float64 `json:"introRate"`
	SkipRate          *float64 `json:"skipRate"`
	AvgSkippedSeconds *float64 `json:"avgSkippedSeconds"`
}

type seriesSeason struct {
	Number   int             `json:"number"`
	Special  bool            `json:"special,omitempty"` // season 0, always last
	Episodes []seriesEpisode `json:"episodes"`
}

type bucketCount struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
}

var cadenceBuckets = []string{"<1min", "1-15min", "15-60min", "sameDay", "nextDay", "2-7d", ">7d"}
var releaseLagBuckets = []string{"<24h", "1-7d", "1-4w", "1-6m", "6-12m", ">1y"}

func fillBuckets(order []string, rows []bucketCount) []bucketCount {
	m := map[string]int{}
	for _, r := range rows {
		m[r.Bucket] = r.Count
	}
	out := make([]bucketCount, len(order))
	for i, b := range order {
		out[i] = bucketCount{b, m[b]}
	}
	return out
}

func (h *V2Handler) Series(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var item struct {
		ID        string
		Name      *string
		YearFrom  *int
		EndDate   *string
		Genres    jsonOf[[]string]
		Seasons   int
		Episodes  int
		LibraryID *string
	}
	if v2Fail(c, h.raw(c, &item, `
		SELECT i."Id" AS id, i."Name" AS name, i."ProductionYear" AS year_from, i."EndDate" AS end_date, i."Genres"::text AS genres,
		       (SELECT COUNT(DISTINCT COALESCE(e."ParentIndexNumber", 1)) FROM jf_library_episodes e WHERE e."SeriesId" = i."Id" AND NOT e.archived AND `+notSpecial+`)::int AS seasons,
		       (SELECT COUNT(*) FROM jf_library_episodes e WHERE e."SeriesId" = i."Id" AND NOT e.archived AND `+notSpecial+`)::int AS episodes,
		       i."ParentId" AS library_id
		FROM jf_library_items i WHERE i."Id" = ? AND i."Type" = 'Series'`, id)) {
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
	var yearTo *int
	if item.EndDate != nil && len(*item.EndDate) >= 4 {
		var y int
		if _, err := fmt.Sscanf((*item.EndDate)[:4], "%d", &y); err == nil {
			yearTo = &y
		}
	}
	f := p.entityFilter().where(`a."MediaType" = 'Episode' AND COALESCE(ce."SeriesId", a."NowPlayingItemId") = ?`, id)

	// Per-episode stats, catalog order (episodes never played included).
	var eps []struct {
		ID            string
		Season        int
		Special       bool
		Number        *int
		Name          *string
		Views         int
		Viewers       int
		AvgCompletion *float64
		Abandons      int
		SkipCounts
	}
	q, args := f.sql(`,
		`+strings.TrimPrefix(completionCTE, ",")+`,
		ep AS (
		  SELECT item_id, COUNT(*) FILTER (WHERE is_view)::int AS views,
		         COUNT(DISTINCT user_id) FILTER (WHERE is_view)::int AS viewers
		  FROM acts GROUP BY item_id
		),
		comp AS (
		  SELECT item_id, AVG(completion_ratio)::float8 AS avg_completion, COUNT(*) FILTER (WHERE NOT finished)::int AS abandons
		  FROM completion GROUP BY item_id
		),
		sk AS (SELECT item_id, `+skipCols+` FROM acts GROUP BY item_id)
		SELECT e."Id" AS id, COALESCE(e."ParentIndexNumber", 0) AS season, COALESCE(e."ParentIndexNumber" = 0, false) AS special, e."IndexNumber" AS number, e."Name" AS name,
		       COALESCE(ep.views, 0) AS views, COALESCE(ep.viewers, 0) AS viewers,
		       comp.avg_completion, COALESCE(comp.abandons, 0) AS abandons,
		       COALESCE(sk.skip_plays, 0) AS skip_plays, COALESCE(sk.skipped, 0) AS skipped, COALESCE(sk.intro_plays, 0) AS intro_plays, COALESCE(sk.intro, 0) AS intro, sk.avg_skipped_seconds
		FROM jf_library_episodes e
		LEFT JOIN ep ON ep.item_id = e."Id"
		LEFT JOIN comp ON comp.item_id = e."Id"
		LEFT JOIN sk ON sk.item_id = e."Id"
		WHERE e."SeriesId" = ? AND NOT e.archived
		ORDER BY `+episodeOrder, id)
	if v2Fail(c, h.raw(c, &eps, q, args...)) {
		return
	}
	seasons := []seriesSeason{}
	retention := [][]int{}
	for _, e := range eps {
		if len(seasons) == 0 || seasons[len(seasons)-1].Number != e.Season || seasons[len(seasons)-1].Special != e.Special {
			seasons = append(seasons, seriesSeason{Number: e.Season, Special: e.Special, Episodes: []seriesEpisode{}})
			retention = append(retention, []int{})
		}
		s := &seasons[len(seasons)-1]
		s.Episodes = append(s.Episodes, seriesEpisode{ID: e.ID, Number: e.Number, Name: strOrEmpty(e.Name), Views: e.Views, Viewers: e.Viewers, AvgCompletion: e.AvgCompletion, Abandons: e.Abandons,
			SkipPlays: e.SkipPlays, IntroRate: ratioOf(e.Intro, e.IntroPlays), SkipRate: ratioOf(e.Skipped, e.SkipPlays), AvgSkippedSeconds: e.AvgSkippedSeconds})
		retention[len(retention)-1] = append(retention[len(retention)-1], e.Viewers)
	}
	// Retention: viewers of the last regular episode / viewers of the first
	// (specials excluded).
	var retentionRatio *float64
	first, last := -1, -1
	for i, e := range eps {
		if e.Special {
			continue
		}
		if first < 0 {
			first = i
		}
		last = i
	}
	if first >= 0 && eps[first].Viewers > 0 {
		r := float64(eps[last].Viewers) / float64(eps[first].Viewers)
		retentionRatio = &r
	}

	var totals struct {
		WatchSeconds int64 `json:"watchSeconds"`
		Views        int   `json:"views"`
		Clicks       int   `json:"clicks"`
		Viewers      int   `json:"viewers"`
		Binges       int   `json:"binges"`
	}
	q, args = f.sql(bingeCTE + `
		SELECT ` + aggSeconds + ` AS watch_seconds, ` + aggPlays + ` AS views, ` + aggClicks + ` AS clicks,
		       COUNT(DISTINCT user_id) FILTER (WHERE is_view)::int AS viewers,
		       (SELECT COUNT(*) FROM binge_sessions)::int AS binges
		FROM acts`)
	if v2Fail(c, h.raw(c, &totals, q, args...)) {
		return
	}

	var vrows []struct {
		UserID, UserName string
		EpisodesWatched  int
		LastWatched      time.Time
	}
	q, args = f.sql(`
		SELECT user_id, COALESCE(MAX(user_name), '') AS user_name, COUNT(DISTINCT item_id)::int AS episodes_watched,
		       MAX(started_at) AS last_watched
		FROM acts WHERE is_view AND user_id IS NOT NULL
		GROUP BY user_id ORDER BY episodes_watched DESC, last_watched DESC`)
	if v2Fail(c, h.raw(c, &vrows, q, args...)) {
		return
	}
	type seriesViewer struct {
		User            refUser   `json:"user"`
		EpisodesWatched int       `json:"episodesWatched"`
		LastWatched     time.Time `json:"lastWatched"`
	}
	viewers := make([]seriesViewer, len(vrows))
	for i, v := range vrows {
		viewers[i] = seriesViewer{refUser{v.UserID, v.UserName}, v.EpisodesWatched, localTime(v.LastWatched)}
	}

	// Cadence: gap between the end of a view and the start of the user's next
	// view of a different episode of the series.
	var cad []bucketCount
	q, args = f.sql(`,
		seq AS (
		  SELECT item_id, started_at, local_ts,
		         LAG(item_id) OVER w AS prev_item, LAG(ended_at) OVER w AS prev_end, LAG(local_ts) OVER w AS prev_local
		  FROM acts WHERE is_view
		  WINDOW w AS (PARTITION BY user_id ORDER BY started_at)
		),
		gaps AS (
		  SELECT EXTRACT(EPOCH FROM (started_at - prev_end)) AS gap, (local_ts::date - prev_local::date) AS days
		  FROM seq WHERE prev_item IS NOT NULL AND prev_item <> item_id
		)
		SELECT CASE
		         WHEN gap < 60 THEN '<1min'
		         WHEN gap < 900 THEN '1-15min'
		         WHEN gap < 3600 THEN '15-60min'
		         WHEN days = 0 THEN 'sameDay'
		         WHEN days = 1 THEN 'nextDay'
		         WHEN days <= 7 THEN '2-7d'
		         ELSE '>7d'
		       END AS bucket, COUNT(*)::int AS count
		FROM gaps GROUP BY 1`)
	if v2Fail(c, h.raw(c, &cad, q, args...)) {
		return
	}

	// Release lag: first view ever of each (user, episode) minus its premiere
	// date, for first views falling in the range.
	var lag []bucketCount
	allTime := acts().forUser(p.UserID).where(`a."MediaType" = 'Episode' AND COALESCE(ce."SeriesId", a."NowPlayingItemId") = ?`, id)
	q, args = allTime.sql(`,
		firsts AS (
		  SELECT user_id, item_id, MIN(started_at) AS first_view FROM acts WHERE is_view GROUP BY user_id, item_id
		),
		lags AS (
		  SELECT GREATEST(EXTRACT(EPOCH FROM (fv.first_view - NULLIF(e."PremiereDate", '')::timestamptz)), 0) / 86400.0 AS d
		  FROM firsts fv JOIN jf_library_episodes e ON e."Id" = fv.item_id
		  WHERE NULLIF(e."PremiereDate", '') IS NOT NULL AND fv.first_view >= ? AND fv.first_view < ?
		)
		SELECT CASE
		         WHEN d < 1 THEN '<24h'
		         WHEN d < 7 THEN '1-7d'
		         WHEN d < 28 THEN '1-4w'
		         WHEN d < 182 THEN '1-6m'
		         WHEN d < 365 THEN '6-12m'
		         ELSE '>1y'
		       END AS bucket, COUNT(*)::int AS count
		FROM lags GROUP BY 1`, p.Range.From, p.Range.To)
	if v2Fail(c, h.raw(c, &lag, q, args...)) {
		return
	}
	skips, err := h.skips(c, f)
	if v2Fail(c, err) {
		return
	}
	since, err := h.pluginSince(c, "plays")
	if v2Fail(c, err) {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"skips":       skips,
		"pluginSince": since,
		"item": gin.H{
			"id": item.ID, "name": strOrEmpty(item.Name), "years": gin.H{"from": item.YearFrom, "to": yearTo},
			"seasons": item.Seasons, "episodes": item.Episodes, "genres": strs(item.Genres),
			"library": lib,
		},
		"totals": gin.H{
			"watchSeconds": totals.WatchSeconds, "views": totals.Views, "clicks": totals.Clicks,
			"viewers": totals.Viewers, "retention": retentionRatio, "binges": totals.Binges,
		},
		"retention":  retention,
		"seasons":    seasons,
		"viewers":    viewers,
		"cadence":    fillBuckets(cadenceBuckets, cad),
		"releaseLag": fillBuckets(releaseLagBuckets, lag),
	})
}

// ---------------------------------------------------------------------------
// GET /api/v2/episodes/:id   (id = catalog Id or Jellyfin EpisodeId)
// ---------------------------------------------------------------------------

func (h *V2Handler) Episode(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var item struct {
		ID           string
		Name         *string
		SeriesID     *string
		SeriesName   *string
		Season       *int
		Episode      *int
		RunTimeTicks *int64
		PremiereDate *string
		LibraryID    *string
		PrevID       *string
		PrevSeason   *int
		PrevEpisode  *int
		NextID       *string
		NextSeason   *int
		NextEpisode  *int
	}
	if v2Fail(c, h.raw(c, &item, `
		WITH target AS (
		  SELECT * FROM jf_library_episodes WHERE "Id" = ? OR "EpisodeId" = ? ORDER BY ("Id" = ?) DESC LIMIT 1
		),
		ordered AS (
		  SELECT e."Id",
		         LAG(e."Id") OVER w AS prev_id, LAG(e."ParentIndexNumber") OVER w AS prev_season, LAG(e."IndexNumber") OVER w AS prev_episode,
		         LEAD(e."Id") OVER w AS next_id, LEAD(e."ParentIndexNumber") OVER w AS next_season, LEAD(e."IndexNumber") OVER w AS next_episode
		  FROM jf_library_episodes e
		  WHERE e."SeriesId" = (SELECT "SeriesId" FROM target) AND (NOT e.archived OR e."Id" = (SELECT "Id" FROM target))
		    AND COALESCE(e."ParentIndexNumber" = 0, false) = (SELECT COALESCE("ParentIndexNumber" = 0, false) FROM target)
		  WINDOW w AS (ORDER BY `+episodeOrder+`)
		)
		SELECT t."Id" AS id, t."Name" AS name, t."SeriesId" AS series_id,
		       COALESCE(s."Name", t."SeriesName") AS series_name,
		       t."ParentIndexNumber" AS season, t."IndexNumber" AS episode, t."RunTimeTicks" AS run_time_ticks,
		       NULLIF(t."PremiereDate", '') AS premiere_date, s."ParentId" AS library_id,
		       o.prev_id, o.prev_season, o.prev_episode, o.next_id, o.next_season, o.next_episode
		FROM target t
		LEFT JOIN jf_library_items s ON s."Id" = t."SeriesId"
		LEFT JOIN ordered o ON o."Id" = t."Id"`, id, id, id)) {
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
	f := p.entityFilter().where(`a."MediaType" = 'Episode' AND a."ItemId" = ?`, item.ID)

	var totals struct {
		Views            int      `json:"views"`
		Viewers          int      `json:"viewers"`
		AvgCompletion    *float64 `json:"avgCompletion"`
		PausedSecondsAvg *float64 `json:"pausedSecondsAvg"`
	}
	q, args := f.sql(completionCTE + `
		SELECT ` + aggPlays + ` AS views, COUNT(DISTINCT user_id) FILTER (WHERE is_view)::int AS viewers,
		       (SELECT AVG(completion_ratio)::float8 FROM completion) AS avg_completion,
		       AVG(paused_seconds) FILTER (WHERE is_view)::float8 AS paused_seconds_avg
		FROM acts`)
	if v2Fail(c, h.raw(c, &totals, q, args...)) {
		return
	}
	fun, err := h.funnel(c, f)
	if v2Fail(c, err) {
		return
	}
	stops, err := h.stopPositions(c, f, 20)
	if v2Fail(c, err) {
		return
	}
	var vrows []struct {
		UserID, UserName string
		StartedAt        time.Time
		WatchSeconds     int64
		Completion       *float64
		Device           *string
	}
	q, args = f.sql(`
		SELECT COALESCE(user_id, '') AS user_id, COALESCE(user_name, '') AS user_name, started_at, duration AS watch_seconds, ` + ratioExpr + ` AS completion, device_name AS device
		FROM acts WHERE is_view ORDER BY started_at DESC LIMIT 50`)
	if v2Fail(c, h.raw(c, &vrows, q, args...)) {
		return
	}
	type epViewer struct {
		User         refUser   `json:"user"`
		StartedAt    time.Time `json:"startedAt"`
		WatchSeconds int64     `json:"watchSeconds"`
		Completion   *float64  `json:"completion"`
		Device       string    `json:"device"`
	}
	viewers := make([]epViewer, len(vrows))
	for i, v := range vrows {
		viewers[i] = epViewer{refUser{v.UserID, v.UserName}, localTime(v.StartedAt), v.WatchSeconds, v.Completion, strOrEmpty(v.Device)}
	}
	pte, err := h.playedToEnd(c, f)
	if v2Fail(c, err) {
		return
	}
	skips, err := h.skips(c, f)
	if v2Fail(c, err) {
		return
	}
	since, err := h.pluginSince(c, "plays")
	if v2Fail(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"playedToEnd": pte,
		"skips":       skips,
		"pluginSince": since,
		"item": gin.H{
			"id": item.ID, "name": strOrEmpty(item.Name), "seriesId": strOrEmpty(item.SeriesID), "seriesName": strOrEmpty(item.SeriesName),
			"season": item.Season, "episode": item.Episode, "runtimeSeconds": ticksToSeconds(item.RunTimeTicks),
			"premiereDate": premiereDate(item.PremiereDate), "library": lib,
			"prev": episodeRefOf(item.PrevID, item.PrevSeason, item.PrevEpisode),
			"next": episodeRefOf(item.NextID, item.NextSeason, item.NextEpisode),
		},
		"totals":        totals,
		"funnel":        fun,
		"stopPositions": stops,
		"viewers":       viewers,
	})
}

// episodeRef is a neighbouring episode in catalog order: regular episodes
// link across season boundaries, specials only to other specials.
type episodeRef struct {
	ID      string `json:"id"`
	Season  *int   `json:"season"`
	Episode *int   `json:"episode"`
}

func episodeRefOf(id *string, season, episode *int) *episodeRef {
	if id == nil {
		return nil
	}
	return &episodeRef{ID: *id, Season: season, Episode: episode}
}

// premiereDate keeps the YYYY-MM-DD part of a stored premiere timestamp.
func premiereDate(s *string) *string {
	if s == nil || len(*s) < 10 {
		return nil
	}
	v := strings.TrimSpace(*s)[:10]
	return &v
}
