package handler

import (
	"net/http"
	"time"

	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
)

const trendDays = 14

// trendRange is the last 14 local days of tr (or tr itself when shorter).
func trendRange(tr TimeRange) TimeRange {
	from := tr.To.In(statsql.Location()).AddDate(0, 0, -trendDays)
	if from.Before(tr.From) {
		from = tr.From
	}
	return TimeRange{From: from, To: tr.To}
}

// trendSeries builds a 14-slot series (oldest first) ending at tr.To from
// date → value; slots before tr.From stay 0.
func trendSeries(tr TimeRange, byDate map[string]int64) []int64 {
	out := make([]int64, trendDays)
	end := tr.To.In(statsql.Location())
	for i := 0; i < trendDays; i++ {
		d := end.AddDate(0, 0, -(trendDays - i))
		out[i] = byDate[d.Format(v2DateLayout)]
	}
	return out
}

type userListRow struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	WatchSeconds int64         `json:"watchSeconds"`
	Views        int           `json:"views"`
	ByType       ByType[int64] `json:"byType"`
	LastActivity *time.Time    `json:"lastActivity"`
	Trend        []int64       `json:"trend"`
}

// GET /api/v2/users — every known user with their stats in range; byType is
// watch seconds per family, trend = watch seconds per day over the last 14
// days of the range, lastActivity = latest playback ever.
func (h *V2Handler) Users(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	f := p.filter(p.Range)
	var rows []struct {
		ID, Name             string
		WatchSeconds         int64
		Views                int
		Movie, Series, Music int64
		LastActivity         *time.Time
	}
	q, args := f.sql(`,
		st AS (
		  SELECT user_id, ` + aggSeconds + ` AS watch_seconds, ` + aggPlays + ` AS views,
		         COALESCE(SUM(duration) FILTER (WHERE media_type = 'Movie'), 0)::bigint AS movie,
		         COALESCE(SUM(duration) FILTER (WHERE media_type = 'Episode'), 0)::bigint AS series,
		         COALESCE(SUM(duration) FILTER (WHERE media_type = 'Audio'), 0)::bigint AS music
		  FROM acts GROUP BY user_id
		),
		known AS (
		  SELECT "Id" AS id, "Name" AS name FROM jf_users
		  UNION
		  SELECT user_id, MAX(user_name) FROM acts WHERE user_id IS NOT NULL AND user_id NOT IN (SELECT "Id" FROM jf_users) GROUP BY user_id
		)
		SELECT k.id, COALESCE(k.name, k.id) AS name, COALESCE(st.watch_seconds, 0) AS watch_seconds, COALESCE(st.views, 0) AS views,
		       COALESCE(st.movie, 0) AS movie, COALESCE(st.series, 0) AS series, COALESCE(st.music, 0) AS music, l.last_activity
		FROM known k LEFT JOIN st ON st.user_id = k.id
		LEFT JOIN LATERAL (
		  SELECT p."StartedAt" AS last_activity FROM jf_playback_activity p
		  WHERE p."UserId" = k.id AND p."StartedAt" IS NOT NULL AND ` + statsql.MediaFilter("p") + `
		  ORDER BY p."StartedAt" DESC LIMIT 1
		) l ON true
		ORDER BY watch_seconds DESC, views DESC, name`)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	tr := trendRange(p.Range)
	var trend []struct {
		UserID, Date string
		WatchSeconds int64
	}
	q, args = p.filter(tr).sql(`SELECT user_id, ` + localDay + ` AS date, ` + aggSeconds + ` AS watch_seconds FROM acts WHERE user_id IS NOT NULL GROUP BY 1, 2`)
	if v2Fail(c, h.raw(c, &trend, q, args...)) {
		return
	}
	byUser := map[string]map[string]int64{}
	for _, t := range trend {
		if byUser[t.UserID] == nil {
			byUser[t.UserID] = map[string]int64{}
		}
		byUser[t.UserID][t.Date] = t.WatchSeconds
	}
	out := make([]userListRow, len(rows))
	for i, r := range rows {
		out[i] = userListRow{
			ID: r.ID, Name: r.Name, WatchSeconds: r.WatchSeconds, Views: r.Views,
			ByType:       ByType[int64]{r.Movie, r.Series, r.Music},
			LastActivity: localTimePtr(r.LastActivity), Trend: trendSeries(p.Range, byUser[r.ID]),
		}
	}
	c.JSON(http.StatusOK, out)
}

