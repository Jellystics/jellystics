package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// V2Params is the common query contract of every /api/v2 statistics endpoint.
//
//	from, to   YYYY-MM-DD local dates (display time zone), `to` exclusive.
//	           Default: the last 30 days ending at tomorrow's local midnight.
//	compare    1|true → Prev holds the previous period of the same number of
//	           calendar days, ending at From.
//	userId, libraryId, type (movie|series|music → MediaType)
//	page, pageSize (default 1 / 25, pageSize capped at 200), sort, desc, q
type V2Params struct {
	Range     TimeRange
	Prev      *TimeRange
	UserID    string
	LibraryID string
	Type      string // movie | series | music ("" = all)
	MediaType string // Movie | Episode | Audio ("" = all)
	Page      int
	PageSize  int
	Sort      string
	Desc      bool
	Q         string
}

const (
	v2DefaultPageSize = 25
	v2MaxPageSize     = 200
	v2DateLayout      = "2006-01-02"
)

var errBadParam = errors.New("invalid parameter")

type paramError struct{ msg string }

func (e paramError) Error() string { return e.msg }
func (e paramError) Unwrap() error { return errBadParam }

func badParam(msg string) error { return paramError{msg} }

// parseLocalDate parses a strict YYYY-MM-DD date as local midnight in loc.
func parseLocalDate(s string, loc *time.Location) (time.Time, bool) {
	t, err := time.ParseInLocation(v2DateLayout, s, loc)
	if err != nil || t.Format(v2DateLayout) != s {
		return time.Time{}, false
	}
	return t, true
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// v2Type maps the `type` query value to (normalised type, MediaType).
func v2Type(s string) (string, string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return "", "", true
	case "movie", "movies":
		return "movie", "Movie", true
	case "series", "tvshows", "episode":
		return "series", "Episode", true
	case "music", "audio":
		return "music", "Audio", true
	}
	return "", "", false
}

func parseV2(c *gin.Context) (V2Params, error) {
	return parseV2At(c, time.Now())
}

func parseV2At(c *gin.Context, now time.Time) (V2Params, error) {
	loc := statsql.Location()
	p := V2Params{Page: 1, PageSize: v2DefaultPageSize}

	tomorrow := statsql.StartOfDay(now, loc).AddDate(0, 0, 1)
	fromStr, toStr := strings.TrimSpace(c.Query("from")), strings.TrimSpace(c.Query("to"))
	var from, to time.Time
	if toStr != "" {
		t, ok := parseLocalDate(toStr, loc)
		if !ok {
			return p, badParam("invalid to")
		}
		to = t
	} else {
		to = tomorrow
	}
	if fromStr != "" {
		t, ok := parseLocalDate(fromStr, loc)
		if !ok {
			return p, badParam("invalid from")
		}
		from = t
	} else {
		from = to.AddDate(0, 0, -30)
	}
	if !from.Before(to) {
		return p, badParam("from must be before to")
	}
	p.Range = TimeRange{From: from, To: to}

	if parseBool(c.Query("compare")) {
		days := calendarDays(from, to)
		prev := TimeRange{From: from.AddDate(0, 0, -days), To: from}
		p.Prev = &prev
	}

	p.UserID = strings.TrimSpace(c.Query("userId"))
	p.LibraryID = strings.TrimSpace(c.Query("libraryId"))
	typ, mt, ok := v2Type(c.Query("type"))
	if !ok {
		return p, badParam("invalid type")
	}
	p.Type, p.MediaType = typ, mt

	if s := c.Query("page"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			return p, badParam("invalid page")
		}
		p.Page = n
	}
	if s := c.Query("pageSize"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			return p, badParam("invalid pageSize")
		}
		if n > v2MaxPageSize {
			n = v2MaxPageSize
		}
		p.PageSize = n
	}
	p.Sort = strings.TrimSpace(c.Query("sort"))
	p.Desc = parseBool(c.Query("desc"))
	p.Q = strings.TrimSpace(c.Query("q"))
	return p, nil
}

// calendarDays is the number of local calendar days in [from, to) (both local
// midnights), independent of DST transitions.
func calendarDays(from, to time.Time) int {
	a := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(b.Sub(a).Hours() / 24)
}

// Offset of the current page.
func (p V2Params) Offset() int { return (p.Page - 1) * p.PageSize }

// filter returns an acts filter for range tr plus the user / type / library
// filters of the request.
func (p V2Params) filter(tr TimeRange) *actFilter {
	f := acts().inRange(tr).forUser(p.UserID)
	if p.MediaType != "" {
		f.where(`a."MediaType" = ?`, p.MediaType)
	}
	if p.LibraryID != "" {
		f.where(statsql.LibraryID+` = ?`, p.LibraryID)
	}
	return f
}

// ---------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------

// V2Handler serves every /api/v2 statistics endpoint.
type V2Handler struct {
	db *gorm.DB
}

// NewV2Handler builds the /api/v2 statistics handler.
func NewV2Handler(db *gorm.DB) *V2Handler { return &V2Handler{db: db} }

// raw runs a parameterised query with the request context and scans into dest.
func (h *V2Handler) raw(c *gin.Context, dest interface{}, q string, args ...interface{}) error {
	return h.db.WithContext(c.Request.Context()).Raw(q, args...).Scan(dest).Error
}

// v2Params parses the common contract; on error it writes a 400 and returns false.
func v2Params(c *gin.Context) (V2Params, bool) {
	p, err := parseV2(c)
	if err != nil {
		var pe paramError
		msg := "invalid parameters"
		if errors.As(err, &pe) {
			msg = pe.msg
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return p, false
	}
	return p, true
}

// v2Fail logs err and writes a generic 500. Returns true when err != nil.
func v2Fail(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	log.Printf("api v2 %s: %v", c.FullPath(), err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	return true
}

func v2NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}

// localDate formats a local midnight as YYYY-MM-DD.
func localDate(t time.Time) string { return t.In(statsql.Location()).Format(v2DateLayout) }

// eachDay calls fn for every local date in tr.
func eachDay(tr TimeRange, fn func(date string)) {
	loc := statsql.Location()
	for d := tr.From.In(loc); d.Before(tr.To); d = d.AddDate(0, 0, 1) {
		fn(d.Format(v2DateLayout))
	}
}

func logV2(c *gin.Context, err error) { log.Printf("api v2 %s: %v", c.FullPath(), err) }

// Register mounts every /api/v2 statistics route on g (the authed group).
func (h *V2Handler) Register(g *gin.RouterGroup) {
	v2 := g.Group("/api/v2")
	v2.GET("/overview", h.Overview)
	v2.GET("/history", h.History)
	v2.GET("/history.csv", h.HistoryCSV)
	v2.GET("/history/calendar", h.HistoryCalendar)
	v2.GET("/movies/:id", h.Movie)
	v2.GET("/series/:id", h.Series)
	v2.GET("/episodes/:id", h.Episode)
	v2.GET("/artists/:id", h.Artist)
	v2.GET("/albums/:id", h.Album)
	v2.GET("/tracks/:id", h.Track)
	v2.GET("/search", h.Search)
	v2.GET("/users", h.Users)
	v2.GET("/users/:id", h.User)
	v2.GET("/libraries", h.Libraries)
	v2.GET("/libraries/:id", h.Library)
}
