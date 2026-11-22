package handler

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
)

// RegisterAnalytics mounts the phase-5 analysis routes (insights, discover,
// playback, recap, user languages) on g (the authed group).
func (h *V2Handler) RegisterAnalytics(g *gin.RouterGroup) {
	v2 := g.Group("/api/v2")
	v2.GET("/insights/completion", h.InsightsCompletion)
	v2.GET("/insights/sessions", h.InsightsSessions)
	v2.GET("/insights/resumes", h.InsightsResumes)
	v2.GET("/insights/rewatches", h.InsightsRewatches)
	v2.GET("/insights/genres", h.InsightsGenres)
	v2.GET("/insights/moments", h.InsightsMoments)
	v2.GET("/insights/pauses", h.InsightsPauses)
	v2.GET("/insights/binges", h.InsightsBinges)
	v2.GET("/insights/rhythm", h.InsightsRhythm)

	v2.GET("/discover/unwatched", h.DiscoverUnwatched)
	v2.GET("/discover/abandoned", h.DiscoverAbandoned)
	v2.GET("/discover/added", h.DiscoverAdded)
	v2.GET("/discover/storage", h.DiscoverStorage)

	v2.GET("/playback/methods", h.PlaybackMethods)
	v2.GET("/playback/clients", h.PlaybackClients)
	v2.GET("/playback/quality", h.PlaybackQuality)
	v2.GET("/playback/network", h.PlaybackNetwork)

	v2.GET("/recap/:userId/:period", h.Recap)
	v2.GET("/users/:id/languages", h.UserLanguages)
}

// resumeTicks: a view starting at or after 30 s into the item is a resume.
const resumeTicks = 30 * 10_000_000

// byTypeRatio builds a ByType of ratios n/d (nil when d == 0).
func byTypeRatio(n, d map[string]int) ByType[*float64] {
	r := func(mt string) *float64 {
		if d[mt] == 0 {
			return nil
		}
		v := float64(n[mt]) / float64(d[mt])
		return &v
	}
	return ByType[*float64]{Movie: r("Movie"), Series: r("Episode"), Music: r("Audio")}
}

// ---------------------------------------------------------------------------
// GET /api/v2/insights/completion
// ---------------------------------------------------------------------------

type completionInsight struct {
	// Distribution counts the (user, item, day) view units per 10 % bucket of
	// completion (index 9 = 90–100 %).
	Distribution []int `json:"distribution"`
	// ByType is the share of completed units (≥ 90 %) per media family.
	ByType ByType[*float64] `json:"byType"`
	Units  int              `json:"units"`
}

