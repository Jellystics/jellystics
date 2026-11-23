// Package statsql holds the SQL building blocks shared by every playback
// statistics query (frontend stats endpoints, Prometheus metrics, repository
// aggregates) so that "a play", "watch time", "completed" and "a local day"
// mean exactly the same thing everywhere.
//
// Definitions (jf_playback_activity, one row = one playback attempt):
//   - only Movie / Episode / Audio rows are ever counted (MediaFilter);
//   - a play (view) is a row with IsView = true (>= 30 s really watched);
//   - a click is any row;
//   - watch time is SUM(PlaybackDuration) over all rows (seconds);
//   - a view is completed when the stop position or the watched time reaches
//     CompletionThreshold of the item runtime (CompletionRatio);
//   - day / hour / weekday buckets are computed in the display time zone
//     (TZ env var, default UTC) via LocalTS.
package statsql

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// MediaTypes is the SQL list of media types that count in statistics.
const MediaTypes = `('Movie','Episode','Audio')`

// CompletionThreshold is the fraction of the runtime a view must reach to be
// considered "completed" (UI and Prometheus alike).
const CompletionThreshold = 0.9

// MediaFilter returns `<alias>."MediaType" IN ('Movie','Episode','Audio')`.
func MediaFilter(alias string) string {
	return col(alias, "MediaType") + ` IN ` + MediaTypes
}

// StartedAt returns the authoritative start timestamp (timestamptz) of a
// playback row: the plain indexed "StartedAt" column, so range predicates can
// use idx_playback_started_at. Legacy rows get it from
// playbackRepo.BackfillTracking (run at startup and after every import); rows
// it cannot date (unparseable ActivityDateInserted) stay excluded from stats.
func StartedAt(alias string) string {
	return col(alias, "StartedAt")
}

// LocalTS returns the start timestamp converted to wall-clock time in the
// display time zone (timestamp without time zone), for day/hour bucketing.
func LocalTS(alias string) string {
	name, _ := Timezone()
	return `(` + StartedAt(alias) + ` AT TIME ZONE '` + name + `')`
}

// AtLocal converts any timestamptz SQL expression to display-zone wall time.
func AtLocal(expr string) string {
	name, _ := Timezone()
	return `((` + expr + `) AT TIME ZONE '` + name + `')`
}

// ISO formats a timestamptz SQL expression as an RFC 3339 UTC string.
func ISO(expr string) string {
	return `to_char((` + expr + `) AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`
}

// CompletionRatio returns a SQL expression in [0,1] (NULL when the runtime is
// unknown) from a stop position (ticks), a watched duration (seconds) and a
// runtime (ticks).
func CompletionRatio(posTicks, durSeconds, runtimeTicks string) string {
	return fmt.Sprintf(`LEAST(GREATEST(COALESCE(%s, 0)::float8, COALESCE(%s, 0)::float8 * 10000000.0) / NULLIF(%s, 0)::float8, 1.0)`,
		posTicks, durSeconds, runtimeTicks)
}

// CatalogJoins resolves, for a jf_playback_activity row aliased `a`, the
// catalog rows of the item actually played (ItemId), keyed by media type so
// each join matches at most one row (no fan-out):
//
//	cm  = movie (jf_library_items)      ce  = episode (jf_library_episodes)
//	cs  = series of the episode         ct  = track (jf_music_tracks)
//	cal = album of the track (jf_library_items)
const CatalogJoins = `
	LEFT JOIN jf_library_items cm    ON a."MediaType" = 'Movie'   AND cm."Id" = a."ItemId"
	LEFT JOIN jf_library_episodes ce ON a."MediaType" = 'Episode' AND ce."Id" = a."ItemId"
	LEFT JOIN jf_library_items cs    ON a."MediaType" = 'Episode' AND cs."Id" = COALESCE(ce."SeriesId", a."NowPlayingItemId")
	LEFT JOIN jf_music_tracks ct     ON a."MediaType" = 'Audio'   AND ct."Id" = a."ItemId"
	LEFT JOIN jf_library_items cal   ON a."MediaType" = 'Audio'   AND cal."Id" = COALESCE(ct."AlbumId", a."NowPlayingItemId")`

// LibraryID is the library of a playback row (requires CatalogJoins).
const LibraryID = `COALESCE(cm."ParentId", cs."ParentId", ct."LibraryId", cal."ParentId")`

func col(alias, name string) string {
	if alias == "" {
		return `"` + name + `"`
	}
	return alias + `."` + name + `"`
}

// ---------------------------------------------------------------------------
// Display time zone
// ---------------------------------------------------------------------------

var (
	tzNameRe = regexp.MustCompile(`^[A-Za-z0-9_+\-/]+$`)
	tzCache  sync.Map // name -> *time.Location
)

// Timezone returns the display time zone configured through the TZ env var
// (e.g. "Europe/Paris"), validated with time.LoadLocation. It falls back to
// UTC when unset or invalid. The returned name is safe to embed in SQL: it is
// a known IANA zone name made only of [A-Za-z0-9_+-/].
func Timezone() (string, *time.Location) {
	name := strings.TrimPrefix(strings.TrimSpace(os.Getenv("TZ")), ":")
	if name == "" || name == "Local" || !tzNameRe.MatchString(name) {
		return "UTC", time.UTC
	}
	if loc, ok := tzCache.Load(name); ok {
		return name, loc.(*time.Location)
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return "UTC", time.UTC
	}
	tzCache.Store(name, loc)
	return name, loc
}

// Location returns only the display *time.Location.
func Location() *time.Location {
	_, loc := Timezone()
	return loc
}

// StartOfDay returns local midnight of t in loc.
func StartOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}
