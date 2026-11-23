package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/statsql"
)

// ---------------------------------------------------------------------------
// Time ranges (display time zone, end-exclusive)
// ---------------------------------------------------------------------------

// TimeRange holds the [From, To) bounds used to filter playback rows by their
// start time. AllTime disables filtering.
type TimeRange struct {
	From    time.Time
	To      time.Time
	AllTime bool
}

// ---------------------------------------------------------------------------
// The `acts` CTE: one normalised row per counted playback attempt
// ---------------------------------------------------------------------------

// catalogJoins resolves the user and the catalog rows of the item actually
// played (see statsql.CatalogJoins for the aliases).
const catalogJoins = `
	LEFT JOIN jf_users u ON u."Id" = a."UserId"` + statsql.CatalogJoins

// actsColumns documents the columns exposed by the acts CTE:
//
//	id, user_id, user_name, item_id (movie/episode/track), media_type
//	(Movie|Episode|Audio), display_type (Movie|Series|Audio), group_id /
//	group_name (movie, series or album: the level shown in "top" lists),
//	item_name, library_id, genres (NULL when the item is not in the catalog),
//	started_at, ended_at (timestamptz), local_ts (display-zone wall time),
//	duration (s), is_view, play_method, client, device_name, runtime_ticks,
//	pos_end, now_playing_item_id, now_playing_item_name, series_name,
//	paused_seconds, app_version, pos_start
func actsSelect() string {
	started := statsql.StartedAt("a")
	return `
	SELECT
	  a."Id" AS id,
	  a."UserId" AS user_id,
	  COALESCE(u."Name", a."UserName", a."UserId") AS user_name,
	  a."ItemId" AS item_id,
	  a."MediaType" AS media_type,
	  CASE a."MediaType" WHEN 'Episode' THEN 'Series' WHEN 'Audio' THEN 'Audio' ELSE 'Movie' END AS display_type,
	  CASE a."MediaType"
	    WHEN 'Movie'   THEN a."ItemId"
	    WHEN 'Episode' THEN COALESCE(cs."Id", ce."SeriesId", a."NowPlayingItemId", a."ItemId")
	    ELSE COALESCE(cal."Id", ct."AlbumId", a."NowPlayingItemId", a."ItemId")
	  END AS group_id,
	  CASE a."MediaType"
	    WHEN 'Movie'   THEN COALESCE(cm."Name", a."NowPlayingItemName")
	    WHEN 'Episode' THEN COALESCE(cs."Name", NULLIF(a."SeriesName", ''), ce."SeriesName", a."NowPlayingItemName")
	    ELSE COALESCE(cal."Name", ct."AlbumName", a."NowPlayingItemName")
	  END AS group_name,
	  COALESCE(cm."Name", ce."Name", ct."Name", a."NowPlayingItemName") AS item_name,
	  ` + statsql.LibraryID + ` AS library_id,
	  CASE a."MediaType"
	    WHEN 'Movie'   THEN cm."Genres"
	    WHEN 'Episode' THEN cs."Genres"
	    ELSE CASE WHEN jsonb_array_length(COALESCE(ct."Genres", '[]'::jsonb)) > 0 THEN ct."Genres"
	              ELSE COALESCE(cal."Genres", ct."Genres") END
	  END AS genres,
	  ` + started + ` AS started_at,
	  COALESCE(a."EndedAt", ` + started + ` + make_interval(secs => COALESCE(a."PlaybackDuration", 0)::float8)) AS ended_at,
	  ` + statsql.LocalTS("a") + ` AS local_ts,
	  COALESCE(a."PlaybackDuration", 0)::bigint AS duration,
	  a."IsView" AS is_view,
	  COALESCE(NULLIF(a."PlayMethod", ''), 'Unknown') AS play_method,
	  a."Client" AS client,
	  a."DeviceName" AS device_name,
	  COALESCE(a."RunTimeTicks", cm."RunTimeTicks", ce."RunTimeTicks", ct."RunTimeTicks") AS runtime_ticks,
	  a."PositionTicksEnd" AS pos_end,
	  a."NowPlayingItemId" AS now_playing_item_id,
	  a."NowPlayingItemName" AS now_playing_item_name,
	  a."SeriesName" AS series_name,
	  COALESCE(a."PausedSeconds", 0)::bigint AS paused_seconds,
	  a."ApplicationVersion" AS app_version,
	  a."PositionTicksStart" AS pos_start
	FROM jf_playback_activity a` + catalogJoins + `
	WHERE ` + statsql.MediaFilter("a")
}

