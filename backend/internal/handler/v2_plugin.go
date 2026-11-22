package handler

// Precise statistics fed by the Jellystics Jellyfin plugin (phase C):
// forward skips / intro skipping, "played to the end" (PlayedToCompletion),
// real library additions (jf_library_events), logins (jf_user_logins) and
// manual vs automatic played marks (jf_user_data_events).
//
// Tracker rows and imports leave the plugin columns NULL: every metric here
// is computed on plugin rows only and its response carries the date the
// plugin data starts at (pluginSince / collectedSince), so the UI can hide
// the widget instead of showing zeros. Open plugin plays (PluginOpen) are
// provisional: they count in watch time, never in completion / skip stats.

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Jellystics/Jellystics/internal/middleware"
	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// First plugin data dates
// ---------------------------------------------------------------------------

// pluginSinceCache keeps, per database and source, the first plugin data
// date once known (reset with ResetStatsCaches). "Nothing yet" is not cached.
var pluginSinceCache = struct {
	sync.Mutex
	m map[*gorm.DB]map[string]string
}{m: map[*gorm.DB]map[string]string{}}

func resetPluginSinceCache() {
	pluginSinceCache.Lock()
	pluginSinceCache.m = map[*gorm.DB]map[string]string{}
	pluginSinceCache.Unlock()
}

// pluginSources: SQL returning the first timestamp of each plugin source.
var pluginSources = map[string]string{
	// PluginOpen is only set on rows recorded by the Jellystics plugin.
	"plays":    `SELECT MIN("StartedAt") AS first FROM jf_playback_activity WHERE "PluginOpen" IS NOT NULL`,
	"library":  `SELECT MIN("At") AS first FROM jf_library_events`,
	"userData": `SELECT MIN("At") AS first FROM jf_user_data_events`,
	"logins":   `SELECT MIN("At") AS first FROM jf_user_logins`,
}

// pluginSince is the local date (YYYY-MM-DD) of the first row of a plugin
// source, nil when the plugin has not sent any yet.
func (h *V2Handler) pluginSince(c *gin.Context, source string) (*string, error) {
	pluginSinceCache.Lock()
	if v, ok := pluginSinceCache.m[h.db][source]; ok {
		pluginSinceCache.Unlock()
		return &v, nil
	}
	pluginSinceCache.Unlock()
	var row struct{ First *time.Time }
	if err := h.raw(c, &row, pluginSources[source]); err != nil {
		return nil, err
	}
	if row.First == nil || row.First.IsZero() {
		return nil, nil
	}
	d := localDate(*row.First)
	pluginSinceCache.Lock()
	if pluginSinceCache.m[h.db] == nil {
		pluginSinceCache.m[h.db] = map[string]string{}
	}
	pluginSinceCache.m[h.db][source] = d
	pluginSinceCache.Unlock()
	return &d, nil
}

// ---------------------------------------------------------------------------
// Skips (SeekCount / SkippedSeconds of closed plugin plays)
// ---------------------------------------------------------------------------

const (
	// introSkipMaxShare: an intro / credits skip is a short forward jump, at
	// most this share of the runtime in total.
	introSkipMaxShare = 0.15
	// introSkipStartShare: ... in a play started from the beginning (start
	// position within this share of the runtime). The plugin records how much
	// was skipped, not where: "intro skipped" is an estimate built on both.
	introSkipStartShare = 0.05
)

// skipEligible: the acts rows whose skips are known (closed plugin views
// with a runtime).
const skipEligible = `(plugin_closed AND is_view AND runtime_ticks > 0 AND skipped_seconds IS NOT NULL)`

const skipAny = `skipped_seconds > 0`

var skipIntro = fmt.Sprintf(`(skipped_seconds > 0 AND COALESCE(pos_start, 0) <= runtime_ticks * %g AND skipped_seconds * 10000000.0 <= runtime_ticks * %g)`,
	introSkipStartShare, introSkipMaxShare)

// skipCols aggregates the skip counters over acts (or a GROUP BY of it).
var skipCols = `COUNT(*) FILTER (WHERE ` + skipEligible + `)::int AS skip_plays,
	COUNT(*) FILTER (WHERE ` + skipEligible + ` AND ` + skipAny + `)::int AS skipped,
	COUNT(*) FILTER (WHERE ` + skipEligible + ` AND ` + skipIntro + `)::int AS intro,
	AVG(skipped_seconds) FILTER (WHERE ` + skipEligible + ` AND ` + skipAny + `)::float8 AS avg_skipped_seconds`

