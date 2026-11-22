package handler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type MetricsCollector struct {
	repos *repository.Container
	db    *gorm.DB

	// ── Global ────────────────────────────────────────────────────────────────
	totalPlays     *prometheus.Desc
	totalDuration  *prometheus.Desc
	totalUsers     *prometheus.Desc
	totalLibraries *prometheus.Desc

	// ── Per-user ──────────────────────────────────────────────────────────────
	playsByUser *prometheus.Desc
	hoursByUser *prometheus.Desc

	// ── Per-library ───────────────────────────────────────────────────────────
	playsByLibrary *prometheus.Desc

	// ── Media type / client / playback method ─────────────────────────────────
	playsByMediaType      *prometheus.Desc
	playsByClient         *prometheus.Desc
	playsByPlaybackMethod *prometheus.Desc

	// ── Time patterns ─────────────────────────────────────────────────────────
	playsByHour      *prometheus.Desc // label: hour (0-23)
	playsByDayOfWeek *prometheus.Desc // label: day (0=Sun..6=Sat)
	heatmapPlays     *prometheus.Desc // labels: day, hour

	// ── Top content ───────────────────────────────────────────────────────────
	itemPlayCount *prometheus.Desc // labels: name, type

	// ── Completion ────────────────────────────────────────────────────────────
	completedPlays *prometheus.Desc
	abandonedPlays *prometheus.Desc
	completionRate *prometheus.Desc

	// ── Library inventory ─────────────────────────────────────────────────────
	itemsByType     *prometheus.Desc // label: type
	unwatchedByType *prometheus.Desc // label: type

	// ── Binge watching ────────────────────────────────────────────────────────
	bingeSessions *prometheus.Desc

	// ── Live ──────────────────────────────────────────────────────────────────
	activeSessions *prometheus.Desc
}

func NewMetricsCollector(repos *repository.Container, db *gorm.DB) *MetricsCollector {
	ns := "jellyfin"
	return &MetricsCollector{
		repos: repos,
		db:    db,

		totalPlays:     prometheus.NewDesc(ns+"_total_plays", "Total number of plays all time", nil, nil),
		totalDuration:  prometheus.NewDesc(ns+"_total_duration_seconds", "Total watch time in seconds all time", nil, nil),
		totalUsers:     prometheus.NewDesc(ns+"_total_users", "Total number of users", nil, nil),
		totalLibraries: prometheus.NewDesc(ns+"_total_libraries", "Total number of libraries", nil, nil),

		playsByUser: prometheus.NewDesc(ns+"_plays_by_user_total", "Total plays per user", []string{"user"}, nil),
		hoursByUser: prometheus.NewDesc(ns+"_hours_by_user_total", "Total watch hours per user", []string{"user"}, nil),

		playsByLibrary: prometheus.NewDesc(ns+"_plays_by_library_total", "Total plays per library", []string{"library"}, nil),

		playsByMediaType:      prometheus.NewDesc(ns+"_plays_by_media_type_total", "Total plays per media type", []string{"type"}, nil),
		playsByClient:         prometheus.NewDesc(ns+"_plays_by_client_total", "Total plays per client", []string{"client"}, nil),
		playsByPlaybackMethod: prometheus.NewDesc(ns+"_plays_by_playback_method_total", "Total plays per playback method", []string{"method"}, nil),

		playsByHour:      prometheus.NewDesc(ns+"_plays_by_hour", "Play count per hour of day (all time) — use as pattern gauge", []string{"hour"}, nil),
		playsByDayOfWeek: prometheus.NewDesc(ns+"_plays_by_day_of_week", "Play count per day of week (all time) — use as pattern gauge", []string{"day"}, nil),
		heatmapPlays:     prometheus.NewDesc(ns+"_heatmap_plays", "Play count per hour+day combination — use for Grafana heatmap", []string{"day", "hour"}, nil),

		itemPlayCount: prometheus.NewDesc(ns+"_item_play_count", "Play count per media item (top 50)", []string{"name", "type"}, nil),

		completedPlays: prometheus.NewDesc(ns+"_completed_plays_total", "Completed views: stop position or watched time >= 90% of runtime (one per user/item/local day)", nil, nil),
		abandonedPlays: prometheus.NewDesc(ns+"_abandoned_plays_total", "Abandoned views: < 90% of runtime (one per user/item/local day)", nil, nil),
		completionRate: prometheus.NewDesc(ns+"_completion_rate_percent", "Percentage of plays that are complete", nil, nil),

		itemsByType:     prometheus.NewDesc(ns+"_items_by_type_total", "Total library items per type", []string{"type"}, nil),
		unwatchedByType: prometheus.NewDesc(ns+"_unwatched_items_by_type", "Unwatched items per type", []string{"type"}, nil),

		bingeSessions: prometheus.NewDesc(ns+"_binge_sessions_total", "Binge sessions: >= 3 distinct episodes of one series by one user on one local day", nil, nil),

		activeSessions: prometheus.NewDesc(ns+"_active_sessions", "Number of currently active playback sessions", nil, nil),
	}
}

