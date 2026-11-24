package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// /api/me: the « Mes statistiques » page, relayed by the Jellystics plugin
// (middleware.PluginMe sets the authenticated Jellyfin user). Every endpoint
// reuses the /api/v2 SQL with the user filter forced from the context: any
// userId / users query parameter is ignored. Responses are explicit structs
// (whitelists): no IP / RemoteEndPoint, no geo, no other user's name.

// MeHandler serves /api/me.
type MeHandler struct{ v2 *V2Handler }

// NewMeHandler builds the /api/me handler.
func NewMeHandler(db *gorm.DB) *MeHandler { return &MeHandler{v2: NewV2Handler(db)} }

// Register mounts the user endpoints on g (the /api/me group guarded by
// middleware.PluginMe with a required user). Health is mounted separately
// (secret only).
func (m *MeHandler) Register(g *gin.RouterGroup) {
	g.GET("/profile", m.Profile)
	g.GET("/overview", m.Overview)
	g.GET("/top", m.Top)
	g.GET("/history", m.History)
	g.GET("/types", m.Types)
	g.GET("/habits", m.Habits)
	g.GET("/devices", m.Devices)
	g.GET("/scatter", m.Scatter)
}

// Health serves GET /api/me/health (plugin reachability check).
func (m *MeHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "version": currentAppVersion})
}

// meUser returns the forced user id and rewrites the query so that any
// client-sent user parameter is replaced; 403 when no user is set.
func meUser(c *gin.Context) (string, bool) {
	uid := middleware.MeUserID(c)
	if uid == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "unknown user"})
		return "", false
	}
	q := c.Request.URL.Query()
	for k := range q {
		switch strings.ToLower(k) {
		case "userid", "users", "user", "id":
			q.Del(k)
		}
	}
	q.Set("userId", uid)
	c.Request.URL.RawQuery = q.Encode()
	return uid, true
}

// params parses the v2 contract with the user forced.
func (m *MeHandler) params(c *gin.Context) (V2Params, bool) {
	uid, ok := meUser(c)
	if !ok {
		return V2Params{}, false
	}
	p, ok := m.v2.params(c)
	p.UserID = uid
	return p, ok && uid != ""
}

// meTotals is Totals without the server-wide activeUsers.
type meTotals struct {
	WatchSeconds int64    `json:"watchSeconds"`
	Views        int      `json:"views"`
	Clicks       int      `json:"clicks"`
	Completion   *float64 `json:"completion"`
}

func toMeTotals(t Totals) meTotals {
	return meTotals{WatchSeconds: t.WatchSeconds, Views: t.Views, Clicks: t.Clicks, Completion: t.Completion}
}

// ---------------------------------------------------------------------------
// GET /api/me/profile — the v2 user page (/api/v2/users/:id) of the user.
// ---------------------------------------------------------------------------

type meProfile struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	PrimaryImageTag *string    `json:"primaryImageTag"`
	MemberSince     *time.Time `json:"memberSince"`
	Totals          meTotals   `json:"totals"`
	Previous        *meTotals  `json:"previous,omitempty"`
	FavoriteDevice  *string    `json:"favoriteDevice"`
	Calendar        []struct {
		Date         string `json:"date"`
		WatchSeconds int64  `json:"watchSeconds"`
	} `json:"calendar"`
	InProgress []struct {
		SeriesID    string    `json:"seriesId"`
		Name        string    `json:"name"`
		NextSeason  *int      `json:"nextSeason"`
		NextEpisode *int      `json:"nextEpisode"`
		NextID      string    `json:"nextId"`
		LastPlayed  time.Time `json:"lastPlayed"`
		Progress    float64   `json:"progress"`
	} `json:"inProgress"`
	Hours   []int `json:"hours"`
	Profile struct {
		Regularity    *float64 `json:"regularity"`
		Discovery     *float64 `json:"discovery"`
		Concentration *float64 `json:"concentration"`
		Completion    *float64 `json:"completion"`
	} `json:"profile"`
	PlayedToEnd playedToEnd            `json:"playedToEnd"`
	PluginSince *string                `json:"pluginSince"`
	Genres      ByType[[]namedSeconds] `json:"genres"`
	Streak      struct {
		Current int `json:"current"`
		Best    int `json:"best"`
	} `json:"streak"`
}

