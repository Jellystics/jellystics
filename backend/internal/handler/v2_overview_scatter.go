package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// playsLimit caps /overview/plays to the most recent plays of the period.
const playsLimit = 5000

type playPoint struct {
	ID           string      `json:"id"`
	StartedAt    time.Time   `json:"startedAt"`
	WatchSeconds int64       `json:"watchSeconds"`
	Type         string      `json:"type"` // movie | series | music
	Item         historyItem `json:"item"`
	User         historyUser `json:"user"`
}

type playsResponse struct {
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Capped bool        `json:"capped"`
	Points []playPoint `json:"points"`
}

// GET /api/v2/overview/plays — one point per play (Movie / Episode / Audio)
// of the period: the playsLimit most recent, newest first.
func (h *V2Handler) OverviewPlays(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	var rows []struct {
		ID           string
		StartedAt    time.Time
		WatchSeconds int64
		MediaType    string
		UserID       *string
		UserName     *string
		ItemID       *string
		Kind         string
		ItemName     *string
		ParentID     *string
		ParentName   *string
		Total        int
	}
	q, args := p.filter(p.Range).sql(`
		SELECT id, started_at, GREATEST(duration, 0)::bigint AS watch_seconds, media_type,
		       user_id, user_name, item_id, `+itemKindExpr+` AS kind, item_name,
		       group_id AS parent_id, group_name AS parent_name,
		       COUNT(*) OVER ()::int AS total
		FROM acts
		ORDER BY started_at DESC, id DESC
		LIMIT ?`, playsLimit)
	if v2Fail(c, h.raw(c, &rows, q, args...)) {
		return
	}
	out := playsResponse{Limit: playsLimit, Points: make([]playPoint, 0, len(rows))}
	for _, r := range rows {
		out.Total = r.Total
		pt := playPoint{
			ID: r.ID, StartedAt: localTime(r.StartedAt), WatchSeconds: r.WatchSeconds, Type: mediaFamily(r.MediaType),
			Item: historyItem{ID: strOrEmpty(r.ItemID), Kind: r.Kind, Name: strOrEmpty(r.ItemName)},
			User: historyUser{ID: strOrEmpty(r.UserID), Name: strOrEmpty(r.UserName)},
		}
		if r.MediaType != "Movie" {
			pt.Item.ParentID, pt.Item.ParentName = r.ParentID, r.ParentName
		}
		out.Points = append(out.Points, pt)
	}
	out.Capped = out.Total > len(out.Points)
	c.JSON(http.StatusOK, out)
}

// mediaFamily maps a stored MediaType to the v2 family (movie|series|music).
func mediaFamily(mt string) string {
	switch mt {
	case "Episode":
		return "series"
	case "Audio":
		return "music"
	}
	return "movie"
}

type libraryTotal struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"` // movie | series | music | other (from CollectionType)
	Views int    `json:"views"`
}

type libraryDayPoint struct {
	Date      string `json:"date"`
	LibraryID string `json:"libraryId"`
	Views     int    `json:"views"`
}

type librariesDailyResponse struct {
	Bucket    string            `json:"bucket"` // day | week (ranges > 400 days)
	Libraries []libraryTotal    `json:"libraries"`
	Points    []libraryDayPoint `json:"points"`
}

// GET /api/v2/overview/libraries-daily — views per (day or ISO week, library):
// only non-zero points; libraries ordered by views desc. Plays whose item is
// not in the catalog (no library) are left out.
func (h *V2Handler) OverviewLibrariesDaily(c *gin.Context) {
	p, ok := h.params(c)
	if !ok {
		return
	}
	week := useWeeks(p.Range)
	out := librariesDailyResponse{Bucket: "day", Libraries: []libraryTotal{}, Points: []libraryDayPoint{}}
	if week {
		out.Bucket = "week"
	}
	f := p.filter(p.Range)
	q, args := f.sql(`
		SELECT ` + bucketExpr(week) + ` AS date, library_id, ` + aggPlays + ` AS views
		FROM acts WHERE library_id IS NOT NULL AND is_view
		GROUP BY 1, 2
		ORDER BY 1, 2`)
	if v2Fail(c, h.raw(c, &out.Points, q, args...)) {
		return
	}
	q, args = f.sql(`
		SELECT acts.library_id AS id, COALESCE(MAX(l."Name"), acts.library_id) AS name,
		       CASE MAX(l."CollectionType") WHEN 'movies' THEN 'movie' WHEN 'tvshows' THEN 'series' WHEN 'music' THEN 'music' ELSE 'other' END AS type,
		       ` + aggPlays + ` AS views
		FROM acts LEFT JOIN jf_libraries l ON l."Id" = acts.library_id
		WHERE acts.library_id IS NOT NULL AND is_view
		GROUP BY acts.library_id
		ORDER BY views DESC, name, id`)
	if v2Fail(c, h.raw(c, &out.Libraries, q, args...)) {
		return
	}
	c.JSON(http.StatusOK, out)
}