func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalPlays
	ch <- c.totalDuration
	ch <- c.totalUsers
	ch <- c.totalLibraries
	ch <- c.playsByUser
	ch <- c.hoursByUser
	ch <- c.playsByLibrary
	ch <- c.playsByMediaType
	ch <- c.playsByClient
	ch <- c.playsByPlaybackMethod
	ch <- c.playsByHour
	ch <- c.playsByDayOfWeek
	ch <- c.heatmapPlays
	ch <- c.itemPlayCount
	ch <- c.completedPlays
	ch <- c.abandonedPlays
	ch <- c.completionRate
	ch <- c.itemsByType
	ch <- c.unwatchedByType
	ch <- c.bingeSessions
	ch <- c.activeSessions
}

func (c *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ── Global stats ──────────────────────────────────────────────────────────
	if gs, err := c.repos.Stats.GetGlobalStats(ctx); err == nil {
		ch <- prometheus.MustNewConstMetric(c.totalPlays, prometheus.CounterValue, float64(gs.TotalPlays))
		ch <- prometheus.MustNewConstMetric(c.totalDuration, prometheus.CounterValue, float64(gs.TotalDuration))
		ch <- prometheus.MustNewConstMetric(c.totalUsers, prometheus.GaugeValue, float64(gs.TotalUsers))
		ch <- prometheus.MustNewConstMetric(c.totalLibraries, prometheus.GaugeValue, float64(gs.TotalLibraries))
	} else {
		log.Printf("metrics: GetGlobalStats: %v", err)
	}

	// ── Per-user stats ────────────────────────────────────────────────────────
	if users, err := c.repos.Stats.GetTopUsers(ctx, 500); err == nil {
		// One series per user id (labelled with the current name); a name shared
		// by two ids gets a suffix so label sets never collide.
		userLabels := labelDeduper{}
		for _, u := range users {
			name := userLabels.unique(u.UserName)
			ch <- prometheus.MustNewConstMetric(c.playsByUser, prometheus.CounterValue, float64(u.TotalPlays), name)
			ch <- prometheus.MustNewConstMetric(c.hoursByUser, prometheus.CounterValue, u.TotalHours, name)
		}
	} else {
		log.Printf("metrics: GetTopUsers: %v", err)
	}

	// ── Per-library stats (views) ─────────────────────────────────────────────
	type libRow struct {
		Name       string
		TotalPlays int
	}
	var libRows []libRow
	q, args := acts().sql(`
		SELECT l."Name" AS name, ` + aggPlays + ` AS total_plays
		FROM acts JOIN jf_libraries l ON l."Id" = acts.library_id AND l.archived = false
		GROUP BY l."Id", l."Name"
		ORDER BY total_plays DESC
		LIMIT 100`)
	c.db.WithContext(ctx).Raw(q, args...).Scan(&libRows)
	libLabels := labelDeduper{}
	for _, l := range libRows {
		ch <- prometheus.MustNewConstMetric(c.playsByLibrary, prometheus.CounterValue, float64(l.TotalPlays), libLabels.unique(l.Name))
	}

	// ── Media type (views): Movie, Series (episodes), Audio (music tracks) ────
	type typeRow struct {
		MediaType string
		Count     int
	}
	var typeRows []typeRow
	q, args = acts().sql(`SELECT media_type, ` + aggPlays + ` AS count FROM acts GROUP BY media_type`)
	c.db.WithContext(ctx).Raw(q, args...).Scan(&typeRows)
	totals := map[string]int{"Audio": 0, "Movie": 0, "Series": 0}
	for _, r := range typeRows {
		switch r.MediaType {
		case "Movie":
			totals["Movie"] += r.Count
		case "Episode":
			totals["Series"] += r.Count
		case "Audio":
			totals["Audio"] += r.Count
		}
	}
	for t, count := range totals {
		ch <- prometheus.MustNewConstMetric(c.playsByMediaType, prometheus.CounterValue, float64(count), t)
	}

	// ── Per-client (views) ────────────────────────────────────────────────────
	type clientRow struct {
		Client string
		Count  int
	}
	var clientRows []clientRow
	q, args = acts().where(`a."Client" IS NOT NULL AND a."Client" <> ''`).sql(`
		SELECT client, ` + aggPlays + ` AS count FROM acts
		GROUP BY client ORDER BY count DESC LIMIT 50`)
	c.db.WithContext(ctx).Raw(q, args...).Scan(&clientRows)
	for _, r := range clientRows {
		ch <- prometheus.MustNewConstMetric(c.playsByClient, prometheus.CounterValue, float64(r.Count), r.Client)
	}

	// ── Playback method (views) ───────────────────────────────────────────────
	type methodRow struct {
		Method string
		Count  int
	}
	var methodRows []methodRow
	q, args = acts().sql(`SELECT play_method AS method, ` + aggPlays + ` AS count FROM acts GROUP BY play_method`)
	c.db.WithContext(ctx).Raw(q, args...).Scan(&methodRows)
	for _, r := range methodRows {
		ch <- prometheus.MustNewConstMetric(c.playsByPlaybackMethod, prometheus.CounterValue, float64(r.Count), r.Method)
	}

	// ── Time patterns (views, display time zone) ──────────────────────────────
	type heatRow struct {
		Day   int
		Hour  int
		Plays int
	}
	var heatRows []heatRow
	q, args = acts().sql(`
		SELECT EXTRACT(DOW FROM local_ts)::int AS day, EXTRACT(HOUR FROM local_ts)::int AS hour,
		       ` + aggPlays + ` AS plays
		FROM acts GROUP BY 1, 2`)
	c.db.WithContext(ctx).Raw(q, args...).Scan(&heatRows)
	byHour := map[int]int{}
	byDay := map[int]int{}
	for _, r := range heatRows {
		byHour[r.Hour] += r.Plays
		byDay[r.Day] += r.Plays
		ch <- prometheus.MustNewConstMetric(c.heatmapPlays, prometheus.GaugeValue, float64(r.Plays), fmt.Sprint(r.Day), fmt.Sprint(r.Hour))
	}
	for hr, plays := range byHour {
		ch <- prometheus.MustNewConstMetric(c.playsByHour, prometheus.GaugeValue, float64(plays), fmt.Sprint(hr))
	}
	for d, plays := range byDay {
		ch <- prometheus.MustNewConstMetric(c.playsByDayOfWeek, prometheus.GaugeValue, float64(plays), fmt.Sprint(d))
	}

	// ── Top 50 items (movie / series / album level, views) ────────────────────
	type itemRow struct {
		Name  string
		Type  string
		Plays int
	}
	var itemRows []itemRow
	q, args = acts().where(`a."IsView"`).sql(`
		SELECT COALESCE(group_name, 'Unknown') AS name, display_type AS type, COUNT(*)::int AS plays
		FROM acts
		GROUP BY 1, 2 ORDER BY plays DESC LIMIT 50`)
	c.db.WithContext(ctx).Raw(q, args...).Scan(&itemRows)
	for _, r := range itemRows {
		ch <- prometheus.MustNewConstMetric(c.itemPlayCount, prometheus.GaugeValue, float64(r.Plays), r.Name, r.Type)
	}

	// ── Completion (same definition as /stats/getCompletionRate) ──────────────
	var comp struct {
		Completed int
		Abandoned int
	}
	q, args = acts().sql(completionCTE + `
		SELECT COUNT(*) FILTER (WHERE ` + completedCond + `)::int AS completed,
		       COUNT(*) FILTER (WHERE NOT (` + completedCond + `))::int AS abandoned
		FROM completion`)
	c.db.WithContext(ctx).Raw(q, args...).Scan(&comp)
	total := comp.Completed + comp.Abandoned
	rate := 0.0
	if total > 0 {
		rate = float64(comp.Completed) / float64(total) * 100
	}
	ch <- prometheus.MustNewConstMetric(c.completedPlays, prometheus.CounterValue, float64(comp.Completed))
	ch <- prometheus.MustNewConstMetric(c.abandonedPlays, prometheus.CounterValue, float64(comp.Abandoned))
	ch <- prometheus.MustNewConstMetric(c.completionRate, prometheus.GaugeValue, rate)

	// ── Inventory and unwatched items by type ─────────────────────────────────
	type invRow struct {
		Type      string
		Total     int
		Unwatched int
	}
	var invRows []invRow
	q, args = acts().where(`a."IsView"`).sql(`,
		watched_items  AS (SELECT DISTINCT item_id  FROM acts WHERE item_id  IS NOT NULL),
		watched_groups AS (SELECT DISTINCT group_id FROM acts WHERE group_id IS NOT NULL)
		SELECT 'Movie' AS type, COUNT(*)::int AS total,
		       COUNT(*) FILTER (WHERE NOT EXISTS (SELECT 1 FROM watched_items w WHERE w.item_id = i."Id"))::int AS unwatched
		FROM jf_library_items i WHERE i.archived = false AND i."Type" = 'Movie'
		UNION ALL
		SELECT 'Series', COUNT(*)::int,
		       COUNT(*) FILTER (WHERE NOT EXISTS (SELECT 1 FROM watched_groups w WHERE w.group_id = i."Id"))::int
		FROM jf_library_items i WHERE i.archived = false AND i."Type" = 'Series'
		UNION ALL
		SELECT 'Episode', COUNT(*)::int,
		       COUNT(*) FILTER (WHERE NOT EXISTS (SELECT 1 FROM watched_items w WHERE w.item_id = e."Id"))::int
		FROM jf_library_episodes e WHERE e.archived = false
		UNION ALL
		SELECT 'Audio', COUNT(*)::int,
		       COUNT(*) FILTER (WHERE NOT EXISTS (SELECT 1 FROM watched_items w WHERE w.item_id = t."Id"))::int
		FROM jf_music_tracks t WHERE t.archived = false`)
	c.db.WithContext(ctx).Raw(q, args...).Scan(&invRows)
	for _, r := range invRows {
		ch <- prometheus.MustNewConstMetric(c.itemsByType, prometheus.GaugeValue, float64(r.Total), r.Type)
		ch <- prometheus.MustNewConstMetric(c.unwatchedByType, prometheus.GaugeValue, float64(r.Unwatched), r.Type)
	}

	// ── Binge sessions (same definition as /stats/getBingeStats) ──────────────
	var bingeCount int
	q, args = acts().where(`a."MediaType" = 'Episode'`).sql(bingeCTE + `SELECT COUNT(*)::int FROM binge_sessions`)
	c.db.WithContext(ctx).Raw(q, args...).Scan(&bingeCount)
	ch <- prometheus.MustNewConstMetric(c.bingeSessions, prometheus.CounterValue, float64(bingeCount))

	// ── Active sessions ───────────────────────────────────────────────────────
	if sessions, err := c.repos.Watchdog.List(ctx); err == nil {
		ch <- prometheus.MustNewConstMetric(c.activeSessions, prometheus.GaugeValue, float64(len(sessions)))
	} else {
		log.Printf("metrics: Watchdog.List: %v", err)
	}
}

// labelDeduper makes label values unique within one metric family (duplicate
// label sets make the whole /metrics scrape fail).
type labelDeduper map[string]int

func (d labelDeduper) unique(v string) string {
	d[v]++
	if n := d[v]; n > 1 {
		return fmt.Sprintf("%s (%d)", v, n)
	}
	return v
}