func (m *MeHandler) Profile(c *gin.Context) {
	uid, ok := meUser(c)
	if !ok {
		return
	}
	var out meProfile
	saved := c.Params
	c.Params = gin.Params{{Key: "id", Value: uid}}
	ok = relay(c, m.v2.User, &out)
	c.Params = saved
	if !ok {
		return
	}
	if out.ID != uid { // defensive: never answer for someone else
		c.JSON(http.StatusForbidden, gin.H{"error": "unknown user"})
		return
	}
	var u struct{ PrimaryImageTag *string }
	if v2Fail(c, m.v2.raw(c, &u, `SELECT "PrimaryImageTag" AS primary_image_tag FROM jf_users WHERE "Id" = ?`, uid)) {
		return
	}
	out.PrimaryImageTag = u.PrimaryImageTag
	c.JSON(http.StatusOK, out)
}

// relay runs a v2 handler into a buffer and decodes its 200 body into out
// (a whitelist struct); other statuses are passed through. Returns true when
// out was filled (the caller writes the response).
func relay(c *gin.Context, h gin.HandlerFunc, out any) bool {
	orig := c.Writer
	buf := &bufferedWriter{ResponseWriter: orig, status: http.StatusOK}
	c.Writer = buf
	h(c)
	c.Writer = orig
	if buf.status != http.StatusOK {
		c.Data(buf.status, "application/json; charset=utf-8", buf.body.Bytes())
		return false
	}
	if err := json.Unmarshal(buf.body.Bytes(), out); err != nil {
		v2Fail(c, err)
		return false
	}
	return true
}

// bufferedWriter captures a handler's status and body.
type bufferedWriter struct {
	gin.ResponseWriter
	status  int
	written bool
	body    bytes.Buffer
}

func (w *bufferedWriter) WriteHeader(code int) { w.status = code }
func (w *bufferedWriter) WriteHeaderNow()      { w.written = true }
func (w *bufferedWriter) Write(b []byte) (int, error) {
	w.written = true
	return w.body.Write(b)
}
func (w *bufferedWriter) WriteString(s string) (int, error) {
	w.written = true
	return w.body.WriteString(s)
}
func (w *bufferedWriter) Status() int   { return w.status }
func (w *bufferedWriter) Size() int     { return w.body.Len() }
func (w *bufferedWriter) Written() bool { return w.written }

// ---------------------------------------------------------------------------
// GET /api/me/overview — KPIs, daily series, previous period (compare=1).
// ---------------------------------------------------------------------------

type meOverview struct {
	Bucket        string       `json:"bucket"`
	Totals        meTotals     `json:"totals"`
	Previous      *meTotals    `json:"previous,omitempty"`
	Daily         []dailyPoint `json:"daily"`
	PreviousDaily []dailyPoint `json:"previousDaily,omitempty"`
	ByMethod      methodSplit  `json:"byMethod"`
	PlayedToEnd   playedToEnd  `json:"playedToEnd"`
	PluginSince   *string      `json:"pluginSince"`
}

