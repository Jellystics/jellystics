package handler

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
)

var recapPeriodRe = regexp.MustCompile(`^(\d{4})(?:-(\d{2}))?$`)

// recapRange parses a recap period (YYYY or YYYY-MM) into its local range,
// the previous period of the same kind, and the bucket granularity.
func recapRange(period string, loc *time.Location) (cur, prev TimeRange, granularity string, ok bool) {
	m := recapPeriodRe.FindStringSubmatch(period)
	if m == nil {
		return cur, prev, "", false
	}
	year, _ := strconv.Atoi(m[1])
	if year < 1970 || year > 9999 {
		return cur, prev, "", false
	}
	if m[2] == "" {
		from := time.Date(year, 1, 1, 0, 0, 0, 0, loc)
		return TimeRange{From: from, To: from.AddDate(1, 0, 0)}, TimeRange{From: from.AddDate(-1, 0, 0), To: from}, "month", true
	}
	month, _ := strconv.Atoi(m[2])
	if month < 1 || month > 12 {
		return cur, prev, "", false
	}
	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
	return TimeRange{From: from, To: from.AddDate(0, 1, 0)}, TimeRange{From: from.AddDate(0, -1, 0), To: from}, "day", true
}

type recapTotals struct {
	WatchSeconds int64 `json:"watchSeconds"`
	Views        int   `json:"views"`
	ActiveDays   int   `json:"activeDays"`
}

type recapTop struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	WatchSeconds int64  `json:"watchSeconds"`
	Views        int    `json:"views"`
}

type recapBucket struct {
	Key          string `json:"key"` // YYYY-MM (year) or YYYY-MM-DD (month)
	WatchSeconds int64  `json:"watchSeconds"`
	Views        int    `json:"views"`
	NewArtists   int    `json:"newArtists"` // artists listened to for the first time ever
}

type recapRank struct {
	ID           string  `json:"id"`
	Kind         string  `json:"kind"` // movie | series | album
	Name         string  `json:"name"`
	ParentName   *string `json:"parentName"` // album artist
	Views        int     `json:"views"`
	Rank         int     `json:"rank"`
	PreviousRank *int    `json:"previousRank"` // nil = not played in the previous period
}

type recapResponse struct {
	User        RefUserJSON   `json:"user"`
	Period      string        `json:"period"`
	Granularity string        `json:"granularity"` // month | day
	From        string        `json:"from"`
	To          string        `json:"to"` // exclusive
	Totals      recapTotals   `json:"totals"`
	Previous    recapTotals   `json:"previous"`
	TopSeries   []recapTop    `json:"topSeries"`
	TopMovies   []recapTop    `json:"topMovies"`
	TopArtists  []recapTop    `json:"topArtists"`
	Buckets     []recapBucket `json:"buckets"`
	Records     struct {
		BestBucket *recapBucket `json:"bestBucket"`
		Binge      *struct {
			SeriesID   string `json:"seriesId"`
			SeriesName string `json:"seriesName"`
			Episodes   int    `json:"episodes"`
			Date       string `json:"date"`
		} `json:"binge"`
		LongestStreak int `json:"longestStreak"` // consecutive local days with a view
	} `json:"records"`
	Ranking []recapRank `json:"ranking"`
}

