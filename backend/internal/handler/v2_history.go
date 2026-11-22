package handler

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
)

type historyUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type historyItem struct {
	ID         string  `json:"id"`
	Kind       string  `json:"kind"`
	Name       string  `json:"name"`
	ParentID   *string `json:"parentId,omitempty"`
	ParentName *string `json:"parentName,omitempty"`
	Season     *int    `json:"season,omitempty"`
	Episode    *int    `json:"episode,omitempty"`
}

// HistoryRow is one playback attempt as shown in history tables.
type HistoryRow struct {
	ID            string      `json:"id"`
	StartedAt     time.Time   `json:"startedAt"`
	EndedAt       time.Time   `json:"endedAt"`
	User          historyUser `json:"user"`
	Item          historyItem `json:"item"`
	MediaType     string      `json:"mediaType"`
	WatchSeconds  int64       `json:"watchSeconds"`
	Completion    *float64    `json:"completion"`
	Client        string      `json:"client"`
	Device        string      `json:"device"`
	Method        string      `json:"method"`
	IsView        bool        `json:"isView"`
	PausedSeconds int64       `json:"pausedSeconds"`
}

type historyScan struct {
	ID            string
	StartedAt     time.Time
	EndedAt       time.Time
	UserID        *string
	UserName      *string
	ItemID        *string
	Kind          string
	ItemName      *string
	ParentID      *string
	ParentName    *string
	Season        *int
	Episode       *int
	MediaType     string
	WatchSeconds  int64
	Completion    *float64
	Client        *string
	Device        *string
	Method        string
	IsView        bool
	PausedSeconds int64
}

func (s historyScan) row() HistoryRow {
	r := HistoryRow{
		ID: s.ID, StartedAt: localTime(s.StartedAt), EndedAt: localTime(s.EndedAt),
		User:      historyUser{ID: strOrEmpty(s.UserID), Name: strOrEmpty(s.UserName)},
		Item:      historyItem{ID: strOrEmpty(s.ItemID), Kind: s.Kind, Name: strOrEmpty(s.ItemName), Season: s.Season, Episode: s.Episode},
		MediaType: s.MediaType, WatchSeconds: s.WatchSeconds, Completion: s.Completion,
		Client: strOrEmpty(s.Client), Device: strOrEmpty(s.Device), Method: s.Method,
		IsView: s.IsView, PausedSeconds: s.PausedSeconds,
	}
	if s.MediaType != "Movie" {
		r.Item.ParentID, r.Item.ParentName = s.ParentID, s.ParentName
	}
	return r
}

// historySelect selects HistoryRow columns from acts (episode numbers via a
// join on jf_library_episodes aliased he).
var historySelect = `
	SELECT acts.id, started_at, ended_at, user_id, user_name, item_id, ` + itemKindExpr + ` AS kind,
	       item_name, group_id AS parent_id, group_name AS parent_name,
	       he."ParentIndexNumber" AS season, he."IndexNumber" AS episode,
	       media_type, duration AS watch_seconds, ` + ratioExpr + ` AS completion,
	       client, device_name AS device, play_method AS method, is_view, paused_seconds
	FROM acts
	LEFT JOIN jf_library_episodes he ON acts.media_type = 'Episode' AND he."Id" = acts.item_id`

// v2HistoryFilter applies the history filters (client, device, method, views, q)
// on top of the common contract.
func v2HistoryFilter(c *gin.Context, p V2Params, tr TimeRange) *actFilter {
	f := p.filter(tr)
	if v := strings.TrimSpace(c.Query("client")); v != "" {
		f.where(`a."Client" = ?`, v)
	}
	if v := strings.TrimSpace(c.Query("device")); v != "" {
		f.where(`a."DeviceName" = ?`, v)
	}
	if v := strings.TrimSpace(c.Query("method")); v != "" {
		f.where(`a."PlayMethod" = ?`, methodValue(v))
	}
	if parseBool(c.Query("views")) {
		f.where(`a."IsView"`)
	}
	if p.Q != "" {
		like := likePattern(p.Q)
		f.where(`(COALESCE(cm."Name", ce."Name", ct."Name", a."NowPlayingItemName") ILIKE ?
		     OR COALESCE(cs."Name", NULLIF(a."SeriesName", ''), cal."Name", ct."AlbumName") ILIKE ?
		     OR COALESCE(u."Name", a."UserName") ILIKE ?)`, like, like, like)
	}
	return f
}

var historySorts = map[string]string{
	"startedAt":    "started_at",
	"watchSeconds": "duration",
	"completion":   ratioExpr,
}

// historyOrder returns the ORDER BY clause (default startedAt desc; desc is
// assumed unless desc=0/false is given explicitly).
func historyOrder(c *gin.Context, p V2Params) string {
	expr, ok := historySorts[p.Sort]
	if !ok {
		expr = "started_at"
	}
	dir := "DESC NULLS LAST"
	if c.Query("desc") != "" && !p.Desc {
		dir = "ASC NULLS FIRST"
	}
	return " ORDER BY " + expr + " " + dir + ", acts.id DESC"
}

// queryHistory returns up to limit rows (limit <= 0: all).
func (h *V2Handler) queryHistory(c *gin.Context, f *actFilter, order string, limit, offset int) ([]HistoryRow, error) {
	var scans []historyScan
	outer := historySelect + order
	var outerArgs []interface{}
	if limit > 0 {
		outer += ` LIMIT ? OFFSET ?`
		outerArgs = append(outerArgs, limit, offset)
	}
	q, args := f.sql(outer, outerArgs...)
	if err := h.raw(c, &scans, q, args...); err != nil {
		return nil, err
	}
	rows := make([]HistoryRow, len(scans))
	for i, s := range scans {
		rows[i] = s.row()
	}
	return rows, nil
}