func (m *MeHandler) Overview(c *gin.Context) {
	p, ok := m.params(c)
	if !ok {
		return
	}
	h, f := m.v2, p.filter(p.Range)
	var out meOverview
	t, err := h.totals(c, f)
	if v2Fail(c, err) {
		return
	}
	out.Totals = toMeTotals(t)
	week := useWeeks(p.Range)
	out.Bucket = "day"
	if week {
		out.Bucket = "week"
	}
	if out.Daily, err = h.daily(c, f, p.Range, week); v2Fail(c, err) {
		return
	}
	if p.Prev != nil {
		pf := p.filter(*p.Prev)
		prev, err := h.totals(c, pf)
		if v2Fail(c, err) {
			return
		}
		pt := toMeTotals(prev)
		out.Previous = &pt
		if out.PreviousDaily, err = h.daily(c, pf, *p.Prev, week); v2Fail(c, err) {
			return
		}
	}
	if out.ByMethod, err = h.byMethod(c, f); v2Fail(c, err) {
		return
	}
	if out.PlayedToEnd, err = h.playedToEnd(c, f); v2Fail(c, err) {
		return
	}
	if out.PluginSince, err = h.pluginSince(c, "plays"); v2Fail(c, err) {
		return
	}
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/me/top — top content (10 per kind: movie / series / album).
// ---------------------------------------------------------------------------

func (m *MeHandler) Top(c *gin.Context) {
	p, ok := m.params(c)
	if !ok {
		return
	}
	items, err := m.v2.topItems(c, p.filter(p.Range), 10)
	if v2Fail(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// ---------------------------------------------------------------------------
// GET /api/me/history — paginated history (v2 filters, no user field).
// ---------------------------------------------------------------------------

type meHistoryRow struct {
	ID            string      `json:"id"`
	StartedAt     time.Time   `json:"startedAt"`
	EndedAt       time.Time   `json:"endedAt"`
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

func (m *MeHandler) History(c *gin.Context) {
	p, ok := m.params(c)
	if !ok {
		return
	}
	h := m.v2
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
	out := make([]meHistoryRow, len(rows))
	for i, r := range rows {
		out[i] = meHistoryRow{ID: r.ID, StartedAt: r.StartedAt, EndedAt: r.EndedAt, Item: r.Item, MediaType: r.MediaType,
			WatchSeconds: r.WatchSeconds, Completion: r.Completion, Client: r.Client, Device: r.Device, Method: r.Method,
			IsView: r.IsView, PausedSeconds: r.PausedSeconds}
	}
	c.JSON(http.StatusOK, gin.H{"rows": out, "total": total, "page": p.Page, "pageSize": p.PageSize})
}

// ---------------------------------------------------------------------------
// GET /api/me/types — split by media family.
// ---------------------------------------------------------------------------

type meTypes struct {
	ByType      ByType[TypeMetrics] `json:"byType"`
	PlayedToEnd ByType[playedToEnd] `json:"playedToEnd"`
}

func (m *MeHandler) Types(c *gin.Context) {
	p, ok := m.params(c)
	if !ok {
		return
	}
	h, f := m.v2, p.filter(p.Range)
	var r struct {
		MovieSeconds, SeriesSeconds, MusicSeconds int64
		MovieViews, SeriesViews, MusicViews       int
		MovieClicks, SeriesClicks, MusicClicks    int
	}
	q, args := f.sql(`
		SELECT COALESCE(SUM(duration) FILTER (WHERE media_type = 'Movie'), 0)::bigint AS movie_seconds,
		  COALESCE(SUM(duration) FILTER (WHERE media_type = 'Episode'), 0)::bigint AS series_seconds,
		  COALESCE(SUM(duration) FILTER (WHERE media_type = 'Audio'), 0)::bigint AS music_seconds,
		  COUNT(*) FILTER (WHERE is_view AND media_type = 'Movie')::int AS movie_views,
		  COUNT(*) FILTER (WHERE is_view AND media_type = 'Episode')::int AS series_views,
		  COUNT(*) FILTER (WHERE is_view AND media_type = 'Audio')::int AS music_views,
		  COUNT(*) FILTER (WHERE media_type = 'Movie')::int AS movie_clicks,
		  COUNT(*) FILTER (WHERE media_type = 'Episode')::int AS series_clicks,
		  COUNT(*) FILTER (WHERE media_type = 'Audio')::int AS music_clicks
		FROM acts`)
	if v2Fail(c, h.raw(c, &r, q, args...)) {
		return
	}
	out := meTypes{ByType: ByType[TypeMetrics]{
		Movie:  TypeMetrics{r.MovieSeconds, r.MovieViews, r.MovieClicks},
		Series: TypeMetrics{r.SeriesSeconds, r.SeriesViews, r.SeriesClicks},
		Music:  TypeMetrics{r.MusicSeconds, r.MusicViews, r.MusicClicks},
	}}
	for _, t := range []struct {
		mt  string
		dst *playedToEnd
	}{{"Movie", &out.PlayedToEnd.Movie}, {"Episode", &out.PlayedToEnd.Series}, {"Audio", &out.PlayedToEnd.Music}} {
		tp := p
		tp.MediaType = t.mt
		v, err := h.playedToEnd(c, tp.filter(p.Range))
		if v2Fail(c, err) {
			return
		}
		*t.dst = v
	}
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/me/habits — views per weekday × hour (Monday first).
// ---------------------------------------------------------------------------

func (m *MeHandler) Habits(c *gin.Context) {
	p, ok := m.params(c)
	if !ok {
		return
	}
	g, err := m.v2.weekHour(c, p.filter(p.Range))
	if v2Fail(c, err) {
		return
	}
	hours, days := make([]int, 24), make([]int, 7)
	for d, row := range g {
		for hr, v := range row {
			hours[hr] += v
			days[d] += v
		}
	}
	c.JSON(http.StatusOK, gin.H{"weekHour": g, "hours": hours, "weekdays": days})
}

// ---------------------------------------------------------------------------
// GET /api/me/devices — devices and clients (no IP).
// ---------------------------------------------------------------------------

type meDevice struct {
	Device       string     `json:"device"`
	Client       string     `json:"client"`
	Version      string     `json:"version"`
	Views        int        `json:"views"`
	Clicks       int        `json:"clicks"`
	WatchSeconds int64      `json:"watchSeconds"`
	LastUsed     *time.Time `json:"lastUsed"`
}

func (m *MeHandler) Devices(c *gin.Context) {
	p, ok := m.params(c)
	if !ok {
		return
	}
	h, f := m.v2, p.filter(p.Range)
	devices := []meDevice{}
	q, args := f.sql(`SELECT COALESCE(device_name, '') AS device, COALESCE(client, '') AS client, COALESCE(app_version, '') AS version,
		` + aggPlays + ` AS views, ` + aggClicks + ` AS clicks, ` + aggSeconds + ` AS watch_seconds, MAX(started_at) AS last_used
		FROM acts GROUP BY 1, 2, 3 ORDER BY views DESC, clicks DESC, device LIMIT 20`)
	if v2Fail(c, h.raw(c, &devices, q, args...)) {
		return
	}
	for i := range devices {
		devices[i].LastUsed = localTimePtr(devices[i].LastUsed)
	}
	clients, err := h.clients(c, f)
	if v2Fail(c, err) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"devices": devices, "clients": clients})
}

// ---------------------------------------------------------------------------
// GET /api/me/scatter — one point per play (most recent playsLimit).
// ---------------------------------------------------------------------------

type meScatterPoint struct {
	ID           string      `json:"id"`
	StartedAt    time.Time   `json:"startedAt"`
	WatchSeconds int64       `json:"watchSeconds"`
	Type         string      `json:"type"`
	Item         historyItem `json:"item"`
}

func (m *MeHandler) Scatter(c *gin.Context) {
	p, ok := m.params(c)
	if !ok {
		return
	}
	res, err := m.v2.plays(c, p.filter(p.Range))
	if v2Fail(c, err) {
		return
	}
	points := make([]meScatterPoint, len(res.Points))
	for i, pt := range res.Points {
		points[i] = meScatterPoint{ID: pt.ID, StartedAt: pt.StartedAt, WatchSeconds: pt.WatchSeconds, Type: pt.Type, Item: pt.Item}
	}
	c.JSON(http.StatusOK, gin.H{"total": res.Total, "limit": res.Limit, "capped": res.Capped, "points": points})
}