func (h *V2Handler) InsightsCompletion(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	var rows []struct {
		Bucket   int
		ItemType string
		N, Done  int
	}
	q, args := p.filter(p.Range).sql(completionCTE + `
		SELECT LEAST(GREATEST(FLOOR(completion_ratio * 10)::int, 0), 9) AS bucket, item_type,
		       COUNT(*)::int AS n, COUNT(*) FILTER (WHERE ` + completedCond + `)::int AS done
		FROM completion GROUP BY 1, 2`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	out := completionInsight{Distribution: make([]int, 10)}
	n, d := map[string]int{}, map[string]int{}
	for _, r := range rows {
		out.Distribution[r.Bucket] += r.N
		out.Units += r.N
		d[r.ItemType] += r.N
		n[r.ItemType] += r.Done
	}
	out.ByType = byTypeRatio(n, d)
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/v2/insights/sessions — length of the views (false starts excluded)
// ---------------------------------------------------------------------------

// sessionBounds are the upper bounds (seconds, exclusive) of the session
// length buckets: < 1 min, 1–5, 5–15, 15–30, 30–45, 45–60 min, 1–1.5 h,
// 1.5–2 h, ≥ 2 h.
var sessionBounds = []int64{60, 300, 900, 1800, 2700, 3600, 5400, 7200}

type sessionsInsight struct {
	Buckets       []int    `json:"buckets"`
	Sessions      int      `json:"sessions"`
	MedianSeconds *float64 `json:"medianSeconds"`
}

func (h *V2Handler) InsightsSessions(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	cases := "CASE"
	for i, b := range sessionBounds {
		cases += fmt.Sprintf(" WHEN duration < %d THEN %d", b, i)
	}
	cases += fmt.Sprintf(" ELSE %d END", len(sessionBounds))
	var rows []struct{ Bucket, N int }
	f := p.filter(p.Range).where(`a."IsView"`)
	q, args := f.sql(`SELECT ` + cases + ` AS bucket, COUNT(*)::int AS n FROM acts GROUP BY 1`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	out := sessionsInsight{Buckets: make([]int, len(sessionBounds)+1)}
	for _, r := range rows {
		out.Buckets[r.Bucket] = r.N
		out.Sessions += r.N
	}
	var med struct{ Median *float64 }
	q, args = f.sql(`SELECT (percentile_cont(0.5) WITHIN GROUP (ORDER BY duration))::float8 AS median FROM acts`)
	if v2Fail(c, h.raw(c, &med, q, args...)) {
		return
	}
	out.MedianSeconds = med.Median
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/v2/insights/resumes — views started from the beginning vs resumed
// ---------------------------------------------------------------------------

type resumesInsight struct {
	FromStart int `json:"fromStart"`
	Resumed   int `json:"resumed"`
	// ByType is the share of resumed views per family (nil without data).
	ByType ByType[*float64] `json:"byType"`
}

func (h *V2Handler) InsightsResumes(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	var rows []struct {
		MediaType string
		N         int
		Resumed   int
	}
	q, args := p.filter(p.Range).sql(`
		SELECT media_type, COUNT(*)::int AS n, COUNT(*) FILTER (WHERE pos_start >= ?)::int AS resumed
		FROM acts WHERE is_view AND pos_start IS NOT NULL GROUP BY 1`, resumeTicks)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	var out resumesInsight
	n, d := map[string]int{}, map[string]int{}
	for _, r := range rows {
		out.Resumed += r.Resumed
		out.FromStart += r.N - r.Resumed
		n[r.MediaType], d[r.MediaType] = r.Resumed, r.N
	}
	out.ByType = byTypeRatio(n, d)
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/v2/insights/rewatches — items viewed several times by one user
// ---------------------------------------------------------------------------

type rewatchRow struct {
	ID         string      `json:"id"`
	Kind       string      `json:"kind"` // movie | episode | track
	Name       string      `json:"name"`
	ParentID   *string     `json:"parentId"`   // series (episode) or album (track)
	ParentName *string     `json:"parentName"` // idem
	Views      int         `json:"views"`
	User       RefUserJSON `json:"user"`
}

// RefUserJSON is a {id, name} user reference.
type RefUserJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// InsightsRewatches: per (user, item), movies and episodes count distinct
// local days with a view, tracks count views. Items reaching ≥ 2 for one user
// are listed with their best user (top 10).
func (h *V2Handler) InsightsRewatches(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	var rows []struct {
		ID, Kind, Name       string
		ParentID, ParentName *string
		Views                int
		UserID, UserName     string
	}
	q, args := p.filter(p.Range).sql(`,
		per AS (
		  SELECT item_id, user_id, MAX(user_name) AS user_name, MAX(media_type) AS media_type,
		         MAX(item_name) AS item_name, MAX(group_id) AS group_id, MAX(group_name) AS group_name,
		         CASE WHEN MAX(media_type) = 'Audio' THEN COUNT(*) ELSE COUNT(DISTINCT local_ts::date) END::int AS views
		  FROM acts WHERE is_view AND item_id IS NOT NULL AND user_id IS NOT NULL
		  GROUP BY item_id, user_id
		),
		best AS (
		  SELECT *, ROW_NUMBER() OVER (PARTITION BY item_id ORDER BY views DESC, user_name, user_id) AS rn FROM per WHERE views >= 2
		)
		SELECT item_id AS id, CASE media_type WHEN 'Movie' THEN 'movie' WHEN 'Episode' THEN 'episode' ELSE 'track' END AS kind,
		       COALESCE(item_name, '') AS name,
		       CASE WHEN media_type = 'Movie' THEN NULL ELSE group_id END AS parent_id,
		       CASE WHEN media_type = 'Movie' THEN NULL ELSE group_name END AS parent_name,
		       views, user_id, COALESCE(user_name, user_id) AS user_name
		FROM best WHERE rn = 1
		ORDER BY views DESC, name, id LIMIT 10`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	out := make([]rewatchRow, len(rows))
	for i, r := range rows {
		out[i] = rewatchRow{ID: r.ID, Kind: r.Kind, Name: r.Name, ParentID: r.ParentID, ParentName: r.ParentName,
			Views: r.Views, User: RefUserJSON{ID: r.UserID, Name: r.UserName}}
	}
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/v2/insights/genres — watch time per month for the top 3 genres
// ---------------------------------------------------------------------------

type genreSeries struct {
	Name   string  `json:"name"`
	Values []int64 `json:"values"`
}

type genresInsight struct {
	Bucket string        `json:"bucket"` // always "month"
	Months []string      `json:"months"` // YYYY-MM, every month overlapping the range (leading empty ones trimmed on long ranges)
	Series []genreSeries `json:"series"` // top 3 genres by watch time, best first
	Other  []int64       `json:"other"`  // every other genre
}

// InsightsGenres attributes each playback row to the first genre of its title
// (catalog), so the stacked series sum to the watch time of the rows with a
// genre, without double counting.
func (h *V2Handler) InsightsGenres(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	var rows []struct {
		Month, Genre string
		Seconds      int64
	}
	q, args := p.filter(p.Range).sql(`
		SELECT to_char(local_ts, 'YYYY-MM') AS month, genres->>0 AS genre, ` + aggSeconds + ` AS seconds
		FROM acts WHERE jsonb_typeof(genres) = 'array' AND genres->>0 IS NOT NULL AND genres->>0 <> ''
		GROUP BY 1, 2`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	out := genresInsight{Bucket: "month", Months: monthsOf(p.Range), Series: []genreSeries{}}
	idx := map[string]int{}
	for i, m := range out.Months {
		idx[m] = i
	}
	total := map[string]int64{}
	for _, r := range rows {
		total[r.Genre] += r.Seconds
	}
	names := make([]string, 0, len(total))
	for g := range total {
		names = append(names, g)
	}
	sort.Slice(names, func(i, j int) bool {
		if total[names[i]] != total[names[j]] {
			return total[names[i]] > total[names[j]]
		}
		return names[i] < names[j]
	})
	top := map[string]int{}
	for i, g := range names {
		if i >= 3 {
			break
		}
		top[g] = i
		out.Series = append(out.Series, genreSeries{Name: g, Values: make([]int64, len(out.Months))})
	}
	out.Other = make([]int64, len(out.Months))
	for _, r := range rows {
		mi, ok := idx[r.Month]
		if !ok {
			continue
		}
		if ti, ok := top[r.Genre]; ok {
			out.Series[ti].Values[mi] += r.Seconds
		} else {
			out.Other[mi] += r.Seconds
		}
	}
	if useWeeks(p.Range) {
		start := leadingEmpty(len(out.Months), func(i int) bool {
			if out.Other[i] != 0 {
				return false
			}
			for _, s := range out.Series {
				if s.Values[i] != 0 {
					return false
				}
			}
			return true
		})
		out.Months, out.Other = out.Months[start:], out.Other[start:]
		for i := range out.Series {
			out.Series[i].Values = out.Series[i].Values[start:]
		}
	}
	c.JSON(http.StatusOK, out)
}

// leadingEmpty is the index of the first non-empty period among n (the last
// one when all are empty, so at least one period is kept).
func leadingEmpty(n int, empty func(i int) bool) int {
	i := 0
	for i < n-1 && empty(i) {
		i++
	}
	return i
}

// monthsOf lists the local months (YYYY-MM) overlapping tr.
func monthsOf(tr TimeRange) []string {
	loc := statsql.Location()
	from, to := tr.From.In(loc), tr.To.In(loc)
	out := []string{}
	for m := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, loc); m.Before(to); m = m.AddDate(0, 1, 0) {
		out = append(out, m.Format("2006-01"))
	}
	return out
}

// ---------------------------------------------------------------------------
// GET /api/v2/insights/moments — views per local hour and family
// ---------------------------------------------------------------------------

func (h *V2Handler) InsightsMoments(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	var rows []struct {
		MediaType string
		Hour, N   int
	}
	q, args := p.filter(p.Range).sql(`
		SELECT media_type, EXTRACT(HOUR FROM local_ts)::int AS hour, COUNT(*)::int AS n
		FROM acts WHERE is_view GROUP BY 1, 2`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	out := ByType[[]int]{Movie: make([]int, 24), Series: make([]int, 24), Music: make([]int, 24)}
	for _, r := range rows {
		if r.Hour < 0 || r.Hour > 23 {
			continue
		}
		switch r.MediaType {
		case "Movie":
			out.Movie[r.Hour] = r.N
		case "Episode":
			out.Series[r.Hour] = r.N
		case "Audio":
			out.Music[r.Hour] = r.N
		}
	}
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/v2/insights/pauses — paused time / watched time per title
// ---------------------------------------------------------------------------

type pauseRow struct {
	ID            string  `json:"id"`
	Kind          string  `json:"kind"` // movie | series | album
	Name          string  `json:"name"`
	Ratio         float64 `json:"ratio"`
	PausedSeconds int64   `json:"pausedSeconds"`
	PauseCount    int     `json:"pauseCount"`
	WatchSeconds  int64   `json:"watchSeconds"`
}

func (h *V2Handler) InsightsPauses(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	out := []pauseRow{}
	q, args := p.filter(p.Range).sql(`
		SELECT group_id AS id, ` + kindExpr + ` AS kind, COALESCE(MAX(group_name), '') AS name,
		       COALESCE(SUM(paused_seconds), 0)::bigint AS paused_seconds, ` + aggSeconds + ` AS watch_seconds,
		       COALESCE(SUM(pa."PauseCount"), 0)::int AS pause_count,
		       (COALESCE(SUM(paused_seconds), 0)::float8 / NULLIF(SUM(duration), 0)::float8) AS ratio
		FROM acts JOIN jf_playback_activity pa ON pa."Id" = acts.id
		WHERE group_id IS NOT NULL
		GROUP BY group_id, display_type
		HAVING SUM(paused_seconds) > 0 AND SUM(duration) > 0
		ORDER BY ratio DESC, name, id LIMIT 10`)
	if v2Fail(c, h.raw(c, &out, q, args...)) {
		return
	}
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/v2/insights/binges — series with the most binge days
// ---------------------------------------------------------------------------

type bingeRow struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Binges int    `json:"binges"`
	Users  int    `json:"users"`
}

func (h *V2Handler) InsightsBinges(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	out := []bingeRow{}
	q, args := p.filter(p.Range).sql(bingeCTE + `
		SELECT series_id AS id, COALESCE(MAX(series_name), '') AS name, COUNT(*)::int AS binges,
		       COUNT(DISTINCT user_id)::int AS users
		FROM binge_sessions WHERE series_id IS NOT NULL
		GROUP BY series_id ORDER BY binges DESC, name, id LIMIT 10`)
	if v2Fail(c, h.raw(c, &out, q, args...)) {
		return
	}
	c.JSON(http.StatusOK, out)
}

// GET /api/v2/insights/rhythm → {weekHour: views[7][24]} (Monday first).
func (h *V2Handler) InsightsRhythm(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	g, err := h.weekHour(c, p.filter(p.Range))
	if v2Fail(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"weekHour": g})
}