// GET /api/v2/recap/:userId/:period (period = YYYY or YYYY-MM, local time).
func (h *V2Handler) Recap(c *gin.Context) {
	loc := statsql.Location()
	userID, period := c.Param("userId"), c.Param("period")
	cur, prev, gran, ok := recapRange(period, loc)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid period"})
		return
	}
	var user struct {
		Name  *string
		Known bool
	}
	if v2Fail(c, h.raw(c, &user, `SELECT
		  COALESCE((SELECT "Name" FROM jf_users WHERE "Id" = ?),
		           (SELECT "UserName" FROM jf_playback_activity WHERE "UserId" = ? AND "UserName" IS NOT NULL ORDER BY "StartedAt" DESC NULLS LAST LIMIT 1)) AS name,
		  (EXISTS (SELECT 1 FROM jf_users WHERE "Id" = ?) OR EXISTS (SELECT 1 FROM jf_playback_activity WHERE "UserId" = ?)) AS known`,
		userID, userID, userID, userID)) {
		return
	}
	if !user.Known {
		v2NotFound(c)
		return
	}
	p := V2Params{UserID: userID}
	f := p.filter(cur)
	out := recapResponse{
		User: RefUserJSON{ID: userID, Name: userID}, Period: period, Granularity: gran,
		From: cur.From.Format(v2DateLayout), To: cur.To.Format(v2DateLayout),
	}
	if user.Name != nil {
		out.User.Name = *user.Name
	}

	totals := func(f *actFilter, dst *recapTotals) error {
		q, args := f.sql(`SELECT ` + aggSeconds + ` AS watch_seconds, ` + aggPlays + ` AS views,
			COUNT(DISTINCT local_ts::date) FILTER (WHERE is_view)::int AS active_days FROM acts`)
		return h.raw(c, dst, q, args...)
	}
	if v2Fail(c, totals(f, &out.Totals)) || v2Fail(c, totals(p.filter(prev), &out.Previous)) {
		return
	}

	top := func(displayType, order string) ([]recapTop, error) {
		rows := []recapTop{}
		q, args := f.sql(`SELECT group_id AS id, COALESCE(MAX(group_name), '') AS name, `+aggSeconds+` AS watch_seconds, `+aggPlays+` AS views
			FROM acts WHERE display_type = ? AND group_id IS NOT NULL
			GROUP BY group_id HAVING `+aggPlays+` > 0 ORDER BY `+order+`, id LIMIT 5`, displayType)
		return rows, h.raw(c, &rows, q, args...)
	}
	var err error
	if out.TopSeries, err = top("Series", "watch_seconds DESC, views DESC"); v2Fail(c, err) {
		return
	}
	if out.TopMovies, err = top("Movie", "views DESC, watch_seconds DESC"); v2Fail(c, err) {
		return
	}

	// Artists: the artist of the track, else the album artist.
	const artistJoin = `
		LEFT JOIN jf_music_tracks t ON t."Id" = acts.item_id
		LEFT JOIN jf_library_items al ON al."Id" = acts.group_id
		LEFT JOIN jf_music_artists ar ON ar."Id" = COALESCE(t."ArtistId", al."ArtistId")`
	const artistID = `COALESCE(t."ArtistId", al."ArtistId")`
	out.TopArtists = []recapTop{}
	q, args := f.sql(`SELECT ` + artistID + ` AS id, COALESCE(MAX(ar."Name"), MAX(al."AlbumArtist"), '') AS name,
			COALESCE(SUM(acts.duration), 0)::bigint AS watch_seconds, COUNT(*) FILTER (WHERE acts.is_view)::int AS views
		FROM acts` + artistJoin + `
		WHERE acts.media_type = 'Audio' AND ` + artistID + ` IS NOT NULL
		GROUP BY 1 HAVING COUNT(*) FILTER (WHERE acts.is_view) > 0
		ORDER BY watch_seconds DESC, views DESC, id LIMIT 5`)
	if v2Fail(c, h.raw(c, &out.TopArtists, q, args...)) {
		return
	}

	// Buckets (+ first listens of each artist, all time).
	bucketFmt := "YYYY-MM"
	if gran == "day" {
		bucketFmt = "YYYY-MM-DD"
	}
	var brows []recapBucket
	q, args = f.sql(`SELECT to_char(local_ts, '` + bucketFmt + `') AS key, ` + aggSeconds + ` AS watch_seconds, ` + aggPlays + ` AS views
		FROM acts GROUP BY 1`)
	if v2Fail(c, h.raw(c, &brows, q, args...)) {
		return
	}
	var arows []struct {
		Key string
		N   int
	}
	q, args = acts().forUser(userID).where(`a."IsView"`).where(`a."MediaType" = 'Audio'`).sql(`,
		firsts AS (
		  SELECT `+artistID+` AS artist_id, MIN(acts.started_at) AS first
		  FROM acts`+artistJoin+` WHERE `+artistID+` IS NOT NULL GROUP BY 1
		)
		SELECT to_char(first AT TIME ZONE '`+tzName()+`', '`+bucketFmt+`') AS key, COUNT(*)::int AS n
		FROM firsts WHERE first >= ? AND first < ? GROUP BY 1`, cur.From, cur.To)
	if v2Fail(c, h.raw(c, &arows, q, args...)) {
		return
	}
	byKey := map[string]recapBucket{}
	for _, b := range brows {
		byKey[b.Key] = b
	}
	for _, a := range arows {
		b := byKey[a.Key]
		b.NewArtists = a.N
		byKey[a.Key] = b
	}
	out.Buckets = []recapBucket{}
	step := func(t time.Time) time.Time { return t.AddDate(0, 1, 0) }
	layout := "2006-01"
	if gran == "day" {
		step = func(t time.Time) time.Time { return t.AddDate(0, 0, 1) }
		layout = v2DateLayout
	}
	for d := cur.From; d.Before(cur.To); d = step(d) {
		k := d.Format(layout)
		b := byKey[k]
		b.Key = k
		out.Buckets = append(out.Buckets, b)
		if b.WatchSeconds > 0 && (out.Records.BestBucket == nil || b.WatchSeconds > out.Records.BestBucket.WatchSeconds) {
			bb := b
			out.Records.BestBucket = &bb
		}
	}

	// Records: biggest binge, longest streak.
	var binge []struct {
		SeriesID, SeriesName, Date string
		Episodes                   int
	}
	q, args = f.sql(bingeCTE + `
		SELECT series_id, COALESCE(series_name, '') AS series_name, to_char(day, 'YYYY-MM-DD') AS date, episode_count::int AS episodes
		FROM binge_sessions WHERE series_id IS NOT NULL ORDER BY episode_count DESC, day DESC LIMIT 1`)
	if v2Fail(c, h.raw(c, &binge, q, args...)) {
		return
	}
	if len(binge) == 1 {
		b := binge[0]
		out.Records.Binge = &struct {
			SeriesID   string `json:"seriesId"`
			SeriesName string `json:"seriesName"`
			Episodes   int    `json:"episodes"`
			Date       string `json:"date"`
		}{b.SeriesID, b.SeriesName, b.Episodes, b.Date}
	}
	var days []string
	q, args = f.sql(`SELECT DISTINCT ` + localDay + ` AS d FROM acts WHERE is_view ORDER BY 1`)
	if v2Fail(c, h.raw(c, &days, q, args...)) {
		return
	}
	out.Records.LongestStreak = longestStreak(days)

	// Ranking: top 10 titles by views vs their rank in the previous period.
	type ranked struct {
		ID, Kind, Name string
		ParentName     *string
		Views          int
		Rank           int
	}
	rank := func(f *actFilter, limit int) ([]ranked, error) {
		var rows []ranked
		q, args := f.sql(`SELECT group_id AS id, `+kindExpr+` AS kind, COALESCE(MAX(group_name), '') AS name,
				CASE WHEN display_type = 'Audio' THEN MAX(al."AlbumArtist") END AS parent_name,
				`+aggPlays+` AS views,
				ROW_NUMBER() OVER (ORDER BY `+aggPlays+` DESC, `+aggSeconds+` DESC, group_id)::int AS rank
			FROM acts LEFT JOIN jf_library_items al ON display_type = 'Audio' AND al."Id" = acts.group_id
			WHERE group_id IS NOT NULL
			GROUP BY group_id, display_type HAVING `+aggPlays+` > 0
			ORDER BY rank LIMIT ?`, limit)
		return rows, h.raw(c, &rows, q, args...)
	}
	now, err := rank(f, 10)
	if v2Fail(c, err) {
		return
	}
	before, err := rank(p.filter(prev), 1000)
	if v2Fail(c, err) {
		return
	}
	prevRank := map[string]int{}
	for _, r := range before {
		prevRank[r.Kind+"/"+r.ID] = r.Rank
	}
	out.Ranking = make([]recapRank, len(now))
	for i, r := range now {
		rr := recapRank{ID: r.ID, Kind: r.Kind, Name: r.Name, ParentName: r.ParentName, Views: r.Views, Rank: r.Rank}
		if pr, ok := prevRank[r.Kind+"/"+r.ID]; ok {
			v := pr
			rr.PreviousRank = &v
		}
		out.Ranking[i] = rr
	}
	c.JSON(http.StatusOK, out)
}

// longestStreak returns the longest run of consecutive dates (YYYY-MM-DD).
func longestStreak(days []string) int {
	sort.Strings(days)
	best, run := 0, 0
	var prev time.Time
	for i, d := range days {
		t, err := time.Parse(v2DateLayout, d)
		if err != nil {
			continue
		}
		if i > 0 && t.Sub(prev) == 24*time.Hour {
			run++
		} else {
			run = 1
		}
		if run > best {
			best = run
		}
		prev = t
	}
	return best
}
