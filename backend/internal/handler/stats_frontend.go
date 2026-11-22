package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// StatsFrontendHandler serves all stats endpoints expected by the frontend.
//
// Every endpoint reads jf_playback_activity only (finished playbacks), counts
// Movie / Episode / Audio rows only, and uses the shared definitions from
// package statsql: plays = IsView rows, clicks = all rows, watch time =
// SUM(PlaybackDuration), day/hour buckets in the display time zone (TZ).
// Live sessions are never mixed into historical numbers.
type StatsFrontendHandler struct {
	db    *gorm.DB
	repos *repository.Container
}

// NewStatsFrontendHandler constructs a StatsFrontendHandler.
func NewStatsFrontendHandler(db *gorm.DB, repos *repository.Container) *StatsFrontendHandler {
	return &StatsFrontendHandler{db: db, repos: repos}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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

// parseTimeRange parses ?from=YYYY-MM-DD&to=YYYY-MM-DD (display time zone,
// `to` inclusive as a day, end-exclusive as a bound). Defaults to the last 30
// days; from=all or from=0 means all time.
func parseTimeRange(c *gin.Context) TimeRange {
	return parseRange(c.Query("from"), c.Query("to"))
}

func strOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// Keep parseDays for pagination only (size, page, limit params — NOT time filtering)
func parseDays(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

type pageResult struct {
	CurrentPage int         `json:"current_page"`
	Pages       int         `json:"pages"`
	Size        int         `json:"size"`
	Results     interface{} `json:"results"`
}

func paginate(total int, pageSize int, page int, results interface{}) pageResult {
	pages := 0
	if total > 0 && pageSize > 0 {
		pages = (total + pageSize - 1) / pageSize
	}
	return pageResult{CurrentPage: page, Pages: pages, Size: pageSize, Results: results}
}

func dbError(c *gin.Context, err error) bool {
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// GET /stats/getGlobalStats (all time)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetGlobalStats(c *gin.Context) {
	var stats struct {
		TotalPlays     int `json:"TotalPlays"`
		TotalClicks    int `json:"TotalClicks"`
		TotalWatchTime int `json:"TotalWatchTime"` // minutes
		ActiveUsers    int `json:"ActiveUsers"`
		TotalUsers     int `json:"TotalUsers"`
		TotalLibraries int `json:"TotalLibraries"`
		TotalItems     int `json:"TotalItems"` // playable leaves: movies + episodes + tracks
	}
	activeSince := statsql.StartOfDay(time.Now(), statsql.Location()).AddDate(0, 0, -30)
	q, args := acts().sql(`
		SELECT
		  (SELECT `+aggPlays+` FROM acts) AS "TotalPlays",
		  (SELECT `+aggClicks+` FROM acts) AS "TotalClicks",
		  (SELECT `+aggMinutes+` FROM acts) AS "TotalWatchTime",
		  (SELECT COUNT(DISTINCT user_id)::int FROM acts WHERE is_view AND started_at >= ?) AS "ActiveUsers",
		  (SELECT COUNT(*)::int FROM jf_users) AS "TotalUsers",
		  (SELECT COUNT(*)::int FROM jf_libraries WHERE archived = false) AS "TotalLibraries",
		  ((SELECT COUNT(*)::int FROM jf_library_items WHERE archived = false AND "Type" = 'Movie')
		   + (SELECT COUNT(*)::int FROM jf_library_episodes WHERE archived = false)
		   + (SELECT COUNT(*)::int FROM jf_music_tracks WHERE archived = false)) AS "TotalItems"
	`, activeSince)
	if dbError(c, h.db.Raw(q, args...).Scan(&stats).Error) {
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ---------------------------------------------------------------------------
// GET /stats/getMostPlayedItems?type=all|Movie|Series|Audio&limit=5&from=&to=
// Movies are listed per movie, episodes per series, tracks per album.
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetMostPlayedItems(c *gin.Context) {
	itemType := c.DefaultQuery("type", "all")
	limit := parseDays(c.DefaultQuery("limit", "5"), 5)
	tr := parseTimeRange(c)

	type Item struct {
		Id             string `json:"Id"`
		Name           string `json:"Name"`
		PlayCount      int    `json:"PlayCount"`
		ClickCount     int    `json:"ClickCount"`
		TotalWatchTime int    `json:"TotalWatchTime"` // minutes
		Type           string `json:"Type"`
	}
	items := []Item{}

	f := acts().inRange(tr)
	if itemType != "" && itemType != "all" {
		mt := mediaTypeParam(itemType)
		if mt == "" {
			c.JSON(http.StatusOK, items)
			return
		}
		f.where(`a."MediaType" = ?`, mt)
	}
	q, args := f.sql(`
		SELECT group_id AS "Id", COALESCE(MAX(group_name), '') AS "Name",
		       `+aggPlays+` AS "PlayCount", `+aggClicks+` AS "ClickCount",
		       `+aggMinutes+` AS "TotalWatchTime", display_type AS "Type"
		FROM acts
		WHERE group_id IS NOT NULL
		GROUP BY group_id, display_type
		HAVING COUNT(*) FILTER (WHERE is_view) > 0
		ORDER BY "PlayCount" DESC, "TotalWatchTime" DESC, "Name"
		LIMIT ?`, limit)
	if dbError(c, h.db.Raw(q, args...).Scan(&items).Error) {
		return
	}
	c.JSON(http.StatusOK, items)
}

// ---------------------------------------------------------------------------
// GET /stats/getMostActiveUsers?limit=5&from=&to=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetMostActiveUsers(c *gin.Context) {
	tr := parseTimeRange(c)
	limit := parseDays(c.DefaultQuery("limit", "5"), 5)

	type UserRow struct {
		UserId         string `json:"UserId"`
		UserName       string `json:"UserName"`
		TotalPlays     int    `json:"TotalPlays"`
		TotalClicks    int    `json:"TotalClicks"`
		TotalWatchTime int    `json:"TotalWatchTime"`
	}
	rows := []UserRow{}
	q, args := acts().inRange(tr).sql(`
		SELECT user_id AS "UserId", MAX(user_name) AS "UserName",
		       `+aggPlays+` AS "TotalPlays", `+aggClicks+` AS "TotalClicks",
		       `+aggMinutes+` AS "TotalWatchTime"
		FROM acts
		WHERE user_id IS NOT NULL
		GROUP BY user_id
		ORDER BY "TotalPlays" DESC, "TotalWatchTime" DESC
		LIMIT ?`, limit)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getWatchStatisticsOverTime?from=&to=&userId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetWatchStatisticsOverTime(c *gin.Context) {
	tr := parseTimeRange(c)
	type Row struct {
		Date     string `json:"date"`
		Plays    int    `json:"plays"`
		Clicks   int    `json:"clicks"`
		Duration int    `json:"duration"` // minutes
	}
	rows := []Row{}
	q, args := acts().inRange(tr).forUser(c.Query("userId")).sql(`
		SELECT TO_CHAR(local_ts::date, 'YYYY-MM-DD') AS date,
		       ` + aggPlays + ` AS plays, ` + aggClicks + ` AS clicks, ` + aggMinutes + ` AS duration
		FROM acts
		GROUP BY local_ts::date
		ORDER BY local_ts::date`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getPopularHourOfDay / getPopularDayOfWeek
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetPopularHourOfDay(c *gin.Context) {
	tr := parseTimeRange(c)
	type Row struct {
		Hour     int `json:"hour"`
		Plays    int `json:"plays"`
		Clicks   int `json:"clicks"`
		Duration int `json:"duration"`
	}
	rows := []Row{}
	q, args := acts().inRange(tr).forUser(c.Query("userId")).sql(`
		SELECT EXTRACT(HOUR FROM local_ts)::int AS hour,
		       ` + aggPlays + ` AS plays, ` + aggClicks + ` AS clicks, ` + aggMinutes + ` AS duration
		FROM acts GROUP BY 1 ORDER BY 1`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StatsFrontendHandler) GetPopularDayOfWeek(c *gin.Context) {
	tr := parseTimeRange(c)
	type Row struct {
		Day      int `json:"day"`
		Plays    int `json:"plays"`
		Clicks   int `json:"clicks"`
		Duration int `json:"duration"`
	}
	rows := []Row{}
	q, args := acts().inRange(tr).forUser(c.Query("userId")).sql(`
		SELECT EXTRACT(DOW FROM local_ts)::int AS day,
		       ` + aggPlays + ` AS plays, ` + aggClicks + ` AS clicks, ` + aggMinutes + ` AS duration
		FROM acts GROUP BY 1 ORDER BY 1`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getMostUsedPlaybackMethod / getMostUsedClients
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetMostUsedPlaybackMethod(c *gin.Context) {
	tr := parseTimeRange(c)
	type Row struct {
		Method   string `json:"method"`
		Count    int    `json:"count"` // plays
		Clicks   int    `json:"clicks"`
		Duration int    `json:"duration"`
	}
	rows := []Row{}
	q, args := acts().inRange(tr).sql(`
		SELECT play_method AS method, ` + aggPlays + ` AS count, ` + aggClicks + ` AS clicks, ` + aggMinutes + ` AS duration
		FROM acts GROUP BY play_method ORDER BY count DESC, clicks DESC`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StatsFrontendHandler) GetMostUsedClients(c *gin.Context) {
	tr := parseTimeRange(c)
	type Row struct {
		Client   string `json:"client"`
		Count    int    `json:"count"` // plays
		Clicks   int    `json:"clicks"`
		Duration int    `json:"duration"`
	}
	rows := []Row{}
	q, args := acts().inRange(tr).where(`a."Client" IS NOT NULL AND a."Client" <> ''`).sql(`
		SELECT client, ` + aggPlays + ` AS count, ` + aggClicks + ` AS clicks, ` + aggMinutes + ` AS duration
		FROM acts GROUP BY client ORDER BY count DESC, clicks DESC LIMIT 10`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getUserStats?userId=   (all time)
// ---------------------------------------------------------------------------

type userStatRow struct {
	UserId         string  `json:"UserId"`
	UserName       string  `json:"UserName"`
	TotalPlays     int     `json:"TotalPlays"`
	TotalClicks    int     `json:"TotalClicks"`
	TotalWatchTime int     `json:"TotalWatchTime"`
	UniqueItems    int     `json:"UniqueItems"`
	LastSeen       *string `json:"LastSeen"`
	FirstSeen      *string `json:"FirstSeen"`
	MostUsedClient *string `json:"MostUsedClient"`
	MostUsedDevice *string `json:"MostUsedDevice"`
	FavoriteGenre  *string `json:"FavoriteGenre"` // genre with the most watch time
}

func (h *StatsFrontendHandler) queryUserStats(userId string) ([]userStatRow, error) {
	outer := `,
		agg AS (
		  SELECT user_id, ` + aggPlays + ` AS plays, ` + aggClicks + ` AS clicks, ` + aggMinutes + ` AS minutes,
		         COUNT(DISTINCT item_id) FILTER (WHERE is_view)::int AS unique_items,
		         MAX(started_at) AS last_seen, MIN(started_at) AS first_seen
		  FROM acts GROUP BY user_id
		),
		cl AS (
		  SELECT DISTINCT ON (user_id) user_id, client FROM acts
		  WHERE client IS NOT NULL AND client <> ''
		  GROUP BY user_id, client
		  ORDER BY user_id, COUNT(*) FILTER (WHERE is_view) DESC, COUNT(*) DESC, client
		),
		dv AS (
		  SELECT DISTINCT ON (user_id) user_id, device_name FROM acts
		  WHERE device_name IS NOT NULL AND device_name <> ''
		  GROUP BY user_id, device_name
		  ORDER BY user_id, COUNT(*) FILTER (WHERE is_view) DESC, COUNT(*) DESC, device_name
		),
		gn AS (
		  SELECT DISTINCT ON (user_id) user_id, genre FROM acts ` + genreLateral + `
		  WHERE acts.genres IS NOT NULL AND genre <> 'No Genre'
		  GROUP BY user_id, genre
		  ORDER BY user_id, SUM(duration) DESC, genre
		)
		SELECT
		  u."Id" AS "UserId",
		  COALESCE(u."Name", u."Id") AS "UserName",
		  COALESCE(agg.plays, 0) AS "TotalPlays",
		  COALESCE(agg.clicks, 0) AS "TotalClicks",
		  COALESCE(agg.minutes, 0) AS "TotalWatchTime",
		  COALESCE(agg.unique_items, 0) AS "UniqueItems",
		  COALESCE(` + statsql.ISO("agg.last_seen") + `, u."LastActivityDate", u."LastLoginDate") AS "LastSeen",
		  ` + statsql.ISO("agg.first_seen") + ` AS "FirstSeen",
		  cl.client AS "MostUsedClient",
		  dv.device_name AS "MostUsedDevice",
		  gn.genre AS "FavoriteGenre"
		FROM jf_users u
		LEFT JOIN agg ON agg.user_id = u."Id"
		LEFT JOIN cl  ON cl.user_id  = u."Id"
		LEFT JOIN dv  ON dv.user_id  = u."Id"
		LEFT JOIN gn  ON gn.user_id  = u."Id"`
	f := acts()
	var outerArgs []interface{}
	if userId != "" {
		f.forUser(userId)
		outer += ` WHERE u."Id" = ?`
		outerArgs = append(outerArgs, userId)
	}
	outer += ` ORDER BY "TotalPlays" DESC, "TotalWatchTime" DESC, "UserName"`
	q, args := f.sql(outer, outerArgs...)
	rows := []userStatRow{}
	err := h.db.Raw(q, args...).Scan(&rows).Error
	return rows, err
}

func (h *StatsFrontendHandler) GetUserStats(c *gin.Context) {
	userId := c.Query("userId")
	rows, err := h.queryUserStats(userId)
	if dbError(c, err) {
		return
	}
	if userId != "" {
		if len(rows) == 0 {
			c.JSON(http.StatusOK, userStatRow{UserId: userId})
			return
		}
		c.JSON(http.StatusOK, rows[0])
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getAllUserActivity / getUserActivity  (history lists: all clicks,
// flagged with IsView)
// ---------------------------------------------------------------------------

type activityRow struct {
	Id                   *string `json:"Id"`
	UserId               *string `json:"UserId"`
	UserName             *string `json:"UserName"`
	ItemId               *string `json:"ItemId"`
	NowPlayingItemName   *string `json:"NowPlayingItemName"`
	NowPlayingItemType   *string `json:"NowPlayingItemType"` // Movie | Episode | Audio
	SeriesName           *string `json:"SeriesName"`
	SeasonId             *string `json:"SeasonId"`
	EpisodeId            *string `json:"EpisodeId"`
	ParentId             *string `json:"ParentId"` // library id
	Client               *string `json:"Client"`
	DeviceName           *string `json:"DeviceName"`
	DeviceId             *string `json:"DeviceId"`
	ApplicationVersion   *string `json:"ApplicationVersion"`
	PlayMethod           *string `json:"PlayMethod"`
	IsPaused             *bool   `json:"IsPaused"`
	IsActive             bool    `json:"IsActive"`
	IsView               bool    `json:"IsView"`
	PlayDuration         *int64  `json:"PlayDuration"` // ticks
	ActivityDateInserted *string `json:"ActivityDateInserted"`
	StartedAt            *string `json:"StartedAt"`
	EndedAt              *string `json:"EndedAt"`
	RemoteEndPoint       *string `json:"RemoteEndPoint"`
}

func (h *StatsFrontendHandler) queryActivity(f *actFilter) ([]activityRow, error) {
	q, args := f.sql(`
		SELECT
		  acts.id AS "Id", acts.user_id AS "UserId", acts.user_name AS "UserName",
		  acts.item_id AS "ItemId",
		  p."NowPlayingItemName",
		  acts.media_type AS "NowPlayingItemType",
		  p."SeriesName", p."SeasonId", p."EpisodeId",
		  acts.library_id AS "ParentId",
		  p."Client", p."DeviceName", p."DeviceId", p."ApplicationVersion",
		  p."PlayMethod", p."IsPaused",
		  false AS "IsActive",
		  acts.is_view AS "IsView",
		  (acts.duration * 10000000)::bigint AS "PlayDuration",
		  p."ActivityDateInserted",
		  ` + statsql.ISO("acts.started_at") + ` AS "StartedAt",
		  ` + statsql.ISO("acts.ended_at") + ` AS "EndedAt",
		  p."RemoteEndPoint"
		FROM acts
		JOIN jf_playback_activity p ON p."Id" = acts.id
		ORDER BY acts.started_at DESC`)
	rows := []activityRow{}
	err := h.db.Raw(q, args...).Scan(&rows).Error
	return rows, err
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func toArgs(vals []string) []interface{} {
	out := make([]interface{}, len(vals))
	for i, v := range vals {
		out[i] = v
	}
	return out
}

func (h *StatsFrontendHandler) GetAllUserActivity(c *gin.Context) {
	f := acts()
	loc := statsql.Location()
	s := statsql.StartedAt("a")
	if v := c.Query("dateFrom"); v != "" {
		if t, err := time.ParseInLocation("2006-01-02", v, loc); err == nil {
			f.where(s+` >= ?`, t)
		}
	}
	if v := c.Query("dateTo"); v != "" {
		if t, err := time.ParseInLocation("2006-01-02", v, loc); err == nil {
			f.where(s+` < ?`, t.AddDate(0, 0, 1))
		}
	}
	if vals := splitCSV(c.Query("client")); len(vals) > 0 {
		f.where(`a."Client" IN (`+inPlaceholders(len(vals))+`)`, toArgs(vals)...)
	}
	if vals := splitCSV(c.Query("playMethod")); len(vals) > 0 {
		f.where(`a."PlayMethod" IN (`+inPlaceholders(len(vals))+`)`, toArgs(vals)...)
	}
	if vals := splitCSV(c.Query("mediaType")); len(vals) > 0 {
		var mts []string
		for _, v := range vals {
			if mt := mediaTypeParam(v); mt != "" {
				mts = append(mts, mt)
			}
		}
		if len(mts) == 0 {
			c.JSON(http.StatusOK, []activityRow{})
			return
		}
		f.where(`a."MediaType" IN (`+inPlaceholders(len(mts))+`)`, toArgs(mts)...)
	}
	if vals := splitCSV(c.Query("deviceName")); len(vals) > 0 {
		f.where(`a."DeviceName" IN (`+inPlaceholders(len(vals))+`)`, toArgs(vals)...)
	}
	if vals := splitCSV(c.Query("userName")); len(vals) > 0 {
		f.where(`COALESCE(u."Name", a."UserName") IN (`+inPlaceholders(len(vals))+`)`, toArgs(vals)...)
	}
	if v, err := strconv.Atoi(c.Query("durMin")); err == nil {
		f.where(`COALESCE(a."PlaybackDuration", 0) >= ?`, v*60)
	}
	if v, err := strconv.Atoi(c.Query("durMax")); err == nil {
		f.where(`COALESCE(a."PlaybackDuration", 0) <= ?`, v*60)
	}
	rows, err := h.queryActivity(f)
	if dbError(c, err) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StatsFrontendHandler) GetUserActivity(c *gin.Context) {
	userId := c.Query("userId")
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}
	rows, err := h.queryActivity(acts().forUser(userId))
	if dbError(c, err) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getUserActivityByDate?userId=   (all time, local days)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetUserActivityByDate(c *gin.Context) {
	userId := c.Query("userId")
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}
	type Row struct {
		Date     string `json:"date"`
		Count    int    `json:"count"` // plays
		Clicks   int    `json:"clicks"`
		Duration int    `json:"duration"`
	}
	rows := []Row{}
	q, args := acts().forUser(userId).sql(`
		SELECT TO_CHAR(local_ts::date, 'YYYY-MM-DD') AS date,
		       ` + aggPlays + ` AS count, ` + aggClicks + ` AS clicks, ` + aggMinutes + ` AS duration
		FROM acts GROUP BY local_ts::date ORDER BY local_ts::date`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraries
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraries(c *gin.Context) {
	type Row struct {
		Id             string  `json:"Id"`
		Name           string  `json:"Name"`
		CollectionType string  `json:"CollectionType"`
		ItemCount      int     `json:"ItemCount"`
		EpisodeCount   int     `json:"EpisodeCount"`
		SeasonCount    int     `json:"SeasonCount"`
		TrackCount     int     `json:"TrackCount"`
		TotalSize      int64   `json:"TotalSize"`
		TotalPlayCount int     `json:"TotalPlayCount"`
		TotalClicks    int     `json:"TotalClicks"`
		TotalWatchTime int     `json:"TotalWatchTime"`
		LastActivity   *string `json:"LastActivity"`
	}
	rows := []Row{}
	q, args := acts().sql(`,
		item_stats AS (
		  SELECT i."ParentId" AS "LibraryId",
		    COUNT(*) FILTER (WHERE i."Type" NOT IN ('Season', 'Folder'))::int AS "ItemCount",
		    COALESCE(SUM(ii."Size") FILTER (WHERE i."Type" NOT IN ('Season', 'Folder')), 0)::bigint AS "DirectSize"
		  FROM jf_library_items i
		  LEFT JOIN jf_item_info ii ON ii."Id" = i."Id"
		  WHERE i.archived = false
		  GROUP BY i."ParentId"
		),
		episode_stats AS (
		  SELECT series."ParentId" AS "LibraryId",
		    COUNT(DISTINCT e."Id")::int AS "EpisodeCount",
		    COALESCE(SUM(ii."Size"), 0)::bigint AS "EpisodeSize"
		  FROM jf_library_episodes e
		  JOIN jf_library_items series ON series."Id" = e."SeriesId" AND series.archived = false
		  LEFT JOIN jf_item_info ii ON ii."Id" = e."Id"
		  WHERE e.archived = false
		  GROUP BY series."ParentId"
		),
		season_stats AS (
		  SELECT series."ParentId" AS "LibraryId", COUNT(DISTINCT s."Id")::int AS "SeasonCount"
		  FROM jf_library_seasons s
		  JOIN jf_library_items series ON series."Id" = s."SeriesId" AND series.archived = false
		  WHERE s.archived = false
		  GROUP BY series."ParentId"
		),
		track_stats AS (
		  SELECT t."LibraryId", COUNT(*)::int AS "TrackCount", COALESCE(SUM(ii."Size"), 0)::bigint AS "TrackSize"
		  FROM jf_music_tracks t
		  LEFT JOIN jf_item_info ii ON ii."Id" = t."Id"
		  WHERE t.archived = false AND t."LibraryId" IS NOT NULL
		  GROUP BY t."LibraryId"
		),
		play_stats AS (
		  SELECT library_id, ` + aggPlays + ` AS plays, ` + aggClicks + ` AS clicks, ` + aggMinutes + ` AS minutes,
		         MAX(started_at) AS last_activity
		  FROM acts WHERE library_id IS NOT NULL GROUP BY library_id
		)
		SELECT
		  l."Id", l."Name",
		  COALESCE(l."CollectionType", l."Type", 'unknown') AS "CollectionType",
		  COALESCE(ist."ItemCount", 0) AS "ItemCount",
		  COALESCE(epst."EpisodeCount", 0) AS "EpisodeCount",
		  COALESCE(ssst."SeasonCount", 0) AS "SeasonCount",
		  COALESCE(tst."TrackCount", 0) AS "TrackCount",
		  COALESCE(ist."DirectSize", 0) + COALESCE(epst."EpisodeSize", 0) + COALESCE(tst."TrackSize", 0) AS "TotalSize",
		  COALESCE(pst.plays, 0) AS "TotalPlayCount",
		  COALESCE(pst.clicks, 0) AS "TotalClicks",
		  COALESCE(pst.minutes, 0) AS "TotalWatchTime",
		  ` + statsql.ISO("pst.last_activity") + ` AS "LastActivity"
		FROM jf_libraries l
		LEFT JOIN item_stats    ist  ON ist."LibraryId"  = l."Id"
		LEFT JOIN episode_stats epst ON epst."LibraryId" = l."Id"
		LEFT JOIN season_stats  ssst ON ssst."LibraryId" = l."Id"
		LEFT JOIN track_stats   tst  ON tst."LibraryId"  = l."Id"
		LEFT JOIN play_stats    pst  ON pst.library_id   = l."Id"
		WHERE l.archived = false
		ORDER BY l."Name"`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraryStats?libraryId=   (all time)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryStats(c *gin.Context) {
	libraryId := c.Query("libraryId")
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId is required"})
		return
	}

	var stat struct {
		Name           string
		TotalItems     int
		TotalPlayCount int
		TotalClicks    int
		TotalWatchTime int
	}
	q, args := acts().where(libraryIdExpr+` = ?`, libraryId).sql(`
		SELECT
		  (SELECT "Name" FROM jf_libraries WHERE "Id" = ?) AS "Name",
		  ((SELECT COUNT(*) FROM jf_library_items WHERE "ParentId" = ? AND archived = false AND "Type" = 'Movie')
		   + (SELECT COUNT(*) FROM jf_library_episodes e JOIN jf_library_items s ON s."Id" = e."SeriesId" AND s.archived = false
		      WHERE s."ParentId" = ? AND e.archived = false)
		   + (SELECT COUNT(*) FROM jf_music_tracks WHERE "LibraryId" = ? AND archived = false))::int AS "TotalItems",
		  (SELECT `+aggPlays+` FROM acts) AS "TotalPlayCount",
		  (SELECT `+aggClicks+` FROM acts) AS "TotalClicks",
		  (SELECT `+aggMinutes+` FROM acts) AS "TotalWatchTime"
	`, libraryId, libraryId, libraryId, libraryId)
	if dbError(c, h.db.Raw(q, args...).Scan(&stat).Error) {
		return
	}

	type TopItem struct {
		Id         string `json:"Id"`
		Name       string `json:"Name"`
		Type       string `json:"Type"`
		PlayCount  int    `json:"PlayCount"`
		ClickCount int    `json:"ClickCount"`
	}
	var topItems []TopItem
	q, args = acts().where(libraryIdExpr+` = ?`, libraryId).sql(`
		SELECT group_id AS "Id", COALESCE(MAX(group_name), '') AS "Name", display_type AS "Type",
		       ` + aggPlays + ` AS "PlayCount", ` + aggClicks + ` AS "ClickCount"
		FROM acts WHERE group_id IS NOT NULL
		GROUP BY group_id, display_type
		HAVING COUNT(*) FILTER (WHERE is_view) > 0
		ORDER BY "PlayCount" DESC, "Name" ASC
		LIMIT 1`)
	if dbError(c, h.db.Raw(q, args...).Scan(&topItems).Error) {
		return
	}

	var mostPlayed *TopItem
	if len(topItems) > 0 {
		mostPlayed = &topItems[0]
	}
	c.JSON(http.StatusOK, gin.H{
		"Name":           stat.Name,
		"TotalItems":     stat.TotalItems,
		"TotalPlayCount": stat.TotalPlayCount,
		"TotalClicks":    stat.TotalClicks,
		"TotalWatchTime": stat.TotalWatchTime,
		"MostPlayedItem": mostPlayed,
	})
}

// groupStatsCTE aggregates acts per group_id (movie / series / album) for one
// library. Appended after the acts CTE; takes no args (library filter is on
// the acts CTE).
const groupStatsCTE = `,
	pc AS (
	  SELECT group_id,
	         ` + aggPlays + ` AS play_count,
	         ` + aggClicks + ` AS click_count,
	         ` + aggSeconds + ` AS play_seconds,
	         MAX(started_at) AS last_played
	  FROM acts WHERE group_id IS NOT NULL
	  GROUP BY group_id
	)`

// ---------------------------------------------------------------------------
// GET /stats/getLibraryItems?libraryId=
// Series and albums aggregate the plays of their episodes / tracks.
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryItems(c *gin.Context) {
	libraryId := c.Query("libraryId")
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId is required"})
		return
	}

	type Row struct {
		Id                string   `json:"Id"`
		Name              string   `json:"Name"`
		Type              string   `json:"Type"`
		ProductionYear    *int     `json:"ProductionYear"`
		CommunityRating   *float64 `json:"CommunityRating"`
		Size              *int64   `json:"Size"`
		PlayCount         int      `json:"PlayCount"`
		ClickCount        int      `json:"ClickCount"`
		TotalWatchTime    int      `json:"TotalWatchTime"` // minutes
		LastPlayed        *string  `json:"LastPlayed"`
		SeriesName        *string  `json:"SeriesName"`
		IndexNumber       *int     `json:"IndexNumber"`
		ParentIndexNumber *int     `json:"ParentIndexNumber"`
	}
	rows := []Row{}
	q, args := acts().where(libraryIdExpr+` = ?`, libraryId).sql(groupStatsCTE+`
		SELECT
		  i."Id", i."Name", i."Type", i."ProductionYear", i."CommunityRating",
		  info."Size",
		  COALESCE(pc.play_count, 0) AS "PlayCount",
		  COALESCE(pc.click_count, 0) AS "ClickCount",
		  FLOOR(COALESCE(pc.play_seconds, 0) / 60.0)::int AS "TotalWatchTime",
		  `+statsql.ISO("pc.last_played")+` AS "LastPlayed",
		  NULL::text AS "SeriesName",
		  NULL::int AS "IndexNumber",
		  NULL::int AS "ParentIndexNumber"
		FROM jf_library_items i
		LEFT JOIN pc ON pc.group_id = i."Id"
		LEFT JOIN jf_item_info info ON info."Id" = i."Id"
		WHERE i."ParentId" = ?
		  AND i.archived = false
		  AND i."Type" NOT IN ('Season', 'Folder')
		ORDER BY "PlayCount" DESC, i."Name"`, libraryId)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getItemDetails?itemId=   (movie, episode, track, series or album)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetItemDetails(c *gin.Context) {
	itemId := c.Query("itemId")
	if itemId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "itemId is required"})
		return
	}

	type ItemInfo struct {
		Id              string          `json:"Id"`
		Name            string          `json:"Name"`
		Type            string          `json:"Type"`
		ProductionYear  *int            `json:"ProductionYear"`
		CommunityRating *float64        `json:"CommunityRating"`
		PremiereDate    *string         `json:"PremiereDate"`
		DateCreated     *string         `json:"DateCreated"`
		RunTimeTicks    *int64          `json:"RunTimeTicks"`
		Genres          json.RawMessage `json:"Genres"`
		ParentId        *string         `json:"ParentId"`
		Size            *int64          `json:"Size"`
		Path            *string         `json:"Path"`
		Bitrate         *int64          `json:"Bitrate"`
		AlbumId         *string         `json:"AlbumId"`
		AlbumName       *string         `json:"AlbumName"`
		Artist          *string         `json:"Artist"`
	}
	var item ItemInfo
	if dbError(c, h.db.Raw(`
		SELECT * FROM (
		  SELECT i."Id", i."Name", i."Type", i."ProductionYear", i."CommunityRating"::float8 AS "CommunityRating",
		         i."PremiereDate", i."DateCreated", i."RunTimeTicks", i."Genres", i."ParentId",
		         info."Size", info."Path", info."Bitrate",
		         NULL::text AS "AlbumId", NULL::text AS "AlbumName", NULL::text AS "Artist"
		  FROM jf_library_items i
		  LEFT JOIN jf_item_info info ON info."Id" = i."Id"
		  WHERE i."Id" = ?
		  UNION ALL
		  SELECT e."Id", e."Name", 'Episode', e."ProductionYear", e."CommunityRating"::float8,
		         e."PremiereDate", e."DateCreated", e."RunTimeTicks", s."Genres", s."ParentId",
		         info."Size", info."Path", info."Bitrate",
		         NULL::text, NULL::text, NULL::text
		  FROM jf_library_episodes e
		  LEFT JOIN jf_library_items s ON s."Id" = e."SeriesId"
		  LEFT JOIN jf_item_info info ON info."Id" = e."Id"
		  WHERE e."Id" = ?
		  UNION ALL
		  SELECT t."Id", t."Name", 'Audio', t."ProductionYear", NULL::float8,
		         NULL::text, t."DateCreated", t."RunTimeTicks", t."Genres", t."LibraryId",
		         NULL::bigint, NULL::text, NULL::bigint,
		         t."AlbumId", t."AlbumName", COALESCE(t."AlbumArtist", a."Name")
		  FROM jf_music_tracks t
		  LEFT JOIN jf_music_artists a ON a."Id" = t."ArtistId"
		  WHERE t."Id" = ?
		) sub LIMIT 1
	`, itemId, itemId, itemId).Scan(&item).Error) {
		return
	}

	// Rows for the item itself, or for the episodes/tracks of a series/album.
	match := func() *actFilter {
		return acts().where(`(a."ItemId" = ? OR CASE a."MediaType"
		    WHEN 'Episode' THEN COALESCE(cs."Id", ce."SeriesId", a."NowPlayingItemId")
		    WHEN 'Audio'   THEN COALESCE(cal."Id", ct."AlbumId", a."NowPlayingItemId")
		  END = ?)`, itemId, itemId)
	}

	var totals struct {
		TotalPlays     int
		TotalClicks    int
		TotalWatchTime int
		UniqueUsers    int
		LastWatched    *string
	}
	q, args := match().sql(`
		SELECT ` + aggPlays + ` AS "TotalPlays", ` + aggClicks + ` AS "TotalClicks",
		       ` + aggMinutes + ` AS "TotalWatchTime",
		       COUNT(DISTINCT user_id) FILTER (WHERE is_view)::int AS "UniqueUsers",
		       ` + statsql.ISO("MAX(started_at)") + ` AS "LastWatched"
		FROM acts`)
	if dbError(c, h.db.Raw(q, args...).Scan(&totals).Error) {
		return
	}

	type UserAgg struct {
		UserId         string  `json:"UserId"`
		UserName       string  `json:"UserName"`
		PlayCount      int     `json:"PlayCount"`
		ClickCount     int     `json:"ClickCount"`
		TotalWatchTime int     `json:"TotalWatchTime"`
		LastWatched    *string `json:"LastWatched"`
	}
	users := []UserAgg{}
	q, args = match().sql(`
		SELECT user_id AS "UserId", MAX(user_name) AS "UserName",
		       ` + aggPlays + ` AS "PlayCount", ` + aggClicks + ` AS "ClickCount",
		       ` + aggMinutes + ` AS "TotalWatchTime",
		       ` + statsql.ISO("MAX(started_at)") + ` AS "LastWatched"
		FROM acts WHERE user_id IS NOT NULL
		GROUP BY user_id
		ORDER BY "PlayCount" DESC, "TotalWatchTime" DESC`)
	if dbError(c, h.db.Raw(q, args...).Scan(&users).Error) {
		return
	}

	type HistEntry struct {
		Id                   *string `json:"Id"`
		UserId               *string `json:"UserId"`
		UserName             *string `json:"UserName"`
		ItemId               *string `json:"ItemId"`
		ItemName             *string `json:"ItemName"`
		Client               *string `json:"Client"`
		DeviceName           *string `json:"DeviceName"`
		PlayMethod           *string `json:"PlayMethod"`
		PlaybackDuration     int     `json:"PlaybackDuration"` // minutes
		IsView               bool    `json:"IsView"`
		ActivityDateInserted *string `json:"ActivityDateInserted"`
		StartedAt            *string `json:"StartedAt"`
		RemoteEndPoint       *string `json:"RemoteEndPoint"`
		IsActive             bool    `json:"IsActive"`
	}
	history := []HistEntry{}
	q, args = match().sql(`
		SELECT acts.id AS "Id", acts.user_id AS "UserId", acts.user_name AS "UserName",
		       acts.item_id AS "ItemId", acts.item_name AS "ItemName",
		       p."Client", p."DeviceName", p."PlayMethod",
		       FLOOR(acts.duration / 60.0)::int AS "PlaybackDuration",
		       acts.is_view AS "IsView",
		       p."ActivityDateInserted",
		       ` + statsql.ISO("acts.started_at") + ` AS "StartedAt",
		       p."RemoteEndPoint",
		       false AS "IsActive"
		FROM acts JOIN jf_playback_activity p ON p."Id" = acts.id
		ORDER BY acts.started_at DESC
		LIMIT 200`)
	if dbError(c, h.db.Raw(q, args...).Scan(&history).Error) {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item": item,
		"stats": gin.H{
			"TotalPlays":     totals.TotalPlays,
			"TotalClicks":    totals.TotalClicks,
			"TotalWatchTime": totals.TotalWatchTime,
			"UniqueUsers":    totals.UniqueUsers,
			"LastWatched":    totals.LastWatched,
			"IsActive":       false,
		},
		"users":   users,
		"history": history,
	})
}

// ---------------------------------------------------------------------------
// GET /stats/getGenreStats?libraryId=&userId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetGenreStats(c *gin.Context) {
	libraryId := c.Query("libraryId")
	userId := c.Query("userId")

	type Row struct {
		Genre     string `json:"Genre"`
		Count     int    `json:"Count"`
		PlayCount int    `json:"PlayCount"`
	}
	rows := []Row{}

	if userId != "" {
		f := acts().forUser(userId).where(`a."IsView"`)
		if libraryId != "" {
			f.where(libraryIdExpr+` = ?`, libraryId)
		}
		q, args := f.sql(`
			SELECT genre AS "Genre", COUNT(DISTINCT item_id)::int AS "Count", COUNT(*)::int AS "PlayCount"
			FROM acts ` + genreLateral + `
			WHERE acts.genres IS NOT NULL
			GROUP BY genre
			ORDER BY "PlayCount" DESC, genre ASC
			LIMIT 100`)
		if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
			return
		}
	} else if libraryId != "" {
		if dbError(c, h.db.Raw(`
			SELECT genre AS "Genre", COUNT(*)::int AS "Count", 0::int AS "PlayCount"
			FROM (
			  SELECT "Genres" FROM jf_library_items
			  WHERE "ParentId" = ? AND archived = false AND "Type" NOT IN ('Season', 'Folder')
			  UNION ALL
			  SELECT "Genres" FROM jf_music_tracks
			  WHERE "LibraryId" = ? AND archived = false
			) combined
			CROSS JOIN LATERAL jsonb_array_elements_text(
			  CASE WHEN jsonb_array_length(COALESCE(combined."Genres", '[]'::jsonb)) = 0 THEN '["No Genre"]'::jsonb
			       ELSE combined."Genres" END
			) AS genre
			GROUP BY genre
			ORDER BY "Count" DESC, genre ASC
		`, libraryId, libraryId).Scan(&rows).Error) {
			return
		}
	}
	c.JSON(http.StatusOK, rows)
}

// GET /stats/getUserGenreStats?userId=
func (h *StatsFrontendHandler) GetUserGenreStats(c *gin.Context) {
	h.GetGenreStats(c)
}

// ---------------------------------------------------------------------------
// Music library listings (plays per track = views whose ItemId is the track)
// ---------------------------------------------------------------------------

const trackPlaysJoin = `
	LEFT JOIN (
	  SELECT "ItemId" AS track_id, COUNT(*)::int AS play_count
	  FROM jf_playback_activity
	  WHERE "MediaType" = 'Audio' AND "IsView"
	  GROUP BY "ItemId"
	) pc ON pc.track_id = t."Id"`

func (h *StatsFrontendHandler) GetLibraryTracks(c *gin.Context) {
	libraryId := c.Query("libraryId")
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId is required"})
		return
	}
	type Row struct {
		Id           string  `json:"Id"`
		Name         string  `json:"Name"`
		Artist       *string `json:"Artist"`
		AlbumName    *string `json:"AlbumName"`
		AlbumId      *string `json:"AlbumId"`
		IndexNumber  *int    `json:"IndexNumber"`
		DiscNumber   *int    `json:"DiscNumber"`
		RunTimeTicks *int64  `json:"RunTimeTicks"`
		PlayCount    int     `json:"PlayCount"`
	}
	rows := []Row{}
	if dbError(c, h.db.Raw(`
		SELECT t."Id", t."Name", t."AlbumArtist" AS "Artist", t."AlbumName", t."AlbumId",
		       t."IndexNumber", t."DiscNumber", t."RunTimeTicks",
		       COALESCE(pc.play_count, 0)::int AS "PlayCount"
		FROM jf_music_tracks t`+trackPlaysJoin+`
		WHERE t."LibraryId" = ? AND t.archived = false
		ORDER BY COALESCE(t."AlbumArtist", '') ASC, COALESCE(t."AlbumName", '') ASC,
		         t."DiscNumber" NULLS LAST, t."IndexNumber" NULLS LAST, t."Name" ASC
	`, libraryId).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StatsFrontendHandler) GetLibraryAlbums(c *gin.Context) {
	libraryId := c.Query("libraryId")
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId is required"})
		return
	}
	type Row struct {
		Id               string  `json:"Id"`
		Name             string  `json:"Name"`
		AlbumArtist      *string `json:"AlbumArtist"`
		ArtistId         *string `json:"ArtistId"`
		ProductionYear   *int    `json:"ProductionYear"`
		ImageTagsPrimary *string `json:"ImageTagsPrimary"`
		TrackCount       int     `json:"TrackCount"`
		PlayCount        int     `json:"PlayCount"`
	}
	rows := []Row{}
	if dbError(c, h.db.Raw(`
		SELECT album."Id", album."Name", album."AlbumArtist", album."ArtistId", album."ProductionYear",
		       album."ImageTagsPrimary",
		       COUNT(t."Id")::int AS "TrackCount",
		       COALESCE(SUM(pc.play_count), 0)::int AS "PlayCount"
		FROM jf_library_items album
		LEFT JOIN jf_music_tracks t ON t."AlbumId" = album."Id" AND t.archived = false`+trackPlaysJoin+`
		WHERE album."ParentId" = ? AND album.archived = false AND album."Type" = 'MusicAlbum'
		GROUP BY album."Id", album."Name", album."AlbumArtist", album."ArtistId", album."ProductionYear", album."ImageTagsPrimary"
		ORDER BY album."Name"
	`, libraryId).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StatsFrontendHandler) GetLibraryArtists(c *gin.Context) {
	libraryId := c.Query("libraryId")
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId is required"})
		return
	}
	type Row struct {
		Id               string  `json:"Id"`
		Name             string  `json:"Name"`
		ImageTagsPrimary *string `json:"ImageTagsPrimary"`
		AlbumCount       int     `json:"AlbumCount"`
		TrackCount       int     `json:"TrackCount"`
		PlayCount        int     `json:"PlayCount"`
	}
	rows := []Row{}
	if dbError(c, h.db.Raw(`
		SELECT COALESCE(t."ArtistId", '') AS "Id", COALESCE(t."AlbumArtist", 'Unknown') AS "Name",
		       ar."ImageTagsPrimary",
		       COUNT(DISTINCT t."AlbumId")::int AS "AlbumCount",
		       COUNT(t."Id")::int AS "TrackCount",
		       COALESCE(SUM(pc.play_count), 0)::int AS "PlayCount"
		FROM jf_music_tracks t
		LEFT JOIN jf_music_artists ar ON ar."Id" = t."ArtistId"`+trackPlaysJoin+`
		WHERE t."LibraryId" = ? AND t.archived = false
		GROUP BY COALESCE(t."ArtistId", ''), COALESCE(t."AlbumArtist", 'Unknown'), ar."ImageTagsPrimary"
		ORDER BY COALESCE(t."AlbumArtist", 'Unknown')
	`, libraryId).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StatsFrontendHandler) GetArtistAlbums(c *gin.Context) {
	libraryId := c.Query("libraryId")
	artistId := c.Query("artistId")
	if libraryId == "" || artistId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId and artistId are required"})
		return
	}
	type Row struct {
		Id               string  `json:"Id"`
		Name             string  `json:"Name"`
		AlbumArtist      *string `json:"AlbumArtist"`
		ArtistId         *string `json:"ArtistId"`
		ProductionYear   *int    `json:"ProductionYear"`
		ImageTagsPrimary *string `json:"ImageTagsPrimary"`
		TrackCount       int     `json:"TrackCount"`
		PlayCount        int     `json:"PlayCount"`
	}
	rows := []Row{}
	if dbError(c, h.db.Raw(`
		SELECT t."AlbumId" AS "Id", COALESCE(album."Name", t."AlbumName") AS "Name",
		       t."AlbumArtist", t."ArtistId", album."ProductionYear", album."ImageTagsPrimary",
		       COUNT(t."Id")::int AS "TrackCount",
		       COALESCE(SUM(pc.play_count), 0)::int AS "PlayCount"
		FROM jf_music_tracks t
		LEFT JOIN jf_library_items album ON album."Id" = t."AlbumId" AND album.archived = false`+trackPlaysJoin+`
		WHERE t."ArtistId" = ? AND t."LibraryId" = ? AND t.archived = false AND t."AlbumId" IS NOT NULL
		GROUP BY t."AlbumId", album."Name", t."AlbumName", t."AlbumArtist", t."ArtistId", album."ProductionYear", album."ImageTagsPrimary"
		ORDER BY COALESCE(album."Name", t."AlbumName")
	`, artistId, libraryId).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StatsFrontendHandler) GetAlbumTracks(c *gin.Context) {
	albumId := c.Query("albumId")
	if albumId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "albumId is required"})
		return
	}
	type Row struct {
		Id           string  `json:"Id"`
		Name         string  `json:"Name"`
		IndexNumber  *int    `json:"IndexNumber"`
		DiscNumber   *int    `json:"DiscNumber"`
		RunTimeTicks *int64  `json:"RunTimeTicks"`
		AlbumId      *string `json:"AlbumId"`
		AlbumName    *string `json:"AlbumName"`
		Artist       *string `json:"Artist"`
		PlayCount    int     `json:"PlayCount"`
	}
	rows := []Row{}
	if dbError(c, h.db.Raw(`
		SELECT t."Id", t."Name", t."IndexNumber", t."DiscNumber", t."RunTimeTicks",
		       t."AlbumId", t."AlbumName", t."AlbumArtist" AS "Artist",
		       COALESCE(pc.play_count, 0)::int AS "PlayCount"
		FROM jf_music_tracks t`+trackPlaysJoin+`
		WHERE t."AlbumId" = ? AND t.archived = false
		ORDER BY t."DiscNumber" NULLS LAST, t."IndexNumber" NULLS LAST, t."Name"
	`, albumId).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getActivityTimeline?year=&month=   (month in display time zone)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetActivityTimeline(c *gin.Context) {
	now := time.Now().In(statsql.Location())
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(now.Year())))
	month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(now.Month()))))
	if year == 0 {
		year = now.Year()
	}
	if month < 1 || month > 12 {
		month = int(now.Month())
	}

	type Row struct {
		Id         *string `json:"Id"`
		UserId     *string `json:"UserId"`
		UserName   *string `json:"UserName"`
		ItemId     *string `json:"ItemId"`
		ItemName   *string `json:"ItemName"`
		StartTime  *string `json:"StartTime"`
		EndTime    *string `json:"EndTime"`
		Duration   int     `json:"Duration"` // seconds actually watched
		IsView     bool    `json:"IsView"`
		Client     *string `json:"Client"`
		PlayMethod *string `json:"PlayMethod"`
	}
	rows := []Row{}
	local := statsql.LocalTS("a")
	q, args := acts().where(local+` >= make_timestamp(?, ?, 1, 0, 0, 0) AND `+local+` < make_timestamp(?, ?, 1, 0, 0, 0) + INTERVAL '1 month'`,
		year, month, year, month).sql(`
		SELECT id AS "Id", user_id AS "UserId", user_name AS "UserName",
		       item_id AS "ItemId", COALESCE(now_playing_item_name, item_name) AS "ItemName",
		       ` + statsql.ISO("started_at") + ` AS "StartTime",
		       ` + statsql.ISO("ended_at") + ` AS "EndTime",
		       duration::int AS "Duration",
		       is_view AS "IsView",
		       client AS "Client", play_method AS "PlayMethod"
		FROM acts
		ORDER BY started_at DESC`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// POST /stats/getMostViewedLibraries  body: {from, to}
// ---------------------------------------------------------------------------

type rangeBody struct {
	From *string `json:"from"`
	To   *string `json:"to"`
}

func (b rangeBody) timeRange() TimeRange {
	return parseRange(strOrEmpty(b.From), strOrEmpty(b.To))
}

func (h *StatsFrontendHandler) GetMostViewedLibraries(c *gin.Context) {
	var body rangeBody
	_ = c.ShouldBindJSON(&body)

	type Row struct {
		Id     string `json:"Id"`
		Name   string `json:"Name"`
		Count  int    `json:"Count"` // plays
		Clicks int    `json:"Clicks"`
	}
	rows := []Row{}
	q, args := acts().inRange(body.timeRange()).sql(`,
		ls AS (
		  SELECT library_id, ` + aggPlays + ` AS plays, ` + aggClicks + ` AS clicks
		  FROM acts WHERE library_id IS NOT NULL GROUP BY library_id
		)
		SELECT l."Id", l."Name", COALESCE(ls.plays, 0) AS "Count", COALESCE(ls.clicks, 0) AS "Clicks"
		FROM jf_libraries l
		LEFT JOIN ls ON ls.library_id = l."Id"
		WHERE l.archived = false
		ORDER BY "Count" DESC, l."Name"`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// POST /stats/getLibraryLastPlayed  body: {libraryid}   (last 15 views)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryLastPlayed(c *gin.Context) {
	var body struct {
		LibraryId string `json:"libraryid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.LibraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}

	type Row struct {
		Id                   *string `json:"Id"`
		UserId               *string `json:"UserId"`
		UserName             *string `json:"UserName"`
		ItemId               *string `json:"ItemId"`
		ItemName             *string `json:"ItemName"`
		NowPlayingItemId     *string `json:"NowPlayingItemId"`
		NowPlayingItemName   *string `json:"NowPlayingItemName"`
		SeriesName           *string `json:"SeriesName"`
		Client               *string `json:"Client"`
		PlayMethod           *string `json:"PlayMethod"`
		ActivityDateInserted *string `json:"ActivityDateInserted"`
		StartedAt            *string `json:"StartedAt"`
		PlaybackDuration     int     `json:"PlaybackDuration"` // minutes
		ItemType             *string `json:"ItemType"`         // Movie | Episode | Audio
	}
	rows := []Row{}
	q, args := acts().where(`a."IsView"`).where(libraryIdExpr+` = ?`, body.LibraryId).sql(`
		SELECT acts.id AS "Id", acts.user_id AS "UserId", acts.user_name AS "UserName",
		       acts.item_id AS "ItemId", acts.item_name AS "ItemName",
		       acts.now_playing_item_id AS "NowPlayingItemId", acts.now_playing_item_name AS "NowPlayingItemName",
		       acts.series_name AS "SeriesName", acts.client AS "Client", acts.play_method AS "PlayMethod",
		       p."ActivityDateInserted",
		       ` + statsql.ISO("acts.started_at") + ` AS "StartedAt",
		       FLOOR(acts.duration / 60.0)::int AS "PlaybackDuration",
		       acts.media_type AS "ItemType"
		FROM acts JOIN jf_playback_activity p ON p."Id" = acts.id
		ORDER BY acts.started_at DESC
		LIMIT 15`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// POST /stats/getGlobalUserStats  body: {userid, hours?}   (default last 24 h)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetGlobalUserStats(c *gin.Context) {
	var body struct {
		UserId string `json:"userid"`
		Hours  *int   `json:"hours"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.UserId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userid is required"})
		return
	}
	hours := 24
	if body.Hours != nil && *body.Hours > 0 {
		hours = *body.Hours
	}

	var stat struct {
		TotalPlays            int   `json:"TotalPlays"`
		Plays                 int   `json:"Plays"` // alias of TotalPlays (used by UserDetail)
		TotalClicks           int   `json:"TotalClicks"`
		TotalWatchTime        int   `json:"TotalWatchTime"`          // minutes
		TotalPlaybackDuration int64 `json:"total_playback_duration"` // seconds
		UniqueItems           int   `json:"UniqueItems"`
	}
	q, args := acts().forUser(body.UserId).
		where(statsql.StartedAt("a")+` >= NOW() - MAKE_INTERVAL(hours => ?)`, hours).sql(`
		SELECT ` + aggPlays + ` AS "TotalPlays", ` + aggPlays + ` AS "Plays", ` + aggClicks + ` AS "TotalClicks",
		       ` + aggMinutes + ` AS "TotalWatchTime", ` + aggSeconds + ` AS "TotalPlaybackDuration",
		       COUNT(DISTINCT item_id) FILTER (WHERE is_view)::int AS "UniqueItems"
		FROM acts`)
	if dbError(c, h.db.Raw(q, args...).Scan(&stat).Error) {
		return
	}
	c.JSON(http.StatusOK, stat)
}

// ---------------------------------------------------------------------------
// POST /stats/getUserLastPlayed  body: {userid}   (last 15 views)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetUserLastPlayed(c *gin.Context) {
	var body struct {
		UserId string `json:"userid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.UserId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userid is required"})
		return
	}

	var rows []map[string]interface{}
	if dbError(c, h.db.Raw(`
		SELECT a.* FROM jf_playback_activity a
		WHERE a."UserId" = ? AND a."IsView" AND `+statsql.MediaFilter("a")+`
		ORDER BY `+statsql.StartedAt("a")+` DESC
		LIMIT 15
	`, body.UserId).Scan(&rows).Error) {
		return
	}
	if rows == nil {
		rows = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraryCardStats  (all libraries)
// POST /stats/getLibraryCardStats  body: {libraryid}
// ---------------------------------------------------------------------------

type libraryCardRow struct {
	Id                   string  `json:"Id"`
	Name                 string  `json:"Name"`
	CollectionType       string  `json:"CollectionType"`
	ItemCount            int     `json:"ItemCount"`
	ActivityDateInserted *string `json:"ActivityDateInserted"` // last playback start (RFC 3339)
	LastActivity         *string `json:"LastActivity"`         // interval since then
}

func (h *StatsFrontendHandler) queryLibraryCards(libraryId string) ([]libraryCardRow, error) {
	outer := `,
		last AS (
		  SELECT library_id, MAX(started_at) AS ts FROM acts
		  WHERE library_id IS NOT NULL GROUP BY library_id
		)
		SELECT l."Id", l."Name",
		  COALESCE(l."CollectionType", l."Type", 'unknown') AS "CollectionType",
		  COUNT(DISTINCT i."Id") FILTER (WHERE i."Type" NOT IN ('Season', 'Folder', 'MusicAlbum'))::int AS "ItemCount",
		  ` + statsql.ISO("last.ts") + ` AS "ActivityDateInserted",
		  (now() - last.ts)::text AS "LastActivity"
		FROM jf_libraries l
		LEFT JOIN jf_library_items i ON i."ParentId" = l."Id" AND i.archived = false
		LEFT JOIN last ON last.library_id = l."Id"
		WHERE l.archived = false`
	f := acts()
	var outerArgs []interface{}
	if libraryId != "" {
		f.where(libraryIdExpr+` = ?`, libraryId)
		outer += ` AND l."Id" = ?`
		outerArgs = append(outerArgs, libraryId)
	}
	outer += ` GROUP BY l."Id", l."Name", l."CollectionType", l."Type", last.ts ORDER BY l."Name"`
	q, args := f.sql(outer, outerArgs...)
	rows := []libraryCardRow{}
	err := h.db.Raw(q, args...).Scan(&rows).Error
	return rows, err
}

func (h *StatsFrontendHandler) GetLibraryCardStatsGET(c *gin.Context) {
	rows, err := h.queryLibraryCards("")
	if dbError(c, err) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StatsFrontendHandler) GetLibraryCardStatsPOST(c *gin.Context) {
	var body struct {
		LibraryId string `json:"libraryid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.LibraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}
	rows, err := h.queryLibraryCards(body.LibraryId)
	if dbError(c, err) {
		return
	}
	if len(rows) == 0 {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, rows[0])
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraryMetadata / getLibraryOverview (catalog only)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryMetadata(c *gin.Context) {
	type Row struct {
		Id             string `json:"Id"`
		Name           string `json:"Name"`
		CollectionType string `json:"CollectionType"`
		ItemCount      int    `json:"ItemCount"`
	}
	rows := []Row{}
	if dbError(c, h.db.Raw(`
		SELECT l."Id", l."Name",
		  COALESCE(l."CollectionType", l."Type", 'unknown') AS "CollectionType",
		  COUNT(DISTINCT i."Id") FILTER (WHERE i.archived = false AND i."Type" NOT IN ('Season', 'Folder'))::int AS "ItemCount"
		FROM jf_libraries l
		LEFT JOIN jf_library_items i ON i."ParentId" = l."Id"
		WHERE l.archived = false
		GROUP BY l."Id", l."Name", l."CollectionType", l."Type"
		ORDER BY l."Name"
	`).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StatsFrontendHandler) GetLibraryOverview(c *gin.Context) {
	type Row struct {
		Id             string `json:"Id"`
		Name           string `json:"Name"`
		CollectionType string `json:"CollectionType"`
		TotalItems     int    `json:"TotalItems"`
	}
	rows := []Row{}
	if dbError(c, h.db.Raw(`
		SELECT l."Id", l."Name",
		  COALESCE(l."CollectionType", l."Type", 'unknown') AS "CollectionType",
		  COUNT(i."Id") FILTER (WHERE i."Type" NOT IN ('Season', 'Folder'))::int AS "TotalItems"
		FROM jf_libraries l
		LEFT JOIN jf_library_items i ON i."ParentId" = l."Id" AND i.archived = false
		WHERE l.archived = false
		GROUP BY l."Id", l."Name", l."CollectionType", l."Type"
		ORDER BY l."Name"
	`).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getViewsByLibraryType?from=&to=
// Keys: Movie, Series (episodes), Music (tracks); "Audio" is kept as an alias
// of Music and "Other" is always 0 (other media types are not counted).
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetViewsByLibraryType(c *gin.Context) {
	tr := parseTimeRange(c)
	type Row struct {
		MediaType string
		Count     int
	}
	var rows []Row
	q, args := acts().inRange(tr).sql(`
		SELECT media_type AS "MediaType", ` + aggPlays + ` AS "Count" FROM acts GROUP BY media_type`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	result := map[string]int{"Movie": 0, "Series": 0, "Music": 0, "Audio": 0, "Other": 0}
	for _, r := range rows {
		switch r.MediaType {
		case "Movie":
			result["Movie"] += r.Count
		case "Episode":
			result["Series"] += r.Count
		case "Audio":
			result["Music"] += r.Count
			result["Audio"] += r.Count
		}
	}
	c.JSON(http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// GET /stats/getGenreUserStats?userid=&type=Movie|Series|Episode|Audio|Music&size=10&page=1
// GET /stats/getGenreLibraryStats?libraryid=&size=50&page=1
// duration = watch time (s, all clicks); plays = views. Genres of the
// series / album / track are used for episodes / tracks.
// ---------------------------------------------------------------------------

type genreStatRow struct {
	Genre    string `json:"genre"`
	Duration int64  `json:"duration"`
	Plays    int    `json:"plays"`
}

func (h *StatsFrontendHandler) genrePage(f *actFilter, orderBy string, size, offset int) (int, []genreStatRow, error) {
	gcte := `,
		g AS (
		  SELECT genre, ` + aggSeconds + ` AS duration, ` + aggPlays + ` AS plays
		  FROM acts ` + genreLateral + `
		  WHERE acts.genres IS NOT NULL
		  GROUP BY genre
		  HAVING COUNT(*) FILTER (WHERE is_view) > 0
		)`
	var total int
	q, args := f.sql(gcte + ` SELECT COUNT(*) FROM g`)
	if err := h.db.Raw(q, args...).Scan(&total).Error; err != nil {
		return 0, nil, err
	}
	rows := []genreStatRow{}
	q, args = f.sql(gcte+` SELECT genre, duration, plays FROM g ORDER BY `+orderBy+` LIMIT ? OFFSET ?`, size, offset)
	err := h.db.Raw(q, args...).Scan(&rows).Error
	return total, rows, err
}

func (h *StatsFrontendHandler) GetGenreUserStats(c *gin.Context) {
	userId := c.Query("userid")
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userid is required"})
		return
	}
	size := parseDays(c.DefaultQuery("size", "10"), 10)
	page := parseDays(c.DefaultQuery("page", "1"), 1)

	f := acts().forUser(userId)
	if t := c.Query("type"); t != "" {
		mt := mediaTypeParam(t)
		if mt == "" {
			c.JSON(http.StatusOK, paginate(0, size, page, []genreStatRow{}))
			return
		}
		f.where(`a."MediaType" = ?`, mt)
	}
	total, rows, err := h.genrePage(f, `plays DESC, duration DESC, genre`, size, (page-1)*size)
	if dbError(c, err) {
		return
	}
	c.JSON(http.StatusOK, paginate(total, size, page, rows))
}

func (h *StatsFrontendHandler) GetGenreLibraryStats(c *gin.Context) {
	libraryId := c.Query("libraryid")
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}
	size := parseDays(c.DefaultQuery("size", "50"), 50)
	page := parseDays(c.DefaultQuery("page", "1"), 1)

	total, rows, err := h.genrePage(acts().where(libraryIdExpr+` = ?`, libraryId), `genre ASC`, size, (page-1)*size)
	if dbError(c, err) {
		return
	}
	c.JSON(http.StatusOK, paginate(total, size, page, rows))
}

// ---------------------------------------------------------------------------
// GET /stats/getPlaybackActivity?size=50&page=1&search=&sort=&desc=true
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetPlaybackActivity(c *gin.Context) {
	size := parseDays(c.DefaultQuery("size", "50"), 50)
	page := parseDays(c.DefaultQuery("page", "1"), 1)
	offset := (page - 1) * size
	search := c.DefaultQuery("search", "")
	sortCol := c.DefaultQuery("sort", "ActivityDateInserted")
	isDesc := c.DefaultQuery("desc", "true") != "false"

	allowedSort := map[string]string{
		"ActivityDateInserted": statsql.StartedAt("a"),
		"StartedAt":            statsql.StartedAt("a"),
		"UserName":             `a."UserName"`,
		"NowPlayingItemName":   `a."NowPlayingItemName"`,
		"PlaybackDuration":     `a."PlaybackDuration"`,
		"Client":               `a."Client"`,
		"PlayMethod":           `a."PlayMethod"`,
	}
	sortExpr, ok := allowedSort[sortCol]
	if !ok {
		sortExpr = statsql.StartedAt("a")
	}
	order := "DESC"
	if !isDesc {
		order = "ASC"
	}

	where := ` WHERE ` + statsql.MediaFilter("a")
	var args []interface{}
	if search != "" {
		where += ` AND LOWER(COALESCE(NULLIF(a."SeriesName", ''), a."NowPlayingItemName")) LIKE ?`
		args = append(args, "%"+strings.ToLower(search)+"%")
	}

	var total int
	if dbError(c, h.db.Raw(`SELECT COUNT(*) FROM jf_playback_activity a`+where, args...).Scan(&total).Error) {
		return
	}
	var rows []map[string]interface{}
	q := `SELECT a.* FROM jf_playback_activity a` + where +
		fmt.Sprintf(` ORDER BY %s %s NULLS LAST LIMIT ? OFFSET ?`, sortExpr, order)
	if dbError(c, h.db.Raw(q, append(args, size, offset)...).Scan(&rows).Error) {
		return
	}
	if rows == nil {
		rows = []map[string]interface{}{}
	}

	pr := paginate(total, size, page, rows)
	c.JSON(http.StatusOK, gin.H{
		"current_page": pr.CurrentPage,
		"pages":        pr.Pages,
		"size":         pr.Size,
		"sort":         sortCol,
		"desc":         isDesc,
		"results":      pr.Results,
	})
}

// ---------------------------------------------------------------------------
// POST /stats/getPlaybackMethodStats  body: {from, to}
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetPlaybackMethodStats(c *gin.Context) {
	var body rangeBody
	_ = c.ShouldBindJSON(&body)

	type Row struct {
		Name   *string `json:"Name"`
		Count  int     `json:"Count"` // plays
		Clicks int     `json:"Clicks"`
	}
	rows := []Row{}
	q, args := acts().inRange(body.timeRange()).sql(`
		SELECT play_method AS "Name", ` + aggPlays + ` AS "Count", ` + aggClicks + ` AS "Clicks"
		FROM acts GROUP BY play_method ORDER BY "Count" DESC, "Clicks" DESC`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// POST /stats/getLibraryItemsWithStats  body: {libraryid}
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryItemsWithStats(c *gin.Context) {
	var body struct {
		LibraryId string `json:"libraryid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.LibraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}

	size := parseDays(c.DefaultQuery("size", "999999999"), 999999999)
	page := parseDays(c.DefaultQuery("page", "1"), 1)
	offset := (page - 1) * size
	search := c.DefaultQuery("search", "")
	sortParam := c.DefaultQuery("sort", "Date")
	isDesc := c.DefaultQuery("desc", "true") != "false"

	sortMap := map[string]string{
		"Date":      `i."DateCreated"`,
		"Views":     `pc.play_count`,
		"Clicks":    `pc.click_count`,
		"Size":      `info."Size"`,
		"WatchTime": `pc.play_seconds`,
		"Title":     `i."Name"`,
	}
	sortExpr, ok := sortMap[sortParam]
	if !ok {
		sortExpr = `i."DateCreated"`
	}
	order := "DESC"
	if !isDesc {
		order = "ASC"
	}

	baseFrom := `
		FROM jf_library_items i
		LEFT JOIN pc ON pc.group_id = i."Id"
		LEFT JOIN jf_item_info info ON info."Id" = i."Id"
		WHERE i."ParentId" = ?
		  AND i.archived = false
		  AND i."Type" NOT IN ('Season', 'Folder')`
	outerArgs := []interface{}{body.LibraryId}
	if search != "" {
		baseFrom += ` AND LOWER(i."Name") LIKE ?`
		outerArgs = append(outerArgs, "%"+strings.ToLower(search)+"%")
	}
	f := acts().where(libraryIdExpr+` = ?`, body.LibraryId)

	var total int
	q, args := f.sql(groupStatsCTE+` SELECT COUNT(*) `+baseFrom, outerArgs...)
	if dbError(c, h.db.Raw(q, args...).Scan(&total).Error) {
		return
	}

	type Row struct {
		Id              string   `json:"Id"`
		Name            string   `json:"Name"`
		Type            string   `json:"Type"`
		ProductionYear  *int     `json:"ProductionYear"`
		CommunityRating *float64 `json:"CommunityRating"`
		DateCreated     *string  `json:"DateCreated"`
		Size            *int64   `json:"Size"`
		TimesPlayed     int      `json:"times_played"`
		TimesClicked    int      `json:"times_clicked"`
		TotalPlayTime   int64    `json:"total_play_time"` // seconds
	}
	rows := []Row{}
	q, args = f.sql(groupStatsCTE+`
		SELECT i."Id", i."Name", i."Type", i."ProductionYear", i."CommunityRating", i."DateCreated", info."Size",
		       COALESCE(pc.play_count, 0) AS times_played,
		       COALESCE(pc.click_count, 0) AS times_clicked,
		       COALESCE(pc.play_seconds, 0) AS total_play_time
		`+baseFrom+fmt.Sprintf(` ORDER BY %s %s NULLS LAST, i."Name" LIMIT ? OFFSET ?`, sortExpr, order),
		append(outerArgs, size, offset)...)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}

	pr := paginate(total, size, page, rows)
	c.JSON(http.StatusOK, gin.H{
		"current_page": pr.CurrentPage,
		"pages":        pr.Pages,
		"size":         pr.Size,
		"sort":         sortParam,
		"desc":         isDesc,
		"results":      pr.Results,
	})
}

// ---------------------------------------------------------------------------
// POST /stats/getLibraryItemsPlayMethodStats  body: {libraryid, startDate?, endDate?, hours?}
// Concurrent views per hour (display time zone), split by play method.
// ---------------------------------------------------------------------------

type playMethodRecord struct {
	StartTime  time.Time
	EndTime    time.Time
	PlayMethod string
}

type hourCount struct {
	Hour        time.Time
	Transcodes  int
	DirectPlays int
}

// countOverlapsPerHour counts, for each local clock hour, the views that were
// running during (or within an hour of) that hour. Result is sorted by time.
func countOverlapsPerHour(records []playMethodRecord, loc *time.Location) []hourCount {
	counts := map[time.Time]*hourCount{}
	for _, r := range records {
		start := r.StartTime.Add(-time.Hour).In(loc)
		end := r.EndTime.Add(time.Hour)
		t := time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), 0, 0, 0, loc)
		for ; t.Before(end); t = t.Add(time.Hour) {
			hc, ok := counts[t]
			if !ok {
				hc = &hourCount{Hour: t}
				counts[t] = hc
			}
			if r.PlayMethod == "Transcode" {
				hc.Transcodes++
			} else {
				hc.DirectPlays++
			}
		}
	}
	out := make([]hourCount, 0, len(counts))
	for _, v := range counts {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Hour.Before(out[j].Hour) })
	return out
}

func (h *StatsFrontendHandler) GetLibraryItemsPlayMethodStats(c *gin.Context) {
	var body struct {
		LibraryId string  `json:"libraryid"`
		StartDate *string `json:"startDate"`
		EndDate   *string `json:"endDate"`
		Hours     *int    `json:"hours"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.LibraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}

	loc := statsql.Location()
	now := time.Now()
	f := acts().where(`a."IsView"`).where(libraryIdExpr+` = ?`, body.LibraryId)

	hasTimeFilter := (body.Hours != nil && *body.Hours > 0) ||
		(body.StartDate != nil && *body.StartDate != "") ||
		(body.EndDate != nil && *body.EndDate != "")
	if hasTimeFilter {
		hours := 24
		if body.Hours != nil && *body.Hours > 0 {
			hours = *body.Hours
		}
		startTime := now.Add(-time.Duration(hours) * time.Hour)
		endTime := now
		if body.StartDate != nil && *body.StartDate != "" {
			if t, err := time.ParseInLocation("2006-01-02", *body.StartDate, loc); err == nil {
				startTime = t
			}
		}
		if body.EndDate != nil && *body.EndDate != "" {
			if t, err := time.ParseInLocation("2006-01-02", *body.EndDate, loc); err == nil {
				endTime = t.AddDate(0, 0, 1)
			}
		}
		// Views overlapping [startTime, endTime).
		f.where(statsql.StartedAt("a")+` < ?`, endTime).
			where(`COALESCE(a."EndedAt", `+statsql.StartedAt("a")+` + make_interval(secs => COALESCE(a."PlaybackDuration", 0)::float8)) >= ?`, startTime)
	}

	type RawRow struct {
		StartedAt  time.Time
		EndedAt    time.Time
		PlayMethod string
	}
	var rawRows []RawRow
	q, args := f.sql(`SELECT started_at AS "StartedAt", ended_at AS "EndedAt", play_method AS "PlayMethod" FROM acts ORDER BY started_at`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rawRows).Error) {
		return
	}

	records := make([]playMethodRecord, 0, len(rawRows))
	for _, r := range rawRows {
		records = append(records, playMethodRecord{StartTime: r.StartedAt, EndTime: r.EndedAt, PlayMethod: r.PlayMethod})
	}

	type StatEntry struct {
		Key         string `json:"Key"`
		Transcodes  int    `json:"Transcodes"`
		DirectPlays int    `json:"DirectPlays"`
	}
	stats := []StatEntry{}
	for _, hc := range countOverlapsPerHour(records, loc) {
		stats = append(stats, StatEntry{Key: hc.Hour.Format("Jan 02, 06 15:00"), Transcodes: hc.Transcodes, DirectPlays: hc.DirectPlays})
	}

	c.JSON(http.StatusOK, gin.H{
		"types": []gin.H{
			{"Id": "Transcodes", "Name": "Transcodes"},
			{"Id": "DirectPlays", "Name": "DirectPlays"},
		},
		"stats": stats,
	})
}

// ---------------------------------------------------------------------------
// GET /stats/getPlaybacksByLibraryOverTime?from=&to=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetPlaybacksByLibraryOverTime(c *gin.Context) {
	tr := parseTimeRange(c)
	type Row struct {
		Date        string `json:"date"        gorm:"column:date"`
		LibraryId   string `json:"libraryId"   gorm:"column:LibraryId"`
		LibraryName string `json:"libraryName" gorm:"column:LibraryName"`
		Count       int    `json:"count"       gorm:"column:count"`
		Clicks      int    `json:"clicks"      gorm:"column:clicks"`
	}
	rows := []Row{}
	q, args := acts().inRange(tr).sql(`
		SELECT TO_CHAR(acts.local_ts::date, 'YYYY-MM-DD') AS date,
		       l."Id" AS "LibraryId", l."Name" AS "LibraryName",
		       ` + aggPlays + ` AS count, ` + aggClicks + ` AS clicks
		FROM acts
		JOIN jf_libraries l ON l."Id" = acts.library_id AND l.archived = false
		GROUP BY acts.local_ts::date, l."Id", l."Name"
		HAVING COUNT(*) FILTER (WHERE is_view) > 0
		ORDER BY acts.local_ts::date, l."Name"`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getPlaybacksScatter?from=&to=  — one point per view
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetPlaybacksScatter(c *gin.Context) {
	tr := parseTimeRange(c)
	type Row struct {
		Ts       string `json:"ts"       gorm:"column:ts"`
		Duration int    `json:"duration" gorm:"column:duration"` // minutes
		Name     string `json:"name"     gorm:"column:name"`
		Type     string `json:"type"     gorm:"column:type"` // Movie | Episode | Audio
	}
	rows := []Row{}
	q, args := acts().inRange(tr).where(`a."IsView"`).sql(`
		SELECT ` + statsql.ISO("started_at") + ` AS ts,
		       GREATEST(FLOOR(duration / 60.0)::int, 1) AS duration,
		       COALESCE(now_playing_item_name, item_name, '') AS name,
		       media_type AS type
		FROM acts
		ORDER BY started_at DESC
		LIMIT 3000`)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getWatchHeatmap?from=&to=&userId=  (weekday × hour, local time)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetWatchHeatmap(c *gin.Context) {
	tr := parseTimeRange(c)
	type sqlRow struct {
		Day      int
		Hour     int
		Plays    int
		Duration int
	}
	var sqlRows []sqlRow
	q, args := acts().inRange(tr).forUser(c.Query("userId")).sql(`
		SELECT EXTRACT(DOW FROM local_ts)::int AS day, EXTRACT(HOUR FROM local_ts)::int AS hour,
		       ` + aggPlays + ` AS plays, ` + aggMinutes + ` AS duration
		FROM acts GROUP BY 1, 2`)
	if dbError(c, h.db.Raw(q, args...).Scan(&sqlRows).Error) {
		return
	}

	type key struct{ day, hour int }
	lookup := make(map[key]sqlRow, len(sqlRows))
	for _, r := range sqlRows {
		lookup[key{r.Day, r.Hour}] = r
	}
	type cell struct {
		Day      int `json:"day"`
		Hour     int `json:"hour"`
		Plays    int `json:"plays"`
		Duration int `json:"duration"`
	}
	cells := make([]cell, 0, 7*24)
	maxPlays, maxDuration := 0, 0
	for d := 0; d < 7; d++ {
		for hr := 0; hr < 24; hr++ {
			cl := cell{Day: d, Hour: hr}
			if r, ok := lookup[key{d, hr}]; ok {
				cl.Plays = r.Plays
				cl.Duration = r.Duration
			}
			if cl.Plays > maxPlays {
				maxPlays = cl.Plays
			}
			if cl.Duration > maxDuration {
				maxDuration = cl.Duration
			}
			cells = append(cells, cl)
		}
	}
	c.JSON(http.StatusOK, gin.H{"cells": cells, "maxPlays": maxPlays, "maxDuration": maxDuration})
}

// ---------------------------------------------------------------------------
// GET /stats/getTimeToWatch?libraryId=&limit=20
// Days between a movie / episode being added and its first view.
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetTimeToWatch(c *gin.Context) {
	libraryID := c.Query("libraryId")
	limit := parseDays(c.DefaultQuery("limit", "20"), 20)

	type row struct {
		Id           string
		Name         string
		Type         string
		DaysToWatch  float64
		DateAdded    string
		FirstWatched string
	}
	outer := `,
		first_watch AS (
		  SELECT item_id, MIN(started_at) AS first_played
		  FROM acts WHERE is_view AND media_type IN ('Movie', 'Episode')
		  GROUP BY item_id
		),
		item_dates AS (
		  SELECT i."Id", i."Name", i."Type", NULLIF(i."DateCreated", '')::timestamptz AS date_added, i."ParentId"
		  FROM jf_library_items i
		  WHERE i.archived = false AND i."Type" = 'Movie'
		  UNION ALL
		  SELECT e."Id", e."Name", COALESCE(e."Type", 'Episode'), NULLIF(e."DateCreated", '')::timestamptz, s."ParentId"
		  FROM jf_library_episodes e
		  JOIN jf_library_items s ON s."Id" = e."SeriesId"
		  WHERE e.archived = false
		)
		SELECT id."Id", id."Name", id."Type",
		       EXTRACT(EPOCH FROM (fw.first_played - id.date_added)) / 86400.0 AS days_to_watch,
		       ` + `TO_CHAR(` + statsql.AtLocal("id.date_added") + `, 'YYYY-MM-DD') AS date_added,
		       TO_CHAR(` + statsql.AtLocal("fw.first_played") + `, 'YYYY-MM-DD') AS first_watched
		FROM item_dates id
		JOIN first_watch fw ON fw.item_id = id."Id"
		WHERE id.date_added IS NOT NULL AND fw.first_played >= id.date_added`
	var outerArgs []interface{}
	if libraryID != "" {
		outer += ` AND id."ParentId" = ?`
		outerArgs = append(outerArgs, libraryID)
	}
	outer += ` ORDER BY days_to_watch DESC`

	var rows []row
	q, args := acts().where(`a."IsView"`).sql(outer, outerArgs...)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}

	if len(rows) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"avgDaysToWatch":    0,
			"medianDaysToWatch": 0,
			"distribution":      []gin.H{},
			"slowestItems":      []gin.H{},
			"fastestItems":      []gin.H{},
		})
		return
	}

	var total float64
	for _, r := range rows {
		total += r.DaysToWatch
	}
	avg := total / float64(len(rows))

	n := len(rows)
	var median float64
	if n%2 == 0 {
		median = (rows[n/2-1].DaysToWatch + rows[n/2].DaysToWatch) / 2.0
	} else {
		median = rows[n/2].DaysToWatch
	}

	type bucket struct {
		Label string
		Max   float64 // exclusive upper bound; -1 means infinity
	}
	buckets := []bucket{
		{"Same day", 1}, {"1-3 days", 4}, {"4-7 days", 8}, {"1-2 weeks", 15},
		{"2-4 weeks", 29}, {"1-3 months", 91}, {"3+ months", -1},
	}
	counts := make([]int, len(buckets))
	for _, r := range rows {
		for i, b := range buckets {
			if b.Max < 0 || r.DaysToWatch < b.Max {
				counts[i]++
				break
			}
		}
	}
	type distEntry struct {
		Bucket string `json:"bucket"`
		Count  int    `json:"count"`
	}
	distribution := make([]distEntry, len(buckets))
	for i, b := range buckets {
		distribution[i] = distEntry{Bucket: b.Label, Count: counts[i]}
	}

	if limit > len(rows) {
		limit = len(rows)
	}
	slowest := rows[:limit]
	fastest := make([]row, limit)
	copy(fastest, rows[len(rows)-limit:])
	for i, j := 0, len(fastest)-1; i < j; i, j = i+1, j-1 {
		fastest[i], fastest[j] = fastest[j], fastest[i]
	}

	toJSON := func(items []row) []gin.H {
		out := make([]gin.H, len(items))
		for i, r := range items {
			out[i] = gin.H{
				"id":           r.Id,
				"name":         r.Name,
				"type":         r.Type,
				"daysToWatch":  math.Round(r.DaysToWatch*100) / 100,
				"dateAdded":    r.DateAdded,
				"firstWatched": r.FirstWatched,
			}
		}
		return out
	}

	c.JSON(http.StatusOK, gin.H{
		"avgDaysToWatch":    math.Round(avg*100) / 100,
		"medianDaysToWatch": math.Round(median*100) / 100,
		"distribution":      distribution,
		"slowestItems":      toJSON(slowest),
		"fastestItems":      toJSON(fastest),
	})
}

// ---------------------------------------------------------------------------
// GET /stats/getUnwatchedContent?libraryId=&type=&page=1&pageSize=25
// An item is watched when it (movie) or one of its episodes / tracks
// (series / album) has at least one view.
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetUnwatchedContent(c *gin.Context) {
	libraryId := c.Query("libraryId")
	typeFilter := c.Query("type")
	pageSize := parseDays(c.DefaultQuery("pageSize", "25"), 25)
	page := parseDays(c.DefaultQuery("page", "1"), 1)
	offset := (page - 1) * pageSize

	watchedCTE := `, watched AS (SELECT DISTINCT group_id FROM acts WHERE group_id IS NOT NULL)`
	f := acts().where(`a."IsView"`)

	baseWhere := ` WHERE i.archived = false AND i."Type" NOT IN ('Season', 'Folder')`
	var baseArgs []interface{}
	if libraryId != "" {
		baseWhere += ` AND i."ParentId" = ?`
		baseArgs = append(baseArgs, libraryId)
	}

	type TypeStat struct {
		Type      string `json:"type"`
		Total     int    `json:"total"`
		Unwatched int    `json:"unwatched"`
	}
	typeStats := []TypeStat{}
	q, args := f.sql(watchedCTE+`
		SELECT i."Type" AS type, COUNT(*)::int AS total,
		       COUNT(*) FILTER (WHERE w.group_id IS NULL)::int AS unwatched
		FROM jf_library_items i
		LEFT JOIN watched w ON w.group_id = i."Id"`+baseWhere+`
		GROUP BY i."Type"`, baseArgs...)
	if dbError(c, h.db.Raw(q, args...).Scan(&typeStats).Error) {
		return
	}

	totalItems, unwatchedItems := 0, 0
	for _, ts := range typeStats {
		totalItems += ts.Total
		unwatchedItems += ts.Unwatched
	}
	unwatchedPercent := 0.0
	if totalItems > 0 {
		unwatchedPercent = math.Round(float64(unwatchedItems)/float64(totalItems)*1000) / 10
	}

	listWhere := baseWhere + ` AND w.group_id IS NULL`
	listArgs := append([]interface{}{}, baseArgs...)
	if typeFilter != "" {
		listWhere += ` AND i."Type" = ?`
		listArgs = append(listArgs, typeFilter)
	}

	var total int64
	q, args = f.sql(watchedCTE+`
		SELECT COUNT(*) FROM jf_library_items i LEFT JOIN watched w ON w.group_id = i."Id"`+listWhere, listArgs...)
	if dbError(c, h.db.Raw(q, args...).Scan(&total).Error) {
		return
	}

	type ItemRow struct {
		Id          string          `json:"id"`
		Name        string          `json:"name"`
		Type        string          `json:"type"`
		DateAdded   *string         `json:"dateAdded"`
		Genres      json.RawMessage `json:"genres"`
		LibraryName string          `json:"libraryName"`
	}
	rows := []ItemRow{}
	q, args = f.sql(watchedCTE+`
		SELECT i."Id" AS id, i."Name" AS name, i."Type" AS type, i."DateCreated" AS date_added,
		       i."Genres" AS genres, l."Name" AS library_name
		FROM jf_library_items i
		LEFT JOIN jf_libraries l ON l."Id" = i."ParentId"
		LEFT JOIN watched w ON w.group_id = i."Id"`+listWhere+`
		ORDER BY i."DateCreated" DESC NULLS LAST LIMIT ? OFFSET ?`, append(listArgs, pageSize, offset)...)
	if dbError(c, h.db.Raw(q, args...).Scan(&rows).Error) {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": gin.H{
			"totalItems":       totalItems,
			"unwatchedItems":   unwatchedItems,
			"unwatchedPercent": unwatchedPercent,
			"byType":           typeStats,
		},
		"items": paginate(int(total), pageSize, page, rows),
	})
}

// ---------------------------------------------------------------------------
// GET /stats/getBingeStats?from=&to=&userId=
// A binge session = >= 3 distinct episodes of the same series viewed by the
// same user on the same local calendar day.
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetBingeStats(c *gin.Context) {
	tr := parseTimeRange(c)
	f := acts().inRange(tr).forUser(c.Query("userId")).where(`a."MediaType" = 'Episode'`)

	var totalBingeSessions int64
	q, args := f.sql(bingeCTE + `SELECT COUNT(*) FROM binge_sessions`)
	if dbError(c, h.db.Raw(q, args...).Scan(&totalBingeSessions).Error) {
		return
	}

	type SeriesRow struct {
		SeriesId             string  `json:"seriesId"`
		SeriesName           string  `json:"seriesName"`
		BingeCount           int64   `json:"bingeCount"`
		TotalEpisodesWatched int64   `json:"totalEpisodesWatched"`
		AvgEpisodesPerBinge  float64 `json:"avgEpisodesPerBinge"`
	}
	topSeries := []SeriesRow{}
	q, args = f.sql(bingeCTE + `
		SELECT series_id, COALESCE(MAX(series_name), '') AS series_name,
		       COUNT(*) AS binge_count,
		       SUM(episode_count) AS total_episodes_watched,
		       ROUND(AVG(episode_count)::numeric, 1)::float8 AS avg_episodes_per_binge
		FROM binge_sessions
		GROUP BY series_id
		ORDER BY binge_count DESC, total_episodes_watched DESC
		LIMIT 20`)
	if dbError(c, h.db.Raw(q, args...).Scan(&topSeries).Error) {
		return
	}

	type UserRow struct {
		UserId               string `json:"userId"`
		UserName             string `json:"userName"`
		BingeCount           int64  `json:"bingeCount"`
		TotalEpisodesWatched int64  `json:"totalEpisodesWatched"`
	}
	topUsers := []UserRow{}
	q, args = f.sql(bingeCTE + `
		SELECT user_id, COALESCE(MAX(user_name), '') AS user_name,
		       COUNT(*) AS binge_count,
		       SUM(episode_count) AS total_episodes_watched
		FROM binge_sessions
		GROUP BY user_id
		ORDER BY binge_count DESC, total_episodes_watched DESC
		LIMIT 20`)
	if dbError(c, h.db.Raw(q, args...).Scan(&topUsers).Error) {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"totalBingeSessions": totalBingeSessions,
		"topBingedSeries":    topSeries,
		"topBingeUsers":      topUsers,
	})
}

// ---------------------------------------------------------------------------
// GET /stats/getCompletionRate?from=&to=&userId=&libraryId=
// Unit = (user, item, local day) with >= 1 view; completed when the furthest
// stop position or longest sitting reaches statsql.CompletionThreshold of the
// runtime. totalPlays/completedPlays/abandonedPlays count those units.
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetCompletionRate(c *gin.Context) {
	tr := parseTimeRange(c)
	f := acts().inRange(tr).forUser(c.Query("userId"))
	if libraryId := c.Query("libraryId"); libraryId != "" {
		f.where(libraryIdExpr+` = ?`, libraryId)
	}

	type OverallResult struct {
		AvgCompletionRate float64 `json:"avgCompletionRate"`
		TotalPlays        int     `json:"totalPlays"`
		CompletedPlays    int     `json:"completedPlays"`
		AbandonedPlays    int     `json:"abandonedPlays"`
	}
	var overall OverallResult
	q, args := f.sql(completionCTE + `
		SELECT COALESCE(AVG(completion_ratio), 0)::float8 AS "avgCompletionRate",
		       COUNT(*)::int AS "totalPlays",
		       COUNT(*) FILTER (WHERE ` + completedCond + `)::int AS "completedPlays",
		       COUNT(*) FILTER (WHERE NOT (` + completedCond + `))::int AS "abandonedPlays"
		FROM completion`)
	if dbError(c, h.db.Raw(q, args...).Scan(&overall).Error) {
		return
	}

	type ByTypeRow struct {
		Type              string  `json:"type"`
		AvgCompletionRate float64 `json:"avgCompletionRate"`
		TotalPlays        int     `json:"totalPlays"`
		CompletedPlays    int     `json:"completedPlays"`
	}
	byType := []ByTypeRow{}
	q, args = f.sql(completionCTE + `
		SELECT item_type AS "type",
		       COALESCE(AVG(completion_ratio), 0)::float8 AS "avgCompletionRate",
		       COUNT(*)::int AS "totalPlays",
		       COUNT(*) FILTER (WHERE ` + completedCond + `)::int AS "completedPlays"
		FROM completion
		GROUP BY item_type
		ORDER BY "totalPlays" DESC`)
	if dbError(c, h.db.Raw(q, args...).Scan(&byType).Error) {
		return
	}

	type DistRow struct {
		Bucket string `json:"bucket"`
		Count  int    `json:"count"`
	}
	dist := []DistRow{}
	q, args = f.sql(completionCTE + `
		SELECT b.bucket, COALESCE(d.cnt, 0)::int AS "count"
		FROM (VALUES (1, '0-25%'), (2, '25-50%'), (3, '50-75%'), (4, '75-100%')) AS b(ord, bucket)
		LEFT JOIN (
		  SELECT CASE
		           WHEN completion_ratio < 0.25 THEN '0-25%'
		           WHEN completion_ratio < 0.50 THEN '25-50%'
		           WHEN completion_ratio < 0.75 THEN '50-75%'
		           ELSE '75-100%'
		         END AS bucket,
		         COUNT(*)::int AS cnt
		  FROM completion GROUP BY 1
		) d ON d.bucket = b.bucket
		ORDER BY b.ord`)
	if dbError(c, h.db.Raw(q, args...).Scan(&dist).Error) {
		return
	}

	overall.AvgCompletionRate = math.Round(overall.AvgCompletionRate*100) / 100
	for i := range byType {
		byType[i].AvgCompletionRate = math.Round(byType[i].AvgCompletionRate*100) / 100
	}

	c.JSON(http.StatusOK, gin.H{
		"overall":      overall,
		"byType":       byType,
		"distribution": dist,
		"threshold":    statsql.CompletionThreshold,
	})
}

// ---------------------------------------------------------------------------
// GET /stats/getViewingDiversity?from=&to=&userId=   (views only)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetViewingDiversity(c *gin.Context) {
	tr := parseTimeRange(c)
	userId := c.Query("userId")
	f := acts().inRange(tr).forUser(userId).where(`a."IsView"`)

	type genreRow struct {
		UserId   string
		UserName string
		Genre    string
		Plays    int
	}
	var genreRows []genreRow
	q, args := f.sql(`
		SELECT user_id, MAX(user_name) AS user_name, genre, COUNT(*)::int AS plays
		FROM acts ` + genreLateral + `
		WHERE acts.genres IS NOT NULL AND user_id IS NOT NULL
		GROUP BY user_id, genre`)
	if dbError(c, h.db.Raw(q, args...).Scan(&genreRows).Error) {
		return
	}

	type libRow struct {
		UserId      string
		LibraryId   string
		LibraryName string
		Plays       int
	}
	var libRows []libRow
	q, args = f.sql(`
		SELECT acts.user_id, l."Id" AS library_id, l."Name" AS library_name, COUNT(*)::int AS plays
		FROM acts JOIN jf_libraries l ON l."Id" = acts.library_id
		WHERE acts.user_id IS NOT NULL
		GROUP BY acts.user_id, l."Id", l."Name"`)
	if dbError(c, h.db.Raw(q, args...).Scan(&libRows).Error) {
		return
	}

	type itemRow struct {
		UserId      string
		UserName    string
		UniqueItems int
		TotalPlays  int
	}
	var itemRows []itemRow
	q, args = f.sql(`
		SELECT user_id, MAX(user_name) AS user_name,
		       COUNT(DISTINCT item_id)::int AS unique_items, COUNT(*)::int AS total_plays
		FROM acts WHERE user_id IS NOT NULL
		GROUP BY user_id`)
	if dbError(c, h.db.Raw(q, args...).Scan(&itemRows).Error) {
		return
	}

	type genreInfo struct {
		Genre string
		Plays int
	}
	userGenres := map[string][]genreInfo{}
	userNames := map[string]string{}
	for _, r := range genreRows {
		userGenres[r.UserId] = append(userGenres[r.UserId], genreInfo{Genre: r.Genre, Plays: r.Plays})
	}
	type libInfo struct {
		LibraryId   string
		LibraryName string
		Plays       int
	}
	userLibs := map[string][]libInfo{}
	for _, r := range libRows {
		userLibs[r.UserId] = append(userLibs[r.UserId], libInfo{LibraryId: r.LibraryId, LibraryName: r.LibraryName, Plays: r.Plays})
	}
	userItems := map[string]itemRow{}
	for _, r := range itemRows {
		userItems[r.UserId] = r
		userNames[r.UserId] = r.UserName
	}

	shannonDiversity := func(genres []genreInfo) float64 {
		total := 0
		for _, g := range genres {
			total += g.Plays
		}
		if total == 0 || len(genres) <= 1 {
			return 0
		}
		entropy := 0.0
		for _, g := range genres {
			p := float64(g.Plays) / float64(total)
			if p > 0 {
				entropy -= p * math.Log(p)
			}
		}
		maxEntropy := math.Log(float64(len(genres)))
		if maxEntropy == 0 {
			return 0
		}
		return math.Round(entropy/maxEntropy*100) / 100
	}

	if userId != "" {
		genres := userGenres[userId]
		libs := userLibs[userId]

		totalGenrePlays := 0
		for _, g := range genres {
			totalGenrePlays += g.Plays
		}
		genreBreakdown := make([]gin.H, 0, len(genres))
		for _, g := range genres {
			pct := 0.0
			if totalGenrePlays > 0 {
				pct = math.Round(float64(g.Plays)/float64(totalGenrePlays)*10000) / 100
			}
			genreBreakdown = append(genreBreakdown, gin.H{"genre": g.Genre, "plays": g.Plays, "percent": pct})
		}
		sort.SliceStable(genreBreakdown, func(i, j int) bool {
			return genreBreakdown[i]["plays"].(int) > genreBreakdown[j]["plays"].(int)
		})

		totalLibPlays := 0
		for _, l := range libs {
			totalLibPlays += l.Plays
		}
		libraryBreakdown := make([]gin.H, 0, len(libs))
		for _, l := range libs {
			pct := 0.0
			if totalLibPlays > 0 {
				pct = math.Round(float64(l.Plays)/float64(totalLibPlays)*10000) / 100
			}
			libraryBreakdown = append(libraryBreakdown, gin.H{
				"libraryId": l.LibraryId, "libraryName": l.LibraryName, "plays": l.Plays, "percent": pct,
			})
		}
		sort.SliceStable(libraryBreakdown, func(i, j int) bool {
			return libraryBreakdown[i]["plays"].(int) > libraryBreakdown[j]["plays"].(int)
		})

		c.JSON(http.StatusOK, gin.H{
			"userId":           userId,
			"userName":         userNames[userId],
			"diversityScore":   shannonDiversity(genres),
			"uniqueGenres":     len(genres),
			"uniqueLibraries":  len(libs),
			"uniqueItems":      userItems[userId].UniqueItems,
			"totalPlays":       userItems[userId].TotalPlays,
			"genreBreakdown":   genreBreakdown,
			"libraryBreakdown": libraryBreakdown,
		})
		return
	}

	type userSummary struct {
		UserId         string  `json:"userId"`
		UserName       string  `json:"userName"`
		DiversityScore float64 `json:"diversityScore"`
		UniqueGenres   int     `json:"uniqueGenres"`
		UniqueLibs     int     `json:"uniqueLibraries"`
		UniqueItems    int     `json:"uniqueItems"`
		TotalPlays     int     `json:"totalPlays"`
	}
	users := make([]userSummary, 0, len(userItems))
	for uid, items := range userItems {
		name := userNames[uid]
		if name == "" {
			name = uid
		}
		users = append(users, userSummary{
			UserId:         uid,
			UserName:       name,
			DiversityScore: shannonDiversity(userGenres[uid]),
			UniqueGenres:   len(userGenres[uid]),
			UniqueLibs:     len(userLibs[uid]),
			UniqueItems:    items.UniqueItems,
			TotalPlays:     items.TotalPlays,
		})
	}
	sort.Slice(users, func(i, j int) bool {
		if users[i].DiversityScore != users[j].DiversityScore {
			return users[i].DiversityScore > users[j].DiversityScore
		}
		return users[i].UserId < users[j].UserId
	})
	c.JSON(http.StatusOK, gin.H{"users": users})
}
