package handler

import (
	"net/http"
	"time"

	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
)

// TypeMetrics is a {watchSeconds, views, clicks} triple.
type TypeMetrics struct {
	WatchSeconds int64 `json:"watchSeconds"`
	Views        int   `json:"views"`
	Clicks       int   `json:"clicks"`
}

// ByType splits a metric by media family.
type ByType[T any] struct {
	Movie  T `json:"movie"`
	Series T `json:"series"`
	Music  T `json:"music"`
}

type dailyPoint struct {
	Date         string              `json:"date"`
	WatchSeconds int64               `json:"watchSeconds"`
	Views        int                 `json:"views"`
	Clicks       int                 `json:"clicks"`
	ByType       ByType[TypeMetrics] `json:"byType"`
}

type topItem struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	Name         string `json:"name"`
	Views        int    `json:"views"`
	WatchSeconds int64  `json:"watchSeconds"`
	ArtID        string `json:"artId"`
}

type topUser struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	WatchSeconds int64  `json:"watchSeconds"`
	Views        int    `json:"views"`
}

type methodSplit struct {
	Direct    int `json:"direct"`
	Stream    int `json:"stream"`
	Transcode int `json:"transcode"`
}

type peakPoint struct {
	Date string `json:"date"`
	Peak int    `json:"peak"`
}

type onThisDayRow struct {
	ItemID string   `json:"itemId"`
	Kind   string   `json:"kind"`
	Name   string   `json:"name"`
	Users  []string `json:"users"`
	Date   string   `json:"date"`
}

type overviewResponse struct {
	Totals          Totals         `json:"totals"`
	Previous        *Totals        `json:"previous,omitempty"`
	Daily           []dailyPoint   `json:"daily"`
	PreviousDaily   []dailyPoint   `json:"previousDaily,omitempty"`
	TopItems        []topItem      `json:"topItems"`
	TopUsers        []topUser      `json:"topUsers"`
	ByMethod        methodSplit    `json:"byMethod"`
	WeekHour        [][]int        `json:"weekHour"`
	ConcurrencyPeak []peakPoint    `json:"concurrencyPeak"`
	OnThisDay       []onThisDayRow `json:"onThisDay"`
}

// GET /api/v2/overview
func (h *V2Handler) Overview(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	f := p.filter(p.Range)
	var out overviewResponse
	var err error

	if out.Totals, err = h.totals(c, f); v2Fail(c, err) {
		return
	}
	if out.Daily, err = h.daily(c, f, p.Range); v2Fail(c, err) {
		return
	}
	if p.Prev != nil {
		pf := p.filter(*p.Prev)
		prev, err := h.totals(c, pf)
		if v2Fail(c, err) {
			return
		}
		out.Previous = &prev
		if out.PreviousDaily, err = h.daily(c, pf, *p.Prev); v2Fail(c, err) {
			return
		}
	}

	// Top items: the 10 best of each kind (movie / series / album) by views.
	out.TopItems = []topItem{}
	q, args := f.sql(`
		SELECT id, kind, name, views, watch_seconds, id AS art_id FROM (
		  SELECT group_id AS id, ` + kindExpr + ` AS kind, COALESCE(MAX(group_name), '') AS name,
		         ` + aggPlays + ` AS views, ` + aggSeconds + ` AS watch_seconds,
		         ROW_NUMBER() OVER (PARTITION BY ` + kindExpr + ` ORDER BY ` + aggPlays + ` DESC, ` + aggSeconds + ` DESC, group_id) AS rn
		  FROM acts WHERE group_id IS NOT NULL
		  GROUP BY group_id, display_type
		  HAVING ` + aggPlays + ` > 0
		) t WHERE rn <= 10
		ORDER BY views DESC, watch_seconds DESC, id`)
	if v2Fail(c, h.raw(c, &out.TopItems, q, args...)) {
		return
	}

	out.TopUsers = []topUser{}
	q, args = f.sql(`
		SELECT user_id AS id, COALESCE(MAX(user_name), '') AS name,
		       ` + aggSeconds + ` AS watch_seconds, ` + aggPlays + ` AS views
		FROM acts WHERE user_id IS NOT NULL
		GROUP BY user_id
		ORDER BY watch_seconds DESC, views DESC, user_id
		LIMIT 10`)
	if v2Fail(c, h.raw(c, &out.TopUsers, q, args...)) {
		return
	}

	q, args = f.sql(`
		SELECT COUNT(*) FILTER (WHERE play_method = 'DirectPlay')::int AS direct,
		       COUNT(*) FILTER (WHERE play_method = 'DirectStream')::int AS stream,
		       COUNT(*) FILTER (WHERE play_method = 'Transcode')::int AS transcode
		FROM acts WHERE is_view`)
	if v2Fail(c, h.raw(c, &out.ByMethod, q, args...)) {
		return
	}

	if out.WeekHour, err = h.weekHour(c, f); v2Fail(c, err) {
		return
	}
	if out.ConcurrencyPeak, err = h.concurrencyPeak(c, f, p.Range); v2Fail(c, err) {
		return
	}
	if out.OnThisDay, err = h.onThisDay(c, p, time.Now()); v2Fail(c, err) {
		return
	}
	c.JSON(http.StatusOK, out)
}