// SkipCounts scans skipCols (exported so it can be embedded in scanned rows).
type SkipCounts struct {
	SkipPlays         int
	Skipped           int
	Intro             int
	AvgSkippedSeconds *float64
}

// skipStats: plays = closed plugin views (the only plays whose skips are
// measured); rates are nil without such plays; avgSkippedSeconds is the mean
// forward skip of the plays with a skip.
type skipStats struct {
	Plays             int      `json:"plays"`
	Skipped           int      `json:"skipped"`
	Intro             int      `json:"intro"`
	SkipRate          *float64 `json:"skipRate"`
	IntroRate         *float64 `json:"introRate"`
	AvgSkippedSeconds *float64 `json:"avgSkippedSeconds"`
}

func ratioOf(n, d int) *float64 {
	if d == 0 {
		return nil
	}
	v := float64(n) / float64(d)
	return &v
}

func (r SkipCounts) stats() skipStats {
	return skipStats{Plays: r.SkipPlays, Skipped: r.Skipped, Intro: r.Intro,
		SkipRate: ratioOf(r.Skipped, r.SkipPlays), IntroRate: ratioOf(r.Intro, r.SkipPlays), AvgSkippedSeconds: r.AvgSkippedSeconds}
}

func (h *V2Handler) skips(c *gin.Context, f *actFilter) (skipStats, error) {
	var r SkipCounts
	q, args := f.sql(`SELECT ` + skipCols + ` FROM acts`)
	err := h.raw(c, &r, q, args...)
	return r.stats(), err
}

// ---------------------------------------------------------------------------
// "Played to the end" (PlayedToCompletion, ratio fallback)
// ---------------------------------------------------------------------------

// playedToEnd counts the closed views whose outcome is known: measured by
// the plugin (PlayedToCompletion) or, for other rows, estimated from the
// completion ratio (>= CompletionThreshold). Measured = 0 means the plugin
// has no data for the period.
type playedToEnd struct {
	Plays            int      `json:"plays"`
	Finished         int      `json:"finished"`
	Measured         int      `json:"measured"`
	MeasuredFinished int      `json:"measuredFinished"`
	Ratio            *float64 `json:"ratio"`
}

var playedToEndEligible = `(is_view AND NOT plugin_open AND (played_to_completion IS NOT NULL OR runtime_ticks > 0))`

var playedToEndFinished = fmt.Sprintf(`COALESCE(played_to_completion, %s >= %g)`,
	statsql.CompletionRatio("pos_end", "duration", "runtime_ticks"), statsql.CompletionThreshold)

func (h *V2Handler) playedToEnd(c *gin.Context, f *actFilter) (playedToEnd, error) {
	var out playedToEnd
	q, args := f.sql(`SELECT
		COUNT(*) FILTER (WHERE ` + playedToEndEligible + `)::int AS plays,
		COUNT(*) FILTER (WHERE ` + playedToEndEligible + ` AND ` + playedToEndFinished + `)::int AS finished,
		COUNT(*) FILTER (WHERE is_view AND NOT plugin_open AND played_to_completion IS NOT NULL)::int AS measured,
		COUNT(*) FILTER (WHERE is_view AND NOT plugin_open AND played_to_completion)::int AS measured_finished
		FROM acts`)
	if err := h.raw(c, &out, q, args...); err != nil {
		return out, err
	}
	out.Ratio = ratioOf(out.Finished, out.Plays)
	return out, nil
}

// ---------------------------------------------------------------------------
// GET /api/v2/insights/skips
// ---------------------------------------------------------------------------

// skipMinPlays: series need this many measured plays to be ranked.
const skipMinPlays = 3

type skippedSeries struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Plays             int      `json:"plays"`
	IntroRate         *float64 `json:"introRate"`
	SkipRate          *float64 `json:"skipRate"`
	AvgSkippedSeconds *float64 `json:"avgSkippedSeconds"`
}

