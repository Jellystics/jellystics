package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Values of jf_playback_activity.source.
const (
	SourceTracker = "watchdog" // Jellystics tracker
	// SourceJellysticsPlugin: recorded live by the Jellystics plugin (the
	// plugin sends source "plugin"; it is renamed so it never mixes with the
	// legacy Playback Reporting import below).
	SourceJellysticsPlugin = "jellystics-plugin"
	// SourceLegacyPBR: legacy Jellystics import of the official Playback
	// Reporting plugin (always imported = true).
	SourceLegacyPBR = "plugin"
	// Imports made by the Jellystics plugin (imported = true).
	SourceImportPBR        = "import-pbr"        // official Playback Reporting DB
	SourceImportJellystics = "import-jellystics" // Jellystics history pushed to the plugin, coming back
)

// Overlap slacks. Plugin times come from the Jellyfin server clock, tracker
// times from the Jellystics clock: the 10 s tracker slack (which also absorbs
// the tracker's start estimate and poll granularity) assumes both machines
// are NTP-synchronised; a larger skew breaks tracker/plugin dedupe.
const (
	trackerOverlapSlack = "interval '10 seconds'"
	importOverlapSlack  = "interval '2 minutes'" // imported plays have imprecise times
)

// UpsertResult counts the plays written by UpsertPlays.
type UpsertResult struct {
	Plugin   int // Jellystics-plugin plays
	Imported int // import-pbr / import-jellystics plays inserted or updated
}

// SourceVideo is the source video stream of an item, as stored in
// jf_item_info."MediaStreams" (fallback when the plugin does not send it).
type SourceVideo struct {
	Width, Height              *int
	VideoRange, VideoRangeType *string
}

// PluginSyncRepository stores the Jellystics plugin sync state and data.
type PluginSyncRepository interface {
	GetState(ctx context.Context) (*models.PluginSyncState, error)
	SaveState(ctx context.Context, st *models.PluginSyncState) error
	// ResetCursors makes the next sync re-read every plugin table (after a
	// restore or a plugin reinstall). Upserts are idempotent.
	ResetCursors(ctx context.Context) error

	// UpsertPlays writes mapped plugin plays, following their source:
	// Jellystics-plugin plays are upserted first (authoritative); then
	// Playback Reporting imports replace the legacy import of the same plays
	// and are inserted unless a non-imported play covers them; Jellystics
	// history coming back is only inserted when missing.
	UpsertPlays(ctx context.Context, acts []models.JFPlaybackActivity) (UpsertResult, error)
	// DedupeTrackerRows deletes tracker rows covered by the given plugin
	// plays (same user + item + device, overlapping in time).
	DedupeTrackerRows(ctx context.Context, pluginIds []string) (int64, error)
	// DedupeRecentTrackerRows does the same against every plugin play started
	// after since (closes the race between a tracker write and a sync).
	DedupeRecentTrackerRows(ctx context.Context, since time.Time) (int64, error)
	// FilterTrackerPlays resolves conflicts before the tracker records plays
	// while the plugin is not healthy: a play covered by a closed plugin play
	// (ending no later than it) is dropped; overlapping open plugin plays
	// (stale checkpoints of a plugin that went away) are deleted so the
	// tracker play replaces them. When the plugin comes back, it re-sends the
	// closed version, which then replaces the tracker row.
	FilterTrackerPlays(ctx context.Context, acts []models.JFPlaybackActivity) ([]models.JFPlaybackActivity, error)
	// CloseStalePluginPlays marks open plugin plays whose last checkpoint is
	// older than before as closed.
	CloseStalePluginPlays(ctx context.Context, before time.Time) (int64, error)
	// ItemSourceVideo returns the source video stream of the given items.
	ItemSourceVideo(ctx context.Context, itemIds []string) (map[string]SourceVideo, error)

	UpsertLibraryEvents(ctx context.Context, rows []models.JFLibraryEvent) error
	UpsertUserDataEvents(ctx context.Context, rows []models.JFUserDataEvent) error
	UpsertLogins(ctx context.Context, rows []models.JFUserLogin) error
}

type pluginSyncRepo struct{ db *gorm.DB }

func (r *pluginSyncRepo) GetState(ctx context.Context) (*models.PluginSyncState, error) {
	st := models.PluginSyncState{ID: 1}
	err := r.db.WithContext(ctx).Where("id = 1").FirstOrCreate(&st).Error
	return &st, err
}

func (r *pluginSyncRepo) SaveState(ctx context.Context, st *models.PluginSyncState) error {
	st.ID = 1
	return r.db.WithContext(ctx).Save(st).Error
}