// GET /api/v2/history
func (h *V2Handler) History(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	f := v2HistoryFilter(c, p, p.Range)
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

// GET /api/v2/history.csv — same filters as /history, every row, CSV.
func (h *V2Handler) HistoryCSV(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	f := v2HistoryFilter(c, p, p.Range)
	q, args := f.sql(historySelect + historyOrder(c, p))
	rows, err := h.db.WithContext(c.Request.Context()).Raw(q, args...).Rows()
	if v2Fail(c, err) {
		return
	}
	defer rows.Close()

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="history_`+localDate(p.Range.From)+`_`+localDate(p.Range.To)+`.csv"`)
	c.Status(http.StatusOK)
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"startedAt", "endedAt", "userId", "userName", "itemId", "kind", "itemName", "parentId", "parentName", "season", "episode", "mediaType", "watchSeconds", "completion", "client", "device", "method", "isView", "pausedSeconds"})
	optInt := func(v *int) string {
		if v == nil {
			return ""
		}
		return strconv.Itoa(*v)
	}
	for rows.Next() {
		var s historyScan
		if err := h.db.ScanRows(rows, &s); err != nil {
			logV2(c, err)
			break
		}
		r := s.row()
		comp := ""
		if r.Completion != nil {
			comp = strconv.FormatFloat(*r.Completion, 'f', 4, 64)
		}
		_ = w.Write([]string{
			r.StartedAt.Format(time.RFC3339), r.EndedAt.Format(time.RFC3339), r.User.ID, r.User.Name,
			r.Item.ID, r.Item.Kind, r.Item.Name, strOrEmpty(r.Item.ParentID), strOrEmpty(r.Item.ParentName),
			optInt(r.Item.Season), optInt(r.Item.Episode), r.MediaType, strconv.FormatInt(r.WatchSeconds, 10), comp,
			r.Client, r.Device, r.Method, strconv.FormatBool(r.IsView), strconv.FormatInt(r.PausedSeconds, 10),
		})
	}
	if err := rows.Err(); err != nil {
		logV2(c, err)
	}
	w.Flush()
}

type calendarItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	User string `json:"user"`
}

type calendarDay struct {
	Date         string         `json:"date"`
	Views        int            `json:"views"`
	WatchSeconds int64          `json:"watchSeconds"`
	Items        []calendarItem `json:"items"`
}

// GET /api/v2/history/calendar?month=YYYY-MM (default: current local month).
// Every day of the month is returned; items = up to 6 distinct titles viewed
// that day (latest first).
func (h *V2Handler) HistoryCalendar(c *gin.Context) {
	p, ok := v2Params(c)
	if !ok {
		return
	}
	loc := statsql.Location()
	var start time.Time
	if m := strings.TrimSpace(c.Query("month")); m != "" {
		t, err := time.ParseInLocation("2006-01", m, loc)
		if err != nil || t.Format("2006-01") != m {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month"})
			return
		}
		start = t
	} else {
		now := time.Now().In(loc)
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	}
	tr := TimeRange{From: start, To: start.AddDate(0, 1, 0)}
	f := v2HistoryFilter(c, p, tr)

	var totals []struct {
		Date         string
		Views        int
		WatchSeconds int64
	}
	q, args := f.sql(`SELECT ` + localDay + ` AS date, ` + aggPlays + ` AS views, ` + aggSeconds + ` AS watch_seconds FROM acts GROUP BY 1`)
	if v2Fail(c, h.raw(c, &totals, q, args...)) {
		return
	}
	var items []struct {
		Date  string
		Items jsonOf[[]calendarItem]
	}
	q, args = f.sql(`,
		per AS (
		  SELECT local_ts::date AS day, group_id, COALESCE(MAX(group_name), '') AS name, ` + kindExpr + ` AS kind,
		         (array_agg(user_name ORDER BY started_at DESC))[1] AS usr, MAX(started_at) AS last
		  FROM acts WHERE is_view AND group_id IS NOT NULL
		  GROUP BY local_ts::date, group_id, display_type
		),
		ranked AS (SELECT *, ROW_NUMBER() OVER (PARTITION BY day ORDER BY last DESC, group_id) AS rn FROM per)
		SELECT to_char(day, 'YYYY-MM-DD') AS date,
		       json_agg(json_build_object('id', group_id, 'name', name, 'kind', kind, 'user', COALESCE(usr, '')) ORDER BY rn)::text AS items
		FROM ranked WHERE rn <= 6 GROUP BY day`)
	if v2Fail(c, h.raw(c, &items, q, args...)) {
		return
	}
	byDate := map[string]*calendarDay{}
	out := []calendarDay{}
	eachDay(tr, func(d string) { out = append(out, calendarDay{Date: d, Items: []calendarItem{}}) })
	for i := range out {
		byDate[out[i].Date] = &out[i]
	}
	for _, t := range totals {
		if d := byDate[t.Date]; d != nil {
			d.Views, d.WatchSeconds = t.Views, t.WatchSeconds
		}
	}
	for _, it := range items {
		if d := byDate[it.Date]; d != nil && it.Items.V != nil {
			d.Items = it.Items.V
		}
	}
	c.JSON(http.StatusOK, out)
}
