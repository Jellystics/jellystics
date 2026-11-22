package handler

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
)

// jsonOf scans a json/jsonb/text column holding JSON into T (NULL → zero T).
type jsonOf[T any] struct{ V T }

func (j *jsonOf[T]) Scan(src interface{}) error {
	var b []byte
	switch v := src.(type) {
	case nil:
		return nil
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("jsonOf: unsupported source %T", src)
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, &j.V)
}

func (j jsonOf[T]) Value() (driver.Value, error) { return json.Marshal(j.V) }

// likePattern escapes LIKE wildcards in q and wraps it in %…%.
func likePattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

// kindExpr maps acts.display_type to the v2 group kind (movie|series|album).
const kindExpr = `CASE display_type WHEN 'Movie' THEN 'movie' WHEN 'Series' THEN 'series' ELSE 'album' END`

// itemKindExpr maps acts.media_type to the kind of the item actually played.
const itemKindExpr = `CASE media_type WHEN 'Movie' THEN 'movie' WHEN 'Episode' THEN 'episode' ELSE 'track' END`

// localDay formats acts.local_ts as YYYY-MM-DD.
const localDay = `to_char(local_ts::date, 'YYYY-MM-DD')`

// ratioExpr is the completion ratio of one acts row (NULL when no runtime).
var ratioExpr = statsql.CompletionRatio("pos_end", "duration", "runtime_ticks")

// methodValue maps a v2 method filter to the stored PlayMethod.
func methodValue(m string) string {
	switch strings.ToLower(m) {
	case "direct", "directplay":
		return "DirectPlay"
	case "stream", "directstream":
		return "DirectStream"
	case "transcode":
		return "Transcode"
	}
	return m
}

// Totals is the common tile block of v2 responses.
type Totals struct {
	WatchSeconds int64    `json:"watchSeconds"`
	Views        int      `json:"views"`
	Clicks       int      `json:"clicks"`
	ActiveUsers  int      `json:"activeUsers"`
	Completion   *float64 `json:"completion"`
}

// totals computes Totals over the acts filtered by f. Completion is the
// average completion ratio of the (user, item, day) view units.
func (h *V2Handler) totals(c *gin.Context, f *actFilter) (Totals, error) {
	var t Totals
	q, args := f.sql(completionCTE + `
		SELECT ` + aggSeconds + ` AS watch_seconds, ` + aggPlays + ` AS views, ` + aggClicks + ` AS clicks,
		       COUNT(DISTINCT user_id)::int AS active_users,
		       (SELECT AVG(completion_ratio)::float8 FROM completion) AS completion
		FROM acts`)
	err := h.raw(c, &t, q, args...)
	return t, err
}

// localTime converts t to the display zone (RFC 3339 with offset once marshalled).
func localTime(t time.Time) time.Time { return t.In(statsql.Location()) }

func localTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := localTime(*t)
	return &v
}

// weekHourGrid returns a zeroed 7×24 grid.
func weekHourGrid() [][]int {
	g := make([][]int, 7)
	for i := range g {
		g[i] = make([]int, 24)
	}
	return g
}

// statsqlCompletion is the completion ratio of a (user, item, day) unit: an
// aggregate over its acts rows (same definition as completionCTE).
var statsqlCompletion = statsql.CompletionRatio("MAX(pos_end)", "MAX(duration)", "MAX(runtime_ticks)")

func completionThreshold() float64 { return statsql.CompletionThreshold }

// episodeOrder is the catalog order of episodes aliased `e`: regular seasons
// by number, then specials (season 0) last.
const episodeOrder = `COALESCE(e."ParentIndexNumber" = 0, false), COALESCE(e."ParentIndexNumber", 0), e."IndexNumber" NULLS LAST, e."Id"`

// notSpecial excludes specials (season 0) of episodes aliased `e`.
const notSpecial = `COALESCE(e."ParentIndexNumber", -1) <> 0`
