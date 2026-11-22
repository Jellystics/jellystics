package handler

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
)

// Phase 3 endpoints: weekly user retention, each user's share over time and
// the concurrency timeline of one day.

const (
	retentionOffsets  = 5 // S0 … S+4
	retentionMaxWeeks = 8 // most recent cohort weeks returned
	concurrencySlot   = 15 * time.Minute
	shareWeekMaxDays  = 120 // up to this many days the share is bucketed by week, else by month
)

// registerUsersExtra mounts the phase 3 routes on the /api/v2 group.
func (h *V2Handler) registerUsersExtra(v2 *gin.RouterGroup) {
	v2.GET("/users/retention", h.UsersRetention)
	v2.GET("/users/share", h.UsersShare)
	v2.GET("/concurrency", h.Concurrency)
}

// mondayOf returns the local midnight of the Monday of t's local week.
func mondayOf(t time.Time) time.Time {
	loc := statsql.Location()
	d := statsql.StartOfDay(t, loc)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// firstOfMonth returns the local midnight of the first day of t's local month.
func firstOfMonth(t time.Time) time.Time {
	l := t.In(statsql.Location())
	return time.Date(l.Year(), l.Month(), 1, 0, 0, 0, 0, statsql.Location())
}

// lastDay is the local midnight of the last day of tr ([From, To), To exclusive).
func lastDay(tr TimeRange) time.Time {
	return statsql.StartOfDay(tr.To, statsql.Location()).AddDate(0, 0, -1)
}

// ---------------------------------------------------------------------------
// GET /api/v2/users/retention
// ---------------------------------------------------------------------------

type retentionRow struct {
	Week     string `json:"week"`
	Cohort   int    `json:"cohort"`
	Retained []*int `json:"retained"`
}

type retentionResponse struct {
	Offsets int            `json:"offsets"`
	Rows    []retentionRow `json:"rows"`
}

// UsersRetention: one row per local week (Monday) of the range (the last 8):
// cohort = users active that week; retained[k] = how many of them are active
// again k weeks later (retained[0] = cohort). Offsets that fall after the
// range are null.
func (h *V2Handler) UsersRetention(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	var rows []struct {
		Week string
		K, N int
	}
	// Whole weeks: a partial first or last week would undercount its cohort and returns.
	lastWeek := mondayOf(lastDay(p.Range))
	whole := TimeRange{From: mondayOf(p.Range.From), To: lastWeek.AddDate(0, 0, 7)}
	q, args := p.filter(whole).sql(`,
		uw AS (SELECT DISTINCT user_id, date_trunc('week', local_ts)::date AS wk FROM acts WHERE user_id IS NOT NULL)
		SELECT to_char(c.wk, 'YYYY-MM-DD') AS week, ((o.wk - c.wk) / 7)::int AS k, COUNT(*)::int AS n
		FROM uw c JOIN uw o ON o.user_id = c.user_id AND o.wk >= c.wk AND o.wk <= c.wk + ?::int
		GROUP BY 1, 2`, 7*(retentionOffsets-1))
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	counts := map[string]map[int]int{}
	for _, r := range rows {
		if counts[r.Week] == nil {
			counts[r.Week] = map[int]int{}
		}
		counts[r.Week][r.K] = r.N
	}
	last := lastWeek
	weeks := []time.Time{}
	for w := mondayOf(p.Range.From); !w.After(last); w = w.AddDate(0, 0, 7) {
		weeks = append(weeks, w)
	}
	if len(weeks) > retentionMaxWeeks {
		weeks = weeks[len(weeks)-retentionMaxWeeks:]
	}
	out := retentionResponse{Offsets: retentionOffsets, Rows: []retentionRow{}}
	for _, w := range weeks {
		key := w.Format(v2DateLayout)
		row := retentionRow{Week: key, Cohort: counts[key][0], Retained: make([]*int, retentionOffsets)}
		for k := 0; k < retentionOffsets; k++ {
			if w.AddDate(0, 0, 7*k).After(last) {
				continue
			}
			n := counts[key][k]
			row.Retained[k] = &n
		}
		out.Rows = append(out.Rows, row)
	}
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/v2/users/share
// ---------------------------------------------------------------------------

type shareUser struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	WatchSeconds int64     `json:"watchSeconds"`
	Share        float64   `json:"share"`
	Values       []float64 `json:"values"`
}

type shareResponse struct {
	Bucket  string      `json:"bucket"`
	Periods []string    `json:"periods"`
	Users   []shareUser `json:"users"`
}

// UsersShare: each user's share (0..1) of the watch time, overall and per
// bucket (week up to 120 days, month beyond). Users without watch time in the
// range are omitted; sorted by share.
func (h *V2Handler) UsersShare(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	bucket, start := "week", mondayOf
	if calendarDays(p.Range.From, p.Range.To) > shareWeekMaxDays {
		bucket, start = "month", firstOfMonth
	}
	periods := []string{}
	index := map[string]int{}
	last := lastDay(p.Range)
	for d := start(p.Range.From); !d.After(last); {
		index[d.Format(v2DateLayout)] = len(periods)
		periods = append(periods, d.Format(v2DateLayout))
		if bucket == "week" {
			d = d.AddDate(0, 0, 7)
		} else {
			d = d.AddDate(0, 1, 0)
		}
	}

	var rows []struct {
		UserID, Name, Period string
		Seconds              int64
	}
	q, args := p.filter(p.Range).sql(`
		SELECT user_id, MAX(user_name) AS name, to_char(date_trunc(?::text, local_ts)::date, 'YYYY-MM-DD') AS period,
		       `+aggSeconds+` AS seconds
		FROM acts WHERE user_id IS NOT NULL GROUP BY user_id, 3`, bucket)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	totals := make([]int64, len(periods))
	var grand int64
	users := map[string]*shareUser{}
	seconds := map[string][]int64{}
	for _, r := range rows {
		i, ok := index[r.Period]
		if !ok {
			continue
		}
		u := users[r.UserID]
		if u == nil {
			u = &shareUser{ID: r.UserID, Name: r.Name}
			users[r.UserID] = u
			seconds[r.UserID] = make([]int64, len(periods))
		}
		u.WatchSeconds += r.Seconds
		seconds[r.UserID][i] += r.Seconds
		totals[i] += r.Seconds
		grand += r.Seconds
	}
	// Drop the leading periods without any watch time (an "all" range starts years back).
	first := 0
	for first < len(totals) && totals[first] == 0 {
		first++
	}
	if first == len(totals) {
		first = 0
	}
	out := shareResponse{Bucket: bucket, Periods: periods[first:], Users: []shareUser{}}
	for id, u := range users {
		if u.WatchSeconds == 0 {
			continue
		}
		u.Values = make([]float64, len(periods)-first)
		for i := first; i < len(periods); i++ {
			if totals[i] > 0 {
				u.Values[i-first] = float64(seconds[id][i]) / float64(totals[i])
			}
		}
		if grand > 0 {
			u.Share = float64(u.WatchSeconds) / float64(grand)
		}
		out.Users = append(out.Users, *u)
	}
	sort.Slice(out.Users, func(i, j int) bool {
		a, b := out.Users[i], out.Users[j]
		if a.WatchSeconds != b.WatchSeconds {
			return a.WatchSeconds > b.WatchSeconds
		}
		return a.Name < b.Name
	})
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// GET /api/v2/concurrency?date=YYYY-MM-DD
// ---------------------------------------------------------------------------

type concurrencySession struct {
	User       refUser   `json:"user"`
	ItemID     string    `json:"itemId"`
	Name       string    `json:"name"`
	ParentName string    `json:"parentName,omitempty"`
	Kind       string    `json:"kind"` // movie | series | music
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
}

type concurrencyResponse struct {
	Date        string               `json:"date"`
	Peak        int                  `json:"peak"`
	PeakAt      *time.Time           `json:"peakAt"`
	SlotMinutes int                  `json:"slotMinutes"`
	Slots       []int                `json:"slots"`
	Sessions    []concurrencySession `json:"sessions"`
}

type concEvent struct {
	ts time.Time
	d  int
}

// concurrencySlots sweeps the sessions (already clipped to [from, to)) and
// returns the max number of simultaneous sessions per slot, the peak and the
// first time it was reached.
func concurrencySlots(sessions []concurrencySession, from, to time.Time, slot time.Duration) ([]int, int, *time.Time) {
	ev := make([]concEvent, 0, 2*len(sessions))
	for _, s := range sessions {
		ev = append(ev, concEvent{s.Start, 1}, concEvent{s.End, -1})
	}
	sort.Slice(ev, func(i, j int) bool {
		if !ev[i].ts.Equal(ev[j].ts) {
			return ev[i].ts.Before(ev[j].ts)
		}
		return ev[i].d < ev[j].d // an end before a start at the same instant
	})
	n := int((to.Sub(from) + slot - 1) / slot)
	slots := make([]int, n)
	running, peak, j := 0, 0, 0
	var peakAt *time.Time
	for i := 0; i < n; i++ {
		end := from.Add(time.Duration(i+1) * slot)
		max := running
		for ; j < len(ev) && ev[j].ts.Before(end); j++ {
			running += ev[j].d
			if running > max {
				max = running
			}
			if running > peak {
				peak = running
				t := localTime(ev[j].ts)
				peakAt = &t
			}
		}
		slots[i] = max
	}
	return slots, peak, peakAt
}

// Concurrency: the playback sessions overlapping one local day (default today;
// user / type / library filters apply, the range does not), clipped to the
// day, with the simultaneous-session count per 15-minute slot.
func (h *V2Handler) Concurrency(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	loc := statsql.Location()
	day := statsql.StartOfDay(time.Now(), loc)
	if s := strings.TrimSpace(c.Query("date")); s != "" {
		d, ok := parseLocalDate(s, loc)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
			return
		}
		day = d
	}
	next := day.AddDate(0, 0, 1)
	var rows []struct {
		UserID, UserName, ItemID, ItemName, GroupName, MediaType string
		StartedAt, EndedAt                                       time.Time
	}
	q, args := p.filter(TimeRange{From: day.AddDate(0, 0, -1), To: next}).sql(`
		SELECT COALESCE(user_id, '') AS user_id, COALESCE(user_name, '') AS user_name, COALESCE(item_id, '') AS item_id,
		       COALESCE(item_name, '') AS item_name, COALESCE(group_name, '') AS group_name, media_type, started_at, ended_at
		FROM acts WHERE ended_at > ? ORDER BY started_at, id`, day)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	out := concurrencyResponse{Date: day.Format(v2DateLayout), SlotMinutes: int(concurrencySlot / time.Minute), Sessions: []concurrencySession{}}
	for _, r := range rows {
		start, end := r.StartedAt, r.EndedAt
		if start.Before(day) {
			start = day
		}
		if end.After(next) {
			end = next
		}
		if !end.After(start) {
			continue
		}
		s := concurrencySession{
			User: refUser{ID: r.UserID, Name: r.UserName}, ItemID: r.ItemID, Name: r.ItemName,
			Start: localTime(start), End: localTime(end),
		}
		switch r.MediaType {
		case "Movie":
			s.Kind = "movie"
		case "Episode":
			s.Kind, s.ParentName = "series", r.GroupName
		default:
			s.Kind, s.ParentName = "music", r.GroupName
		}
		out.Sessions = append(out.Sessions, s)
	}
	out.Slots, out.Peak, out.PeakAt = concurrencySlots(out.Sessions, day, next, concurrencySlot)
	c.JSON(http.StatusOK, out)
}