// actFilter accumulates WHERE conditions (on jf_playback_activity alias `a`)
// for the acts CTE.
type actFilter struct {
	conds []string
	args  []interface{}
}

func acts() *actFilter { return &actFilter{} }

func (f *actFilter) where(cond string, args ...interface{}) *actFilter {
	f.conds = append(f.conds, cond)
	f.args = append(f.args, args...)
	return f
}

func (f *actFilter) inRange(tr TimeRange) *actFilter {
	if tr.AllTime {
		return f
	}
	s := statsql.StartedAt("a")
	return f.where(s+` >= ? AND `+s+` < ?`, tr.From, tr.To)
}

func (f *actFilter) forUser(userId string) *actFilter {
	if userId == "" {
		return f
	}
	return f.where(`a."UserId" = ?`, userId)
}

// sql returns `WITH acts AS (...) <outer>` and the combined positional args
// (CTE args first, then outerArgs).
func (f *actFilter) sql(outer string, outerArgs ...interface{}) (string, []interface{}) {
	q := actsSelect()
	for _, c := range f.conds {
		q += "\n\t  AND " + c
	}
	args := append(append([]interface{}{}, f.args...), outerArgs...)
	return "WITH acts AS (" + q + "\n)\n" + outer, args
}

// Aggregate snippets over acts.
const (
	aggPlays   = `COUNT(*) FILTER (WHERE is_view)::int`
	aggClicks  = `COUNT(*)::int`
	aggMinutes = `FLOOR(COALESCE(SUM(duration), 0) / 60.0)::int`
	aggSeconds = `COALESCE(SUM(duration), 0)::bigint`
)

// bingeCTE (append after the acts CTE) defines binge_sessions: one row per
// (user, series, local calendar day) on which the user viewed >= 3 distinct
// episodes of that series.
const bingeCTE = `,
ep_days AS (
  SELECT user_id, MAX(user_name) AS user_name,
         group_id AS series_id, MAX(group_name) AS series_name,
         local_ts::date AS day,
         COUNT(DISTINCT item_id) AS episode_count
  FROM acts
  WHERE media_type = 'Episode' AND is_view
  GROUP BY user_id, group_id, local_ts::date
),
binge_sessions AS (
  SELECT * FROM ep_days WHERE episode_count >= 3
)
`

// completionCTE (append after the acts CTE) defines `completion`: one row per
// (user, item, local calendar day) with at least one view and a known runtime.
// The ratio uses the furthest stop position / longest sitting of the day, so
// an episode resumed in two sittings whose last sitting reaches the end counts
// as one completed view, not one completed + one abandoned.
var completionCTE = `,
completion AS (
  SELECT
    MAX(media_type) AS item_type,
    ` + statsql.CompletionRatio("MAX(pos_end)", "MAX(duration)", "MAX(runtime_ticks)") + ` AS completion_ratio
  FROM acts
  WHERE is_view AND runtime_ticks > 0
  GROUP BY user_id, item_id, local_ts::date
)
`

// libraryIdExpr resolves the library of a playback row inside the acts CTE
// WHERE clause (catalog join aliases are in scope there).
const libraryIdExpr = statsql.LibraryID

// genreLateral expands acts.genres into one `genre` row per genre; catalog
// items without genres yield "No Genre". Rows whose item is not in the
// catalog (genres IS NULL) must be filtered out by the caller.
const genreLateral = `
	CROSS JOIN LATERAL jsonb_array_elements_text(
	  CASE WHEN jsonb_typeof(acts.genres) <> 'array' OR jsonb_array_length(acts.genres) = 0
	       THEN '["No Genre"]'::jsonb ELSE acts.genres END
	) AS genre`

func strOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// completedCond is the shared "completed" predicate on completion_ratio.
var completedCond = fmt.Sprintf(`completion_ratio >= %g`, statsql.CompletionThreshold)

// inPlaceholders returns "?,?,?" for n values.
func inPlaceholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
