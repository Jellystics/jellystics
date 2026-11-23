package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// backupTableDef mirrors the hardcoded list from backend/global/backup_tables.js.
type backupTableDef struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

var backupTables = []backupTableDef{
	{Value: "jf_libraries", Name: "Libraries"},
	{Value: "jf_library_items", Name: "Library Items"},
	{Value: "jf_library_seasons", Name: "Seasons"},
	{Value: "jf_library_episodes", Name: "Episodes"},
	{Value: "jf_users", Name: "Users"},
	{Value: "jf_playback_activity", Name: "Activity"},
	{Value: "jf_playback_reporting_plugin_data", Name: "Playback Reporting Plugin Data"},
	{Value: "jf_item_info", Name: "Item Info"},
}

// AdminApiHandler handles admin-level purge, activity deletion and backup-table endpoints.
type AdminApiHandler struct {
	repos *repository.Container
	db    *gorm.DB
}

// NewAdminApiHandler constructs an AdminApiHandler.
func NewAdminApiHandler(repos *repository.Container, db *gorm.DB) *AdminApiHandler {
	return &AdminApiHandler{repos: repos, db: db}
}

// ---------------------------------------------------------------------------
// Purge endpoints
// ---------------------------------------------------------------------------

// DELETE /api/item/purge
// Body: { "id": "<uuid>", "withActivity": true|false }
func (h *AdminApiHandler) PurgeItem(c *gin.Context) {
	var body struct {
		Id           string `json:"id"`
		WithActivity bool   `json:"withActivity"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	id := body.Id
	db := h.db

	// Check if there is a library item with this id.
	type idRow struct{ Id string }
	var libItems []idRow
	db.Raw(`SELECT "Id" FROM jf_library_items WHERE "Id" = ?`, id).Scan(&libItems)

	// Get seasons associated with this id (as series or directly).
	type seasonRow struct {
		Id       string
		Archived bool
	}
	var seasons []seasonRow
	db.Raw(`SELECT "Id", archived FROM jf_library_seasons WHERE "SeriesId" = ? OR "Id" = ?`, id, id).Scan(&seasons)

	itemArchived := len(libItems) == 0 || func() bool {
		var r struct{ Archived bool }
		db.Raw(`SELECT archived FROM jf_library_items WHERE "Id" = ?`, id).Scan(&r)
		return r.Archived
	}()

	if len(seasons) > 0 {
		for _, season := range seasons {
			// Delete archived episodes for each season (or all if season/item is archived).
			episodeQuery := `DELETE FROM jf_library_episodes WHERE "SeasonId" = ?`
			if !season.Archived && !itemArchived {
				episodeQuery += ` AND archived = true`
			}
			db.Exec(episodeQuery, season.Id)
			// Delete the season itself if archived.
			if season.Archived || itemArchived {
				db.Exec(`DELETE FROM jf_library_seasons WHERE "Id" = ?`, season.Id)
			}
		}
	} else {
		// Try as episode.
		var archivedEpisodes []struct{ EpisodeId *string }
		db.Raw(`SELECT "EpisodeId" FROM jf_library_episodes WHERE "EpisodeId" = ? AND archived = true`, id).Scan(&archivedEpisodes)
		if len(archivedEpisodes) > 0 {
			db.Exec(`DELETE FROM jf_library_episodes WHERE "EpisodeId" = ? AND archived = true`, id)
		}
		if itemArchived && len(libItems) > 0 {
			db.Exec(`DELETE FROM jf_library_episodes WHERE "SeriesId" = ?`, id)
			db.Exec(`DELETE FROM jf_library_seasons WHERE "SeriesId" = ?`, id)
			db.Exec(`DELETE FROM jf_library_items WHERE "Id" = ?`, id)
		}
	}

	if body.WithActivity {
		// Collect episode IDs from seasons.
		var episodeIds []string
		if len(seasons) > 0 {
			for _, s := range seasons {
				var eids []struct{ Id string }
				db.Raw(`SELECT "Id" FROM jf_library_episodes WHERE "SeasonId" = ?`, s.Id).Scan(&eids)
				for _, e := range eids {
					episodeIds = append(episodeIds, e.Id)
				}
			}
		}
		var seasonIds []string
		for _, s := range seasons {
			seasonIds = append(seasonIds, s.Id)
		}
		h.deletePlaybackForItem(db, id, episodeIds, seasonIds)
	}

	// Refresh materialized views.
	_ = repository.RefreshStatsViews(db)
	c.JSON(http.StatusOK, gin.H{"message": "item purged successfully"})
}

// DELETE /api/library/purge
// Body: { "id": "<uuid>", "withActivity": true|false }
func (h *AdminApiHandler) PurgeLibrary(c *gin.Context) {
	var body struct {
		Id           string `json:"id"`
		WithActivity bool   `json:"withActivity"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	h.purgeLibraryItems(body.Id, body.WithActivity, true)
	h.db.Exec(`DELETE FROM jf_libraries WHERE "Id" = ?`, body.Id)
	_ = repository.RefreshStatsViews(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "library purged successfully"})
}