// InsightsSkips → {collectedSince, totals: skipStats, series: top 10 series
// by intro-skip rate (>= skipMinPlays measured plays)} over episodes.
func (h *V2Handler) InsightsSkips(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	since, err := h.pluginSince(c, "plays")
	if v2Fail(c, err) {
		return
	}
	f := p.filter(p.Range).where(`a."MediaType" = 'Episode'`)
	totals, err := h.skips(c, f)
	if v2Fail(c, err) {
		return
	}
	var rows []struct {
		ID, Name string
		SkipCounts
	}
	q, args := f.sql(`SELECT group_id AS id, COALESCE(MAX(group_name), '') AS name, `+skipCols+`
		FROM acts WHERE group_id IS NOT NULL GROUP BY group_id
		HAVING COUNT(*) FILTER (WHERE `+skipEligible+`) >= ?`, skipMinPlays)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	series := make([]skippedSeries, 0, len(rows))
	for _, r := range rows {
		s := r.stats()
		series = append(series, skippedSeries{r.ID, r.Name, s.Plays, s.IntroRate, s.SkipRate, s.AvgSkippedSeconds})
	}
	sort.SliceStable(series, func(i, j int) bool {
		a, b := *series[i].IntroRate, *series[j].IntroRate
		if a != b {
			return a > b
		}
		if series[i].Plays != series[j].Plays {
			return series[i].Plays > series[j].Plays
		}
		return series[i].Name < series[j].Name
	})
	if len(series) > 10 {
		series = series[:10]
	}
	c.JSON(http.StatusOK, gin.H{"collectedSince": since, "totals": totals, "series": series})
}

// ---------------------------------------------------------------------------
// GET /api/v2/users/:id/logins
// ---------------------------------------------------------------------------

type loginDevice struct {
	Device  string    `json:"device"`
	Client  string    `json:"client"`
	Version string    `json:"version"`
	Count   int       `json:"count"`
	LastAt  time.Time `json:"lastAt"`
}

// isAdmin: the caller is a Jellystics admin (false without claims).
func isAdmin(c *gin.Context) bool {
	cl := middleware.GetClaims(c)
	return cl != nil && cl.IsAdmin
}

// UserLogins → {collectedSince, count, lastLogin, devices, failed}: the
// successful logins of the user in the range (by device / client / version),
// the last successful login ever, and the failed attempts in the range
// (admins only, else null; attempts without a resolved user match on the
// user name).
func (h *V2Handler) UserLogins(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	var user struct{ ID, Name string }
	if v2Fail(c, h.raw(c, &user, `SELECT "Id" AS id, COALESCE("Name", '') AS name FROM jf_users WHERE "Id" = ?`, id)) {
		return
	}
	if user.ID == "" {
		v2NotFound(c)
		return
	}
	since, err := h.pluginSince(c, "logins")
	if v2Fail(c, err) {
		return
	}
	devices := []loginDevice{}
	if v2Fail(c, h.raw(c, &devices, `
		SELECT COALESCE("DeviceName", '') AS device, COALESCE("Client", '') AS client, COALESCE("ApplicationVersion", '') AS version,
		       COUNT(*)::int AS count, MAX("At") AS last_at
		FROM jf_user_logins WHERE "UserId" = ? AND "Success" AND "At" >= ? AND "At" < ?
		GROUP BY 1, 2, 3 ORDER BY count DESC, last_at DESC LIMIT 20`, id, p.Range.From, p.Range.To)) {
		return
	}
	count := 0
	for i := range devices {
		devices[i].LastAt = localTime(devices[i].LastAt)
		count += devices[i].Count
	}
	if len(devices) == 20 { // more devices than listed: count them all
		var n struct{ N int }
		if v2Fail(c, h.raw(c, &n, `SELECT COUNT(*)::int AS n FROM jf_user_logins WHERE "UserId" = ? AND "Success" AND "At" >= ? AND "At" < ?`, id, p.Range.From, p.Range.To)) {
			return
		}
		count = n.N
	}
	var last struct{ At *time.Time }
	if v2Fail(c, h.raw(c, &last, `SELECT MAX("At") AS at FROM jf_user_logins WHERE "UserId" = ? AND "Success"`, id)) {
		return
	}
	var failed *int
	if isAdmin(c) {
		var n struct{ N int }
		if v2Fail(c, h.raw(c, &n, `
			SELECT COUNT(*)::int AS n FROM jf_user_logins
			WHERE NOT "Success" AND "At" >= ? AND "At" < ?
			  AND ("UserId" = ? OR ("UserId" IS NULL AND ? <> '' AND lower("UserName") = lower(?)))`,
			p.Range.From, p.Range.To, id, user.Name, user.Name)) {
			return
		}
		failed = &n.N
	}
	c.JSON(http.StatusOK, gin.H{"collectedSince": since, "count": count, "lastLogin": localTimePtr(last.At), "devices": devices, "failed": failed})
}