func (r *pluginSyncRepo) ResetCursors(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE plugin_sync_state SET cursor_plays = 0, cursor_library_events = 0,
			cursor_user_data_events = 0, cursor_logins = 0`).Error
}

// importTmp is the per-transaction staging table of imported plays.
const importTmp = "tmp_plugin_imports"

func (r *pluginSyncRepo) UpsertPlays(ctx context.Context, acts []models.JFPlaybackActivity) (UpsertResult, error) {
	var res UpsertResult
	var plugin, imports []models.JFPlaybackActivity
	for _, a := range acts {
		switch a.Source {
		case SourceImportPBR, SourceImportJellystics:
			imports = append(imports, a)
		default:
			plugin = append(plugin, a)
		}
	}
	if len(acts) == 0 {
		return res, nil
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(plugin) > 0 {
			if err := (&playbackRepo{tx}).Upsert(ctx, plugin); err != nil {
				return err
			}
			res.Plugin = len(plugin)
		}
		if len(imports) == 0 {
			return nil
		}
		if err := tx.Exec(`CREATE TEMP TABLE ` + importTmp + ` (LIKE jf_playback_activity INCLUDING DEFAULTS) ON COMMIT DROP`).Error; err != nil {
			return err
		}
		if err := tx.Table(importTmp).CreateInBatches(&imports, 500).Error; err != nil {
			return err
		}
		// The legacy Jellystics import of the same Playback Reporting rows is
		// superseded by the plugin's repaired ones.
		if err := tx.Exec(`
			DELETE FROM jf_playback_activity a USING `+importTmp+` t
			WHERE t.source = ? AND a.source = ? AND a.imported
			  AND (
				(t."Id" LIKE 'pbr-%' AND a."Id" = substr(t."Id", 5))
				OR (a."UserId" = t."UserId" AND a."ItemId" = t."ItemId"
				    AND a."StartedAt" <= COALESCE(t."EndedAt", t."StartedAt") + `+importOverlapSlack+`
				    AND COALESCE(a."EndedAt", a."StartedAt") >= t."StartedAt" - `+importOverlapSlack+`)
			  )`, SourceImportPBR, SourceLegacyPBR).Error; err != nil {
			return err
		}
		updates := make([]string, 0, len(playbackUpsertColumns))
		for _, c := range playbackUpsertColumns {
			q := c
			if c != "source" && c != "imported" {
				q = `"` + c + `"`
			}
			updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", q, q))
		}
		ins := tx.Exec(`
			INSERT INTO jf_playback_activity SELECT t.* FROM `+importTmp+` t
			WHERE t.source = ? AND NOT EXISTS (
				SELECT 1 FROM jf_playback_activity a
				WHERE NOT COALESCE(a.imported, false) AND a."UserId" = t."UserId" AND a."ItemId" = t."ItemId"
				  AND a."StartedAt" <= COALESCE(t."EndedAt", t."StartedAt") + `+importOverlapSlack+`
				  AND COALESCE(a."EndedAt", a."StartedAt") >= t."StartedAt" - `+importOverlapSlack+`)
			ON CONFLICT ("Id") DO UPDATE SET `+strings.Join(updates, ", "), SourceImportPBR)
		if ins.Error != nil {
			return ins.Error
		}
		res.Imported += int(ins.RowsAffected)
		ins = tx.Exec(`
			INSERT INTO jf_playback_activity SELECT t.* FROM `+importTmp+` t
			WHERE t.source = ? ON CONFLICT ("Id") DO NOTHING`, SourceImportJellystics)
		if ins.Error != nil {
			return ins.Error
		}
		res.Imported += int(ins.RowsAffected)
		return nil
	})
	return res, err
}

// dedupeSQL deletes tracker rows t covered by plugin rows p (p filtered by
// the extra condition).
const dedupeSQL = `
	DELETE FROM jf_playback_activity t
	USING jf_playback_activity p
	WHERE p.source = @plugin AND t.source = @tracker AND NOT COALESCE(t.imported, false)
	  AND t."UserId" = p."UserId" AND t."ItemId" = p."ItemId"
	  AND t."DeviceId" IS NOT DISTINCT FROM p."DeviceId"
	  AND t."StartedAt" < COALESCE(p."EndedAt", p."StartedAt") + ` + trackerOverlapSlack + `
	  AND COALESCE(t."EndedAt", t."StartedAt") > p."StartedAt" - ` + trackerOverlapSlack

func (r *pluginSyncRepo) DedupeTrackerRows(ctx context.Context, pluginIds []string) (int64, error) {
	var total int64
	for start := 0; start < len(pluginIds); start += 500 {
		end := min(start+500, len(pluginIds))
		res := r.db.WithContext(ctx).Exec(dedupeSQL+` AND p."Id" IN @ids`, map[string]any{
			"plugin": SourceJellysticsPlugin, "tracker": SourceTracker, "ids": pluginIds[start:end]})
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
	}
	return total, nil
}

func (r *pluginSyncRepo) DedupeRecentTrackerRows(ctx context.Context, since time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Exec(dedupeSQL+` AND p."StartedAt" > @since`, map[string]any{
		"plugin": SourceJellysticsPlugin, "tracker": SourceTracker, "since": since})
	return res.RowsAffected, res.Error
}

func (r *pluginSyncRepo) FilterTrackerPlays(ctx context.Context, acts []models.JFPlaybackActivity) ([]models.JFPlaybackActivity, error) {
	out := acts[:0:0]
	for _, a := range acts {
		if a.UserId == nil || a.ItemId == nil || a.StartedAt == nil {
			out = append(out, a)
			continue
		}
		end := time.Now()
		if a.EndedAt != nil {
			end = *a.EndedAt
		}
		var rows []struct {
			Id         string
			PluginOpen *bool
			Covers     bool // the plugin play ends no earlier than the tracker play (- slack)
		}
		err := r.db.WithContext(ctx).Raw(`
			SELECT "Id" AS id, "PluginOpen" AS plugin_open,
			       CAST(@end AS timestamptz) <= COALESCE("EndedAt", "StartedAt") + `+trackerOverlapSlack+` AS covers
			FROM jf_playback_activity
			WHERE source = @plugin AND "UserId" = @user AND "ItemId" = @item AND "DeviceId" IS NOT DISTINCT FROM @device
			  AND "StartedAt" < CAST(@end AS timestamptz) + `+trackerOverlapSlack+`
			  AND COALESCE("EndedAt", "StartedAt") > CAST(@start AS timestamptz) - `+trackerOverlapSlack,
			map[string]any{"plugin": SourceJellysticsPlugin, "user": *a.UserId, "item": *a.ItemId,
				"device": a.DeviceId, "start": *a.StartedAt, "end": end}).Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		var open []string
		covered := false
		for _, p := range rows {
			switch {
			case p.PluginOpen != nil && *p.PluginOpen:
				open = append(open, p.Id)
			case p.Covers:
				covered = true
			}
		}
		if covered {
			continue
		}
		if len(open) > 0 {
			if err := r.db.WithContext(ctx).Exec(`DELETE FROM jf_playback_activity WHERE "Id" IN ?`, open).Error; err != nil {
				return nil, err
			}
		}
		out = append(out, a)
	}
	return out, nil
}

func (r *pluginSyncRepo) CloseStalePluginPlays(ctx context.Context, before time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE jf_playback_activity SET "PluginOpen" = false
		WHERE source = ? AND "PluginOpen" AND COALESCE("EndedAt", "StartedAt") < ?`,
		SourceJellysticsPlugin, before)
	return res.RowsAffected, res.Error
}