// DELETE /api/libraryItems/purge
// Body: { "id": "<uuid>", "withActivity": true|false }
func (h *AdminApiHandler) PurgeLibraryItems(c *gin.Context) {
	var body struct {
		Id           string `json:"id"`
		WithActivity bool   `json:"withActivity"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	h.purgeLibraryItems(body.Id, body.WithActivity, false)
	_ = repository.RefreshStatsViews(h.db)
	c.JSON(http.StatusOK, gin.H{"message": "library items purged successfully"})
}

// purgeLibraryItems replicates the Node.js purgeLibraryItems() helper.
// purgeAll=true deletes all items; purgeAll=false deletes only archived ones.
func (h *AdminApiHandler) purgeLibraryItems(libraryId string, withActivity bool, purgeAll bool) {
	db := h.db

	type itemRow struct {
		Id       string
		Archived bool
	}
	var items []itemRow
	db.Raw(`SELECT "Id", archived FROM jf_library_items WHERE "ParentId" = ?`, libraryId).Scan(&items)

	var allSeasonIds []string
	var allEpisodeIds []string

	for _, item := range items {
		type seasonRow struct {
			Id       string
			Archived bool
		}
		var seasons []seasonRow
		seasonQuery := `SELECT "Id", archived FROM jf_library_seasons WHERE "SeriesId" = ?`
		if !item.Archived && !purgeAll {
			seasonQuery += ` AND archived = true`
		}
		db.Raw(seasonQuery, item.Id).Scan(&seasons)

		for _, season := range seasons {
			allSeasonIds = append(allSeasonIds, season.Id)

			var episodeIds []struct{ Id string }
			epQuery := `SELECT "Id" FROM jf_library_episodes WHERE "SeasonId" = ?`
			if !item.Archived && !season.Archived && !purgeAll {
				epQuery += ` AND archived = true`
			}
			db.Raw(epQuery, season.Id).Scan(&episodeIds)
			for _, e := range episodeIds {
				allEpisodeIds = append(allEpisodeIds, e.Id)
			}
		}

		if len(seasons) == 0 {
			// No seasons — try episodes directly under series.
			var episodeIds []struct{ Id string }
			epQuery := `SELECT "Id" FROM jf_library_episodes WHERE "SeriesId" = ?`
			if !item.Archived && !purgeAll {
				epQuery += ` AND archived = true`
			}
			db.Raw(epQuery, item.Id).Scan(&episodeIds)
			for _, e := range episodeIds {
				allEpisodeIds = append(allEpisodeIds, e.Id)
			}
		}
	}

	// Delete episodes.
	if len(allEpisodeIds) > 0 {
		db.Exec(`DELETE FROM jf_library_episodes WHERE "Id" IN ?`, allEpisodeIds)
	}
	// Delete seasons.
	if len(allSeasonIds) > 0 {
		db.Exec(`DELETE FROM jf_library_seasons WHERE "Id" IN ?`, allSeasonIds)
	}
	// Delete library items.
	itemsQuery := `DELETE FROM jf_library_items WHERE "ParentId" = ?`
	if !purgeAll {
		itemsQuery += ` AND archived = true`
	}
	db.Exec(itemsQuery, libraryId)

	if withActivity {
		h.deletePlaybackForItem(db, libraryId, allEpisodeIds, allSeasonIds)
	}
}

// deletePlaybackForItem removes playback_activity rows related to a purged item.
func (h *AdminApiHandler) deletePlaybackForItem(db *gorm.DB, itemId string, episodeIds, seasonIds []string) {
	parts := []string{`"NowPlayingItemId" = ?`}
	args := []interface{}{itemId}

	if len(episodeIds) > 0 {
		db.Exec(`DELETE FROM jf_playback_activity WHERE "EpisodeId" IN ?`, episodeIds)
	}
	if len(seasonIds) > 0 {
		db.Exec(`DELETE FROM jf_playback_activity WHERE "SeasonId" IN ?`, seasonIds)
	}

	query := `DELETE FROM jf_playback_activity WHERE ` + strings.Join(parts, " OR ")
	db.Exec(query, args...)
	ResetStatsCaches()
}

// POST /api/deletePlaybackActivity
// Body: { "ids": ["id1","id2",...] }
func (h *AdminApiHandler) DeletePlaybackActivity(c *gin.Context) {
	var body struct {
		Ids []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a non-empty list of ids is required"})
		return
	}
	if err := h.db.Exec(`DELETE FROM jf_playback_activity WHERE "Id" IN ?`, body.Ids).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ResetStatsCaches()
	// Refresh materialized views.
	_ = repository.RefreshStatsViews(h.db)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d records deleted", len(body.Ids))})
}

// ---------------------------------------------------------------------------
// Backup table endpoints
// ---------------------------------------------------------------------------

// getExcludedTables reads the ExcludedTables list from app_config settings JSON.
func (h *AdminApiHandler) getExcludedTables(c *gin.Context) ([]string, error) {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		return nil, err
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
		return []string{}, nil
	}
	raw, ok := settings["ExcludedTables"]
	if !ok {
		return []string{}, nil
	}
	iface, ok := raw.([]interface{})
	if !ok {
		return []string{}, nil
	}
	out := make([]string, 0, len(iface))
	for _, v := range iface {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out, nil
}

// setExcludedTables persists the ExcludedTables list back to app_config.
func (h *AdminApiHandler) setExcludedTables(c *gin.Context, excluded []string) error {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		return err
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
		settings = map[string]interface{}{}
	}
	settings["ExcludedTables"] = excluded
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	cfg.Settings = raw
	return h.repos.Config.Save(c.Request.Context(), cfg)
}

// GET /api/getBackupTables
func (h *AdminApiHandler) GetBackupTables(c *gin.Context) {
	excluded, err := h.getExcludedTables(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	excludedSet := make(map[string]bool, len(excluded))
	for _, t := range excluded {
		excludedSet[t] = true
	}

	type tableEntry struct {
		Value    string `json:"value"`
		Name     string `json:"name"`
		Excluded bool   `json:"Excluded"`
	}
	result := make([]tableEntry, len(backupTables))
	for i, t := range backupTables {
		result[i] = tableEntry{
			Value:    t.Value,
			Name:     t.Name,
			Excluded: excludedSet[t.Value],
		}
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/setExcludedBackupTable
// Body: { "table": "<tableName>" }
// Toggles the table in/out of the excluded list.
func (h *AdminApiHandler) SetExcludedBackupTable(c *gin.Context) {
	var body struct {
		Table string `json:"table"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Table == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "table is required"})
		return
	}

	// Validate table name against known list.
	valid := false
	for _, t := range backupTables {
		if t.Value == body.Table {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table provided"})
		return
	}

	excluded, err := h.getExcludedTables(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Toggle.
	found := false
	newExcluded := make([]string, 0, len(excluded))
	for _, t := range excluded {
		if t == body.Table {
			found = true
		} else {
			newExcluded = append(newExcluded, t)
		}
	}
	if !found {
		newExcluded = append(newExcluded, body.Table)
	}

	if err := h.setExcludedTables(c, newExcluded); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	excludedSet := make(map[string]bool, len(newExcluded))
	for _, t := range newExcluded {
		excludedSet[t] = true
	}

	type tableEntry struct {
		Value    string `json:"value"`
		Name     string `json:"name"`
		Excluded bool   `json:"Excluded"`
	}
	result := make([]tableEntry, len(backupTables))
	for i, t := range backupTables {
		result[i] = tableEntry{
			Value:    t.Value,
			Name:     t.Name,
			Excluded: excludedSet[t.Value],
		}
	}
	c.JSON(http.StatusOK, result)
}