// daily returns one point per local day of tr (zero-filled).
func (h *V2Handler) daily(c *gin.Context, f *actFilter, tr TimeRange) ([]dailyPoint, error) {
	type row struct {
		Date                                      string
		WatchSeconds                              int64
		Views, Clicks                             int
		MovieSeconds, SeriesSeconds, MusicSeconds int64
		MovieViews, SeriesViews, MusicViews       int
		MovieClicks, SeriesClicks, MusicClicks    int
	}
	var rows []row
	q, args := f.sql(`
		SELECT ` + localDay + ` AS date, ` + aggSeconds + ` AS watch_seconds, ` + aggPlays + ` AS views, ` + aggClicks + ` AS clicks,
		  COALESCE(SUM(duration) FILTER (WHERE media_type = 'Movie'), 0)::bigint AS movie_seconds,
		  COALESCE(SUM(duration) FILTER (WHERE media_type = 'Episode'), 0)::bigint AS series_seconds,
		  COALESCE(SUM(duration) FILTER (WHERE media_type = 'Audio'), 0)::bigint AS music_seconds,
		  COUNT(*) FILTER (WHERE is_view AND media_type = 'Movie')::int AS movie_views,
		  COUNT(*) FILTER (WHERE is_view AND media_type = 'Episode')::int AS series_views,
		  COUNT(*) FILTER (WHERE is_view AND media_type = 'Audio')::int AS music_views,
		  COUNT(*) FILTER (WHERE media_type = 'Movie')::int AS movie_clicks,
		  COUNT(*) FILTER (WHERE media_type = 'Episode')::int AS series_clicks,
		  COUNT(*) FILTER (WHERE media_type = 'Audio')::int AS music_clicks
		FROM acts GROUP BY 1`)
	if err := h.raw(c, &rows, q, args...); err != nil {
		return nil, err
	}
	byDate := make(map[string]row, len(rows))
	for _, r := range rows {
		byDate[r.Date] = r
	}
	out := []dailyPoint{}
	eachDay(tr, func(d string) {
		r := byDate[d]
		out = append(out, dailyPoint{
			Date: d, WatchSeconds: r.WatchSeconds, Views: r.Views, Clicks: r.Clicks,
			ByType: ByType[TypeMetrics]{
				Movie:  TypeMetrics{r.MovieSeconds, r.MovieViews, r.MovieClicks},
				Series: TypeMetrics{r.SeriesSeconds, r.SeriesViews, r.SeriesClicks},
				Music:  TypeMetrics{r.MusicSeconds, r.MusicViews, r.MusicClicks},
			},
		})
	})
	return out, nil
}

