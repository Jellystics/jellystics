package repository

import (
	"context"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Play sources written by the Jellystics plugin sync (jf_playback_activity.source).
const (
	SourceTracker          = "watchdog"          // Jellystics tracker
	SourcePlugin           = "plugin"            // recorded live by the Jellystics plugin
	SourceImportPBR        = "import-pbr"        // plugin import of the official Playback Reporting DB
	SourceImportJellystics = "import-jellystics" // Jellystics history pushed to the plugin, coming back
)

// PluginPlayOpen is the PlayState."State" value of a plugin play still in
// progress (the plugin checkpoints open plays every 15 s).
const PluginPlayOpen = "open"

// trackerOverlapSlack absorbs the tracker's start estimate (head credit) and
// poll granularity when matching a tracker row with a plugin play.
const trackerOverlapSlack = "interval '10 seconds'"

// PluginSyncRepository stores the Jellystics plugin sync state and data.
type PluginSyncRepository interface {
	GetState(ctx context.Context) (*models.PluginSyncState, error)
	SaveState(ctx context.Context, st *models.PluginSyncState) error

	// UpsertPlays writes plays mapped from the plugin, following their source:
	// plugin plays are upserted (the plugin is authoritative); Jellystics
	// history coming back is only inserted when missing; Playback Reporting
	// imports replace the legacy import of the same row and are skipped when
	// a non-imported play already covers them. Returns the rows written.
	UpsertPlays(ctx context.Context, acts []models.JFPlaybackActivity) (int, error)
	// DedupeTrackerRows deletes tracker rows covered by the given plugin plays
	// (same user + item + device, overlapping in time).
	DedupeTrackerRows(ctx context.Context, pluginIds []string) (int64, error)
	// FilterTrackerPlays resolves conflicts before the tracker records plays
	// while the plugin is not healthy: a play covered by a closed plugin play
	// is dropped (the plugin is authoritative); overlapping open plugin plays
	// (stale checkpoints of a plugin that went away) are deleted so the
	// tracker play replaces them. When the plugin comes back, it re-sends the
	// closed version, which then replaces the tracker row.
	FilterTrackerPlays(ctx context.Context, acts []models.JFPlaybackActivity) ([]models.JFPlaybackActivity, error)

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

func (r *pluginSyncRepo) UpsertPlays(ctx context.Context, acts []models.JFPlaybackActivity) (int, error) {
	if len(acts) == 0 {
		return 0, nil
	}
	written := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var upsert, insertOnly []models.JFPlaybackActivity
		for _, a := range acts {
			switch a.Source {
			case SourceImportJellystics:
				insertOnly = append(insertOnly, a)
			case SourceImportPBR:
				covered, err := coveredByNonImported(tx, a)
				if err != nil {
					return err
				}
				if covered {
					continue
				}
				// The legacy Jellystics import of the same Playback Reporting
				// row (Id = rowid) is superseded by the plugin's repaired one.
				if rowid := strings.TrimPrefix(a.Id, "pbr-"); rowid != a.Id {
					if err := tx.Exec(`DELETE FROM jf_playback_activity WHERE "Id" = ? AND source = ? AND imported`,
						rowid, SourcePlugin).Error; err != nil {
						return err
					}
				}
				upsert = append(upsert, a)
			default:
				upsert = append(upsert, a)
			}
		}
		if len(upsert) > 0 {
			if err := (&playbackRepo{tx}).Upsert(ctx, upsert); err != nil {
				return err
			}
			written += len(upsert)
		}
		if len(insertOnly) > 0 {
			res := tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(insertOnly, 500)
			if res.Error != nil {
				return res.Error
			}
			written += int(res.RowsAffected)
		}
		return nil
	})
	return written, err
}

// coveredByNonImported reports whether a non-imported play (tracker or
// plugin) of the same user and item overlaps a (less precise) imported play.
func coveredByNonImported(tx *gorm.DB, a models.JFPlaybackActivity) (bool, error) {
	if a.UserId == nil || a.ItemId == nil || a.StartedAt == nil {
		return false, nil
	}
	end := *a.StartedAt
	if a.EndedAt != nil {
		end = *a.EndedAt
	}
	var n int64
	err := tx.Raw(`
		SELECT COUNT(*) FROM jf_playback_activity
		WHERE NOT COALESCE(imported, false) AND "UserId" = ? AND "ItemId" = ?
		  AND "StartedAt" <= CAST(? AS timestamptz) + interval '2 minutes'
		  AND COALESCE("EndedAt", "StartedAt") >= CAST(? AS timestamptz) - interval '2 minutes'`,
		*a.UserId, *a.ItemId, end, *a.StartedAt).Scan(&n).Error
	return n > 0, err
}

func (r *pluginSyncRepo) DedupeTrackerRows(ctx context.Context, pluginIds []string) (int64, error) {
	if len(pluginIds) == 0 {
		return 0, nil
	}
	var total int64
	for start := 0; start < len(pluginIds); start += 500 {
		end := min(start+500, len(pluginIds))
		res := r.db.WithContext(ctx).Exec(`
			DELETE FROM jf_playback_activity t
			USING jf_playback_activity p
			WHERE p."Id" IN ? AND p.source = ? AND NOT COALESCE(p.imported, false)
			  AND t.source = ? AND NOT COALESCE(t.imported, false)
			  AND t."Id" <> p."Id"
			  AND t."UserId" = p."UserId" AND t."ItemId" = p."ItemId"
			  AND t."DeviceId" IS NOT DISTINCT FROM p."DeviceId"
			  AND t."StartedAt" < COALESCE(p."EndedAt", p."StartedAt") + `+trackerOverlapSlack+`
			  AND COALESCE(t."EndedAt", t."StartedAt") > p."StartedAt" - `+trackerOverlapSlack,
			pluginIds[start:end], SourcePlugin, SourceTracker)
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
	}
	return total, nil
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
			Id    string
			State *string
		}
		err := r.db.WithContext(ctx).Raw(`
			SELECT "Id", "PlayState"->>'State' AS state FROM jf_playback_activity
			WHERE source = ? AND NOT COALESCE(imported, false)
			  AND "UserId" = ? AND "ItemId" = ? AND "DeviceId" IS NOT DISTINCT FROM ?
			  AND "StartedAt" < CAST(? AS timestamptz) + `+trackerOverlapSlack+`
			  AND COALESCE("EndedAt", "StartedAt") > CAST(? AS timestamptz) - `+trackerOverlapSlack,
			SourcePlugin, *a.UserId, *a.ItemId, a.DeviceId, end, *a.StartedAt).Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		var open []string
		closed := false
		for _, p := range rows {
			if p.State != nil && *p.State == PluginPlayOpen {
				open = append(open, p.Id)
			} else {
				closed = true
			}
		}
		if closed {
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