type namedSeconds struct {
	Name         string `json:"name"`
	WatchSeconds int64  `json:"watchSeconds"`
}

// GET /api/v2/users/:id
func (h *V2Handler) User(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var user struct {
		ID          string
		Name        string
		MemberSince *time.Time
	}
	if v2Fail(c, h.raw(c, &user, `
		WITH u AS (
		  SELECT "Id" AS id, "Name" AS name FROM jf_users WHERE "Id" = ?
		  UNION ALL
		  SELECT "UserId", MAX("UserName") FROM jf_playback_activity
		  WHERE "UserId" = ? AND NOT EXISTS (SELECT 1 FROM jf_users WHERE "Id" = ?) GROUP BY "UserId"
		)
		SELECT u.id, COALESCE(u.name, u.id) AS name,
		       (SELECT MIN(`+statsql.StartedAt("p")+`) FROM jf_playback_activity p WHERE p."UserId" = u.id AND `+statsql.MediaFilter("p")+`) AS member_since
		FROM u LIMIT 1`, id, id, id)) {
		return
	}
	if user.ID == "" {
		v2NotFound(c)
		return
	}
	up := p
	up.UserID = id
	f := up.filter(p.Range)
	out := gin.H{"id": user.ID, "name": user.Name, "memberSince": localTimePtr(user.MemberSince)}

	totals, err := h.totals(c, f)
	if v2Fail(c, err) {
		return
	}
	out["totals"] = totals
	if p.Prev != nil {
		prev, err := h.totals(c, up.filter(*p.Prev))
		if v2Fail(c, err) {
			return
		}
		out["previous"] = prev
	}

	var fav struct{ Device *string }
	q, args := f.sql(`SELECT device_name AS device FROM acts WHERE is_view AND device_name IS NOT NULL AND device_name <> ''
		GROUP BY device_name ORDER BY COUNT(*) DESC, device_name LIMIT 1`)
	if v2Fail(c, h.raw(c, &fav, q, args...)) {
		return
	}
	out["favoriteDevice"] = fav.Device

	// Calendar: last 365 local days ending today, independent of the range.
	loc := statsql.Location()
	calRange := TimeRange{To: statsql.StartOfDay(time.Now(), loc).AddDate(0, 0, 1)}
	calRange.From = calRange.To.AddDate(0, 0, -365)
	var cal []struct {
		Date         string
		WatchSeconds int64
	}
	q, args = acts().inRange(calRange).forUser(id).sql(`SELECT ` + localDay + ` AS date, ` + aggSeconds + ` AS watch_seconds FROM acts GROUP BY 1`)
	if v2Fail(c, h.raw(c, &cal, q, args...)) {
		return
	}
	calMap := map[string]int64{}
	for _, d := range cal {
		calMap[d.Date] = d.WatchSeconds
	}
	type calDay struct {
		Date         string `json:"date"`
		WatchSeconds int64  `json:"watchSeconds"`
	}
	calendar := []calDay{}
	eachDay(calRange, func(d string) { calendar = append(calendar, calDay{d, calMap[d]}) })
	out["calendar"] = calendar

	// In progress (all time): series whose last viewed episode has a next one.
	type inProgress struct {
		SeriesID    string    `json:"seriesId"`
		Name        string    `json:"name"`
		NextSeason  *int      `json:"nextSeason"`
		NextEpisode *int      `json:"nextEpisode"`
		NextID      string    `json:"nextId"`
		LastPlayed  time.Time `json:"lastPlayed"`
		Progress    float64   `json:"progress"`
	}
	inProg := []inProgress{}
	q, args = acts().forUser(id).where(`a."MediaType" = 'Episode' AND a."IsView"`).where(notSpecialPlay).sql(`,
		ord AS (
		  SELECT e."Id", e."SeriesId", e."ParentIndexNumber", e."IndexNumber",
		         LEAD(e."Id") OVER w AS next_id, LEAD(e."ParentIndexNumber") OVER w AS next_season,
		         LEAD(e."IndexNumber") OVER w AS next_episode, COUNT(*) OVER (PARTITION BY e."SeriesId") AS total
		  FROM jf_library_episodes e WHERE NOT e.archived AND ` + notSpecial + `
		  WINDOW w AS (PARTITION BY e."SeriesId" ORDER BY ` + episodeOrder + `)
		),
		last AS (
		  SELECT DISTINCT ON (group_id) group_id, group_name, item_id, started_at
		  FROM acts ORDER BY group_id, started_at DESC
		),
		seen AS (SELECT group_id, COUNT(DISTINCT item_id) AS n FROM acts GROUP BY group_id)
		SELECT l.group_id AS series_id, COALESCE(l.group_name, '') AS name, o.next_season, o.next_episode, o.next_id,
		       l.started_at AS last_played, LEAST(s.n::float8 / NULLIF(o.total, 0), 1.0) AS progress
		FROM last l JOIN ord o ON o."Id" = l.item_id JOIN seen s ON s.group_id = l.group_id
		WHERE o.next_id IS NOT NULL
		ORDER BY l.started_at DESC LIMIT 10`)
	if v2Fail(c, h.raw(c, &inProg, q, args...)) {
		return
	}
	for i := range inProg {
		inProg[i].LastPlayed = localTime(inProg[i].LastPlayed)
	}
	out["inProgress"] = inProg

	var hrs []struct{ Hour, Views int }
	q, args = f.sql(`SELECT EXTRACT(HOUR FROM local_ts)::int AS hour, COUNT(*)::int AS views FROM acts WHERE is_view GROUP BY 1`)
	if v2Fail(c, h.raw(c, &hrs, q, args...)) {
		return
	}
	hours := make([]int, 24)
	for _, r := range hrs {
		if r.Hour >= 0 && r.Hour < 24 {
			hours[r.Hour] = r.Views
		}
	}
	out["hours"] = hours

	// Profile ratios (0..1, null without views):
	//   regularity    active local days / days in range
	//   discovery     views of titles the user had never viewed before the range
	//   concentration share of views on the user's top 3 titles
	//   completion    completed (user, item, day) units / units
	var prof struct {
		ActiveDays int
		Views      int
		NewViews   int
		Top3       int
		Completion *float64
	}
	q, args = f.sql(completionCTE+`,
		per AS (SELECT group_id, COUNT(*) AS n FROM acts WHERE is_view GROUP BY group_id),
		prior AS (
		  SELECT DISTINCT group_id FROM (`+actsSelect()+`
		    AND a."UserId" = ? AND a."IsView" AND a."StartedAt" < ?) x
		  WHERE group_id IS NOT NULL
		)
		SELECT COUNT(DISTINCT local_ts::date) FILTER (WHERE is_view)::int AS active_days,
		       `+aggPlays+` AS views,
		       COUNT(*) FILTER (WHERE is_view AND group_id IS NOT NULL AND group_id NOT IN (SELECT group_id FROM prior))::int AS new_views,
		       (SELECT COALESCE(SUM(n), 0) FROM (SELECT n FROM per ORDER BY n DESC LIMIT 3) t)::int AS top3,
		       (SELECT AVG(CASE WHEN `+completedCond+` THEN 1.0 ELSE 0.0 END)::float8 FROM completion) AS completion
		FROM acts`, id, p.Range.From)
	if v2Fail(c, h.raw(c, &prof, q, args...)) {
		return
	}
	ratio := func(n, d int) *float64 {
		if d == 0 {
			return nil
		}
		v := float64(n) / float64(d)
		return &v
	}
	days := calendarDays(p.Range.From, p.Range.To)
	profile := gin.H{"regularity": ratio(prof.ActiveDays, days), "discovery": ratio(prof.NewViews, prof.Views),
		"concentration": ratio(prof.Top3, prof.Views), "completion": prof.Completion}
	if prof.Views == 0 {
		profile["regularity"] = nil
	}
	out["profile"] = profile

	var gen []struct {
		MediaType, Name string
		WatchSeconds    int64
	}
	q, args = f.sql(`,
		g AS (
		  SELECT media_type, genre AS name, SUM(duration)::bigint AS watch_seconds,
		         ROW_NUMBER() OVER (PARTITION BY media_type ORDER BY SUM(duration) DESC, genre) AS rn
		  FROM acts ` + genreLateral + `
		  WHERE acts.genres IS NOT NULL GROUP BY media_type, genre
		)
		SELECT media_type, name, watch_seconds FROM g WHERE rn <= 10 ORDER BY media_type, rn`)
	if v2Fail(c, h.raw(c, &gen, q, args...)) {
		return
	}
	genres := ByType[[]namedSeconds]{Movie: []namedSeconds{}, Series: []namedSeconds{}, Music: []namedSeconds{}}
	for _, g := range gen {
		ns := namedSeconds{g.Name, g.WatchSeconds}
		switch g.MediaType {
		case "Movie":
			genres.Movie = append(genres.Movie, ns)
		case "Episode":
			genres.Series = append(genres.Series, ns)
		case "Audio":
			genres.Music = append(genres.Music, ns)
		}
	}
	out["genres"] = genres

	type device struct {
		Device  string `json:"device"`
		Client  string `json:"client"`
		Version string `json:"version"`
		Views   int    `json:"views"`
	}
	devices := []device{}
	q, args = f.sql(`SELECT COALESCE(device_name, '') AS device, COALESCE(client, '') AS client, COALESCE(app_version, '') AS version,
		COUNT(*)::int AS views FROM acts WHERE is_view GROUP BY 1, 2, 3 ORDER BY views DESC, device LIMIT 20`)
	if v2Fail(c, h.raw(c, &devices, q, args...)) {
		return
	}
	out["devices"] = devices

	var dayRows []struct{ Day time.Time }
	q, args = acts().forUser(id).where(`a."IsView"`).sql(`SELECT DISTINCT local_ts::date AS day FROM acts ORDER BY 1`)
	if v2Fail(c, h.raw(c, &dayRows, q, args...)) {
		return
	}
	dates := make([]string, len(dayRows))
	for i, d := range dayRows {
		dates[i] = d.Day.Format(v2DateLayout)
	}
	cur, best := streaks(dates, time.Now().In(loc).Format(v2DateLayout))
	out["streak"] = gin.H{"current": cur, "best": best}

	c.JSON(http.StatusOK, out)
}

// streaks returns the current (ending today or yesterday) and best runs of
// consecutive days in sorted, distinct YYYY-MM-DD dates.
func streaks(dates []string, today string) (current, best int) {
	parse := func(s string) time.Time { t, _ := time.Parse(v2DateLayout, s); return t }
	run := 0
	var prev time.Time
	for i, s := range dates {
		d := parse(s)
		if i > 0 && d.Sub(prev) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		if run > best {
			best = run
		}
		prev = d
	}
	if len(dates) > 0 {
		gap := parse(today).Sub(prev)
		if gap == 0 || gap == 24*time.Hour {
			current = run
		}
	}
	return current, best
}