// ---------------------------------------------------------------------------
// GET /api/v2/items/:id/marks
// ---------------------------------------------------------------------------

// Jellyfin UserDataSaveReason values of a "played" mark.
var (
	manualReasons    = []string{"TogglePlayed"}
	automaticReasons = []string{"PlaybackFinished", "PlaybackProgress", "PlaybackStart"}
)

func sqlList(vals []string) string {
	q := make([]string, len(vals))
	for i, v := range vals {
		q[i] = "'" + strings.ReplaceAll(v, "'", "''") + "'"
	}
	return "(" + strings.Join(q, ",") + ")"
}

// ItemMarks (movie or episode id) → {collectedSince, manual, automatic,
// other, unplayed, watched, manualWithoutView}: users whose current played
// state (latest played / unplayed event) is "played" set by hand
// (TogglePlayed), by playback (PlaybackFinished…) or otherwise (import, API);
// users whose latest event unmarked it; users with a recorded view of the
// item (all time); manual marks of users without any recorded view. All
// time; userId restricts to one user.
func (h *V2Handler) ItemMarks(c *gin.Context) {
	p, ok := h.paramsNoRange(c)
	if !ok {
		return
	}
	id := c.Param("id")
	since, err := h.pluginSince(c, "userData")
	if v2Fail(c, err) {
		return
	}
	var out struct {
		Manual            int `json:"manual"`
		Automatic         int `json:"automatic"`
		Other             int `json:"other"`
		Unplayed          int `json:"unplayed"`
		Watched           int `json:"watched"`
		ManualWithoutView int `json:"manualWithoutView"`
	}
	if v2Fail(c, h.raw(c, &out, `
		WITH latest AS (
		  SELECT DISTINCT ON ("UserId") "UserId" AS user_id, "Kind" AS kind, COALESCE("Reason", '') AS reason
		  FROM jf_user_data_events
		  WHERE "ItemId" = ? AND "Kind" IN ('played', 'unplayed') AND "UserId" IS NOT NULL AND (? = '' OR "UserId" = ?)
		  ORDER BY "UserId", "At" DESC, "Version" DESC
		),
		viewers AS (
		  SELECT DISTINCT "UserId" AS user_id FROM jf_playback_activity
		  WHERE "ItemId" = ? AND "IsView" AND "UserId" IS NOT NULL AND (? = '' OR "UserId" = ?)
		)
		SELECT
		  (SELECT COUNT(*) FROM latest WHERE kind = 'played' AND reason IN `+sqlList(manualReasons)+`)::int AS manual,
		  (SELECT COUNT(*) FROM latest WHERE kind = 'played' AND reason IN `+sqlList(automaticReasons)+`)::int AS automatic,
		  (SELECT COUNT(*) FROM latest WHERE kind = 'played' AND reason NOT IN `+sqlList(append(append([]string{}, manualReasons...), automaticReasons...))+`)::int AS other,
		  (SELECT COUNT(*) FROM latest WHERE kind = 'unplayed')::int AS unplayed,
		  (SELECT COUNT(*) FROM viewers)::int AS watched,
		  (SELECT COUNT(*) FROM latest l WHERE l.kind = 'played' AND l.reason IN `+sqlList(manualReasons)+`
		     AND NOT EXISTS (SELECT 1 FROM viewers v WHERE v.user_id = l.user_id))::int AS manual_without_view`,
		id, p.UserID, p.UserID, id, p.UserID, p.UserID)) {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"collectedSince": since, "manual": out.Manual, "automatic": out.Automatic, "other": out.Other,
		"unplayed": out.Unplayed, "watched": out.Watched, "manualWithoutView": out.ManualWithoutView,
	})
}