func (r *pluginSyncRepo) ItemSourceVideo(ctx context.Context, itemIds []string) (map[string]SourceVideo, error) {
	out := map[string]SourceVideo{}
	if len(itemIds) == 0 {
		return out, nil
	}
	var rows []struct {
		Id           string
		MediaStreams []byte
	}
	if err := r.db.WithContext(ctx).Raw(`SELECT "Id" AS id, "MediaStreams" AS media_streams FROM jf_item_info WHERE "Id" IN ?`, itemIds).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	type stream struct {
		Type           string
		Index          int
		IsExternal     bool
		IsDefault      bool
		Width, Height  *int
		VideoRange     *string
		VideoRangeType *string
	}
	for _, row := range rows {
		var streams []stream
		if json.Unmarshal(row.MediaStreams, &streams) != nil {
			continue
		}
		var best *stream
		for i := range streams {
			s := &streams[i]
			if s.Type != "Video" {
				continue
			}
			if best == nil || (best.IsExternal && !s.IsExternal) ||
				(best.IsExternal == s.IsExternal && ((s.IsDefault && !best.IsDefault) || (s.IsDefault == best.IsDefault && s.Index < best.Index))) {
				best = s
			}
		}
		if best != nil {
			out[row.Id] = SourceVideo{Width: best.Width, Height: best.Height, VideoRange: best.VideoRange, VideoRangeType: best.VideoRangeType}
		}
	}
	return out, nil
}

func (r *pluginSyncRepo) UpsertLibraryEvents(ctx context.Context, rows []models.JFLibraryEvent) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "Id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"Kind", "At", "ItemId", "ItemType", "ItemName", "SeriesId", "AlbumId", "LibraryId",
			"RunTimeTicks", "SizeBytes", "Version",
		}),
	}).CreateInBatches(rows, 500).Error
}

func (r *pluginSyncRepo) UpsertUserDataEvents(ctx context.Context, rows []models.JFUserDataEvent) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "Id"}},
		DoUpdates: clause.AssignmentColumns([]string{"At", "UserId", "ItemId", "Kind", "Reason", "Version"}),
	}).CreateInBatches(rows, 500).Error
}

func (r *pluginSyncRepo) UpsertLogins(ctx context.Context, rows []models.JFUserLogin) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "Id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"At", "UserId", "UserName", "DeviceId", "DeviceName", "Client", "ApplicationVersion",
			"RemoteEndPoint", "Success", "Version",
		}),
	}).CreateInBatches(rows, 500).Error
}