// weekHour returns views per (ISO weekday − 1, local hour): Monday first.
func (h *V2Handler) weekHour(c *gin.Context, f *actFilter) ([][]int, error) {
	var rows []struct{ Dow, Hour, Views int }
	q, args := f.sql(`
		SELECT (EXTRACT(ISODOW FROM local_ts)::int - 1) AS dow, EXTRACT(HOUR FROM local_ts)::int AS hour,
		       COUNT(*)::int AS views
		FROM acts WHERE is_view GROUP BY 1, 2`)
	if err := h.raw(c, &rows, q, args...); err != nil {
		return nil, err
	}
	g := weekHourGrid()
	for _, r := range rows {
		if r.Dow >= 0 && r.Dow < 7 && r.Hour >= 0 && r.Hour < 24 {
			g[r.Dow][r.Hour] = r.Views
		}
	}
	return g, nil
}

// concurrencyPeak returns, per local day of tr, the maximum number of
// playback rows running at the same time (zero-filled).
func (h *V2Handler) concurrencyPeak(c *gin.Context, f *actFilter, tr TimeRange) ([]peakPoint, error) {
	var rows []peakPoint
	q, args := f.sql(`,
		ev AS (
		  SELECT started_at AS ts, 1 AS d FROM acts
		  UNION ALL
		  SELECT GREATEST(ended_at, started_at), -1 FROM acts
		),
		run AS (
		  SELECT ts, d, SUM(d) OVER (ORDER BY ts, d ROWS UNBOUNDED PRECEDING) AS n FROM ev
		)
		SELECT to_char(` + statsql.AtLocal("ts") + `::date, 'YYYY-MM-DD') AS date, MAX(n)::int AS peak
		FROM run WHERE d = 1 GROUP BY 1`)
	if err := h.raw(c, &rows, q, args...); err != nil {
		return nil, err
	}
	byDate := make(map[string]int, len(rows))
	for _, r := range rows {
		byDate[r.Date] = r.Peak
	}
	out := []peakPoint{}
	eachDay(tr, func(d string) { out = append(out, peakPoint{Date: d, Peak: byDate[d]}) })
	return out, nil
}

// onThisDay lists what was viewed on today's local month-day in earlier years
// (user / type / library filters apply, the range does not).
func (h *V2Handler) onThisDay(c *gin.Context, p V2Params, now time.Time) ([]onThisDayRow, error) {
	loc := statsql.Location()
	today := statsql.StartOfDay(now, loc)
	f := p.filter(TimeRange{AllTime: true}).
		where(`to_char(`+statsql.LocalTS("a")+`, 'MM-DD') = ?`, today.Format("01-02")).
		where(statsql.StartedAt("a")+` < ?`, today).
		where(`a."IsView"`)
	var rows []struct {
		ItemID string
		Kind   string
		Name   string
		Users  jsonOf[[]string]
		Date   string
	}
	q, args := f.sql(`
		SELECT group_id AS item_id, ` + kindExpr + ` AS kind, COALESCE(MAX(group_name), '') AS name,
		       json_agg(DISTINCT user_name)::text AS users, ` + localDay + ` AS date
		FROM acts WHERE group_id IS NOT NULL
		GROUP BY group_id, display_type, local_ts::date
		ORDER BY date DESC, name
		LIMIT 12`)
	if err := h.raw(c, &rows, q, args...); err != nil {
		return nil, err
	}
	out := make([]onThisDayRow, 0, len(rows))
	for _, r := range rows {
		users := r.Users.V
		if users == nil {
			users = []string{}
		}
		out = append(out, onThisDayRow{ItemID: r.ItemID, Kind: r.Kind, Name: r.Name, Users: users, Date: r.Date})
	}
	return out, nil
}