// ---------------------------------------------------------------------------
// Library events (real additions)
// ---------------------------------------------------------------------------

// libraryEventsSQL: the added / removed events of the movies, episodes and
// tracks of library ? (four placeholders, all the library id). Events carry
// their library; older ones without it match through the catalog.
const libraryEventsSQL = `
	SELECT ev."ItemId" AS item_id, ev."Kind" AS kind, ev."At" AS at, ev."ItemType" AS item_type
	FROM jf_library_events ev
	WHERE ev."ItemType" IN ('Movie', 'Episode', 'Audio')
	  AND (ev."LibraryId" = ? OR (ev."LibraryId" IS NULL AND ev."ItemId" IN (SELECT l."Id" FROM (` + libraryLeavesSQL + `) l)))`

// libraryEventsSince is the first event of the library (nil: none, the
// catalog dates are the only source).
func (h *V2Handler) libraryEventsSince(c *gin.Context, id string) (*time.Time, error) {
	var row struct{ First *time.Time }
	if err := h.raw(c, &row, `SELECT MIN(at) AS first FROM (`+libraryEventsSQL+`) e`, id, id, id, id); err != nil {
		return nil, err
	}
	if row.First == nil || row.First.IsZero() {
		return nil, nil
	}
	return row.First, nil
}

// ---------------------------------------------------------------------------
// GET /api/v2/libraries/:id/first-view-delay
// ---------------------------------------------------------------------------

var firstViewBuckets = []string{"<1h", "1-24h", "1-7d", "1-4w", ">4w"}

// LibraryFirstViewDelay → {collectedSince, added, viewed, notViewed,
// medianSeconds, buckets}: the movies / episodes / tracks really added to
// the library in the range (first plugin "added" event of the item in the
// range), and the delay to their first view after that addition (by the
// userId user when given). `type` restricts the media type.
func (h *V2Handler) LibraryFirstViewDelay(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if _, ok := h.libraryOf(c, id); !ok {
		return
	}
	first, err := h.libraryEventsSince(c, id)
	if v2Fail(c, err) {
		return
	}
	var since *string
	if first != nil {
		d := localDate(*first)
		since = &d
	}
	var rows []struct {
		Bucket string
		N      int
		Median *float64
	}
	vf := acts().forUser(p.UserID).where(`a."IsView"`)
	if p.MediaType != "" {
		vf.where(`a."MediaType" = ?`, p.MediaType)
	}
	q, args := vf.sql(`,
		ev AS (`+libraryEventsSQL+`),
		added AS (
		  SELECT item_id, MIN(at) AS added_at FROM ev
		  WHERE kind = 'added' AND item_id IS NOT NULL AND at >= ? AND at < ? AND (? = '' OR item_type = ?)
		  GROUP BY item_id
		),
		delays AS (
		  SELECT ad.item_id,
		         (SELECT EXTRACT(EPOCH FROM MIN(acts.started_at) - ad.added_at) FROM acts
		           WHERE acts.item_id = ad.item_id AND acts.started_at >= ad.added_at) AS secs
		  FROM added ad
		)
		SELECT CASE
		         WHEN secs IS NULL THEN 'none'
		         WHEN secs < 3600 THEN '<1h'
		         WHEN secs < 86400 THEN '1-24h'
		         WHEN secs < 7 * 86400 THEN '1-7d'
		         WHEN secs < 28 * 86400 THEN '1-4w'
		         ELSE '>4w'
		       END AS bucket, COUNT(*)::int AS n,
		       (SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY secs) FROM delays WHERE secs IS NOT NULL)::float8 AS median
		FROM delays GROUP BY 1`, id, id, id, id, p.Range.From, p.Range.To, p.MediaType, p.MediaType)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	var counts []bucketCount
	added, viewed, notViewed := 0, 0, 0
	var median *float64
	for _, r := range rows {
		added += r.N
		median = r.Median
		if r.Bucket == "none" {
			notViewed = r.N
			continue
		}
		viewed += r.N
		counts = append(counts, bucketCount{r.Bucket, r.N})
	}
	c.JSON(http.StatusOK, gin.H{
		"collectedSince": since, "added": added, "viewed": viewed, "notViewed": notViewed,
		"medianSeconds": median, "buckets": fillBuckets(firstViewBuckets, counts),
	})
}
