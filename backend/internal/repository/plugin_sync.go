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
	// SourceTSVImport: Playback Reporting TSV backup imported by Jellystics
	// (imported = true).
	SourceTSVImport = "import"
	// Imports made by the Jellystics plugin (imported = true).
	SourceImportPBR        = "import-pbr"        // official Playback Reporting DB
	SourceImportJellystics = "import-jellystics" // Jellystics history pushed to the plugin, coming back
)

// Overlap slacks. Plugin times come from the Jellyfin server clock, tracker
// times from the Jellystics clock: the 10 s tracker slack (which also absorbs
// the tracker's start estimate and poll granularity) assumes both machines
// are NTP-synchronised; a larger skew breaks tracker/plugin dedupe.
const trackerOverlapSlack = "interval '10 seconds'"

// RestorePlays restores backup rows: a row whose Id exists is updated (same
// row); a new row that is the same play as an existing row of another source
// is skipped (one play = one row); others are inserted. Only rows existing
// before the restore are compared, so an empty database gets the whole
// backup.
func (r *playbackRepo) RestorePlays(ctx context.Context, rows []models.JFPlaybackActivity) (restored, skipped int, err error) {
	if len(rows) == 0 {
		return 0, 0, nil
	}
	// Last occurrence of a duplicated Id wins (one ON CONFLICT per row).
	pos := map[string]int{}
	uniq := make([]models.JFPlaybackActivity, 0, len(rows))
	for _, a := range rows {
		if i, ok := pos[a.Id]; ok {
			uniq[i] = a
			continue
		}
		pos[a.Id] = len(uniq)
		uniq = append(uniq, a)
	}
	const tmp = "tmp_restore_plays"
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`CREATE TEMP TABLE ` + tmp + ` (LIKE jf_playback_activity INCLUDING DEFAULTS) ON COMMIT DROP`).Error; err != nil {
			return err
		}
		if err := tx.Table(tmp).CreateInBatches(&uniq, 500).Error; err != nil {
			return err
		}
		del := tx.Exec(`
			DELETE FROM ` + tmp + ` t
			WHERE NOT EXISTS (SELECT 1 FROM jf_playback_activity x WHERE x."Id" = t."Id")
			  AND EXISTS (SELECT 1 FROM jf_playback_activity a WHERE a.source IS DISTINCT FROM t.source AND ` + samePlay("a", "t") + `)`)
		if del.Error != nil {
			return del.Error
		}
		skipped = int(del.RowsAffected)
		ins := tx.Exec(`INSERT INTO jf_playback_activity SELECT t.* FROM ` + tmp + ` t
			ON CONFLICT ("Id") DO UPDATE SET ` + strings.Join(upsertSet(false), ", "))
		if ins.Error != nil {
			return ins.Error
		}
		restored = int(ins.RowsAffected)
		return nil
	})
	return restored, skipped, err
}

// upsertSet is the ON CONFLICT update list of playbackUpsertColumns;
// stickyRepaired keeps DurationRepaired once set.
func upsertSet(stickyRepaired bool) []string {
	updates := make([]string, 0, len(playbackUpsertColumns))
	for _, c := range playbackUpsertColumns {
		q := c
		if c != "source" && c != "imported" {
			q = `"` + c + `"`
		}
		if c == "DurationRepaired" && stickyRepaired {
			updates = append(updates, `"DurationRepaired" = jf_playback_activity."DurationRepaired" OR EXCLUDED."DurationRepaired"`)
			continue
		}
		updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", q, q))
	}
	return updates
}

func (r *playbackRepo) DeleteCoveredTSVImports(ctx context.Context, ids []string) (int64, error) {
	var total int64
	for start := 0; start < len(ids); start += 500 {
		end := min(start+500, len(ids))
		res := r.db.WithContext(ctx).Exec(`
			DELETE FROM jf_playback_activity t
			WHERE t.source = @tsv AND t."Id" IN @ids AND t."StartedAt" IS NOT NULL
			  AND EXISTS (
				SELECT 1 FROM jf_playback_activity a
				WHERE a.source <> @tsv AND `+samePlay("a", "t")+`)`,
			map[string]any{"tsv": SourceTSVImport, "ids": ids[start:end]})
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
	}
	return total, nil
}

// SamePlayCond is THE "same play" rule, shared by every dedupe / guard path
// (plugin imports, history gap-fill, TSV and legacy Playback Reporting
// imports, backup restore, tracker / plugin reconciliation): same user, same
// item, start times within ±2 minutes and, when both durations are known
// (> 0), similar durations: |d1 − d2| ≤ max(60 s, 20 % of the longer).
// A mere time overlap is not enough: a track replayed back to back (A
// 0:00–3:00, B 3:00–6:00) is two plays.
//
// Arguments are SQL expressions: user, item, start (timestamptz) and duration
// (seconds) of each side.
func SamePlayCond(userA, itemA, startA, durA, userB, itemB, startB, durB string) string {
	return `(` + userA + ` = ` + userB + ` AND ` + itemA + ` = ` + itemB +
		` AND ` + startA + ` BETWEEN ` + startB + ` - ` + samePlayStartSlack + ` AND ` + startB + ` + ` + samePlayStartSlack +
		` AND abs(extract(epoch FROM ` + startA + ` - ` + startB + `)) <= ` + samePlayWindow(durA, durB) +
		` AND (COALESCE(` + durA + `, 0) <= 0 OR COALESCE(` + durB + `, 0) <= 0` +
		` OR abs(` + durA + ` - ` + durB + `) <= GREATEST(60, 0.2 * GREATEST(` + durA + `, ` + durB + `))))`
}

// samePlayWindow is the start-time window in seconds: min(120, max(10, 50 %
// of the shorter duration)) when both durations are known (> 0), so a short
// track replayed back to back (90 s: A 0:00–1:30, B 1:30–3:00) stays two
// plays; 120 s when a duration is unknown.
func samePlayWindow(durA, durB string) string {
	return `(CASE WHEN COALESCE(` + durA + `, 0) > 0 AND COALESCE(` + durB + `, 0) > 0` +
		` THEN LEAST(120, GREATEST(10, 0.5 * LEAST(` + durA + `, ` + durB + `))) ELSE 120 END)`
}

// samePlayStartSlack: start times of the same play recorded by two sources
// (imported starts are estimates: Playback Reporting writes ~20 s late).
const samePlayStartSlack = "interval '2 minutes'"

// samePlay is SamePlayCond for two jf_playback_activity aliases.
func samePlay(a, b string) string {
	col := func(t, c string) string { return t + `."` + c + `"` }
	return SamePlayCond(col(a, "UserId"), col(a, "ItemId"), col(a, "StartedAt"), col(a, "PlaybackDuration"),
		col(b, "UserId"), col(b, "ItemId"), col(b, "StartedAt"), col(b, "PlaybackDuration"))
}

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

	// SavePushState writes only the history push columns of the state
	// (SaveState never writes them).
	SavePushState(ctx context.Context, st *models.PluginSyncState) error
	// CountPushable / ListPushable select the plays the history push sends
	// to the plugin (everything not coming from the plugin), in ("StartedAt",
	// "Id") order after the given cursor (nil afterAt = from the start).
	CountPushable(ctx context.Context) (int64, error)
	ListPushable(ctx context.Context, afterAt *time.Time, afterId string, limit int) ([]models.JFPlaybackActivity, error)
	// CountPluginPlays counts the plays recorded by the Jellystics plugin.
	CountPluginPlays(ctx context.Context) (int64, error)

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

// pushColumns are owned by the history push (SavePushState): the sync never
// writes them, so a concurrent sync run and push cannot overwrite each other.
var pushColumns = []string{"push_state", "push_cursor", "push_cursor_at", "push_total", "push_sent", "push_imported",
	"push_skipped", "push_rejected", "push_started_at", "push_finished_at", "push_error"}

func (r *pluginSyncRepo) SaveState(ctx context.Context, st *models.PluginSyncState) error {
	st.ID = 1
	return r.db.WithContext(ctx).Model(&models.PluginSyncState{ID: 1}).Select("*").Omit(append([]string{"id"}, pushColumns...)...).Updates(st).Error
}

func (r *pluginSyncRepo) SavePushState(ctx context.Context, st *models.PluginSyncState) error {
	cols := make([]string, len(pushColumns))
	copy(cols, pushColumns)
	return r.db.WithContext(ctx).Model(&models.PluginSyncState{ID: 1}).Select(cols).Updates(st).Error
}

// pushableCond selects the plays the history push sends to the plugin: every
// play not coming from the plugin itself (tracker, legacy / TSV imports,
// restores...), of a type the plugin stores, with the identity it requires.
const pushableCond = `source NOT IN ('` + SourceJellysticsPlugin + `', '` + SourceImportPBR + `', '` + SourceImportJellystics + `')
	AND "MediaType" IN ('Movie', 'Episode', 'Audio') AND "ItemId" IS NOT NULL AND "UserId" IS NOT NULL AND "StartedAt" IS NOT NULL`

func (r *pluginSyncRepo) CountPushable(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.JFPlaybackActivity{}).Where(pushableCond).Count(&n).Error
	return n, err
}

func (r *pluginSyncRepo) ListPushable(ctx context.Context, afterAt *time.Time, afterId string, limit int) ([]models.JFPlaybackActivity, error) {
	var out []models.JFPlaybackActivity
	q := r.db.WithContext(ctx).Where(pushableCond)
	if afterAt != nil {
		q = q.Where(`("StartedAt", "Id") > (?, ?)`, *afterAt, afterId)
	}
	err := q.Order(`"StartedAt", "Id"`).Limit(limit).Find(&out).Error
	return out, err
}

func (r *pluginSyncRepo) CountPluginPlays(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.JFPlaybackActivity{}).Where("source = ?", SourceJellysticsPlugin).Count(&n).Error
	return n, err
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
		// Jellystics' own imports of the same Playback Reporting rows (legacy
		// sync, TSV backup) are superseded by the plugin's: one play = one
		// row. A duration Jellystics had repaired keeps its flag.
		if err := tx.Exec(`
			WITH gone AS (
				DELETE FROM jf_playback_activity a USING `+importTmp+` t
				WHERE t.source = @pbr AND a.source IN (@legacy, @tsv) AND a.imported
				  AND (
					(t."Id" LIKE 'pbr-%' AND a."Id" IN (substr(t."Id", 5), 'plugin-' || substr(t."Id", 5)))
					OR `+samePlay("a", "t")+`
				  )
				RETURNING t."Id" AS tid, a."DurationRepaired" AS repaired
			)
			UPDATE `+importTmp+` t SET "DurationRepaired" = true FROM gone WHERE gone.tid = t."Id" AND gone.repaired`,
			map[string]any{"pbr": SourceImportPBR, "legacy": SourceLegacyPBR, "tsv": SourceTSVImport}).Error; err != nil {
			return err
		}
		// Sticky repaired flag: once repaired (by Jellystics or the plugin),
		// a re-read of the same play does not clear it.
		updates := upsertSet(true)
		// A PBR import is skipped when another row already records the play
		// (tracker, plugin, restored or Jellystics history); other PBR rows
		// do not block it (two real plays can be close).
		ins := tx.Exec(`
			INSERT INTO jf_playback_activity SELECT t.* FROM `+importTmp+` t
			WHERE t.source = @pbr AND NOT EXISTS (
				SELECT 1 FROM jf_playback_activity a
				WHERE a.source <> @pbr AND NOT (COALESCE(a.imported, false) AND a.source IN (@legacy, @tsv))
				  AND `+samePlay("a", "t")+`)
			ON CONFLICT ("Id") DO UPDATE SET `+strings.Join(updates, ", "),
			map[string]any{"pbr": SourceImportPBR, "legacy": SourceLegacyPBR, "tsv": SourceTSVImport})
		if ins.Error != nil {
			return ins.Error
		}
		res.Imported += int(ins.RowsAffected)
		// Jellystics history coming back from the plugin only fills genuine
		// gaps: never when its own Id exists or a row of another source
		// records the same play (a pushed row since replaced by a plugin play
		// or a PBR import must not come back). Other import-jellystics rows
		// never block it: the plugin already dedupes its own.
		ins = tx.Exec(`
			INSERT INTO jf_playback_activity SELECT t.* FROM `+importTmp+` t
			WHERE t.source = @js AND NOT EXISTS (
				SELECT 1 FROM jf_playback_activity a
				WHERE a."Id" = t."Id" OR (a.source <> @js AND `+samePlay("a", "t")+`))
			ON CONFLICT ("Id") DO NOTHING`, map[string]any{"js": SourceImportJellystics})
		if ins.Error != nil {
			return ins.Error
		}
		res.Imported += int(ins.RowsAffected)
		return nil
	})
	return res, err
}

// trackerInside: tracker row t lies within plugin row p (10 s slack): the
// tracker took over in the middle of a play the plugin recorded whole.
func trackerInside(t, p string) string {
	return t + `."StartedAt" >= ` + p + `."StartedAt" - ` + trackerOverlapSlack +
		` AND COALESCE(` + t + `."EndedAt", ` + t + `."StartedAt") <= COALESCE(` + p + `."EndedAt", ` + p + `."StartedAt") + ` + trackerOverlapSlack
}

// dedupeSQL deletes tracker rows t that are the same play as plugin rows p
// on the same device (the shared rule, or a partial tracker row inside the
// plugin play); p is filtered by the extra condition.
var dedupeSQL = `
	DELETE FROM jf_playback_activity t
	USING jf_playback_activity p
	WHERE p.source = @plugin AND t.source = @tracker AND NOT COALESCE(t.imported, false)
	  AND t."UserId" = p."UserId" AND t."ItemId" = p."ItemId"
	  AND t."DeviceId" IS NOT DISTINCT FROM p."DeviceId"
	  AND (` + samePlay("t", "p") + ` OR (` + trackerInside("t", "p") + `))`

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
			  AND COALESCE("EndedAt", "StartedAt") > CAST(@start AS timestamptz) - `+trackerOverlapSlack+`
			  -- Same play, not just adjacent ones (back-to-back replays): starts
			  -- within the shared same-play window, or the tracker play inside
			  -- the plugin play. Durations are not compared: open plugin plays
			  -- are partial.
			  AND (abs(extract(epoch FROM "StartedAt" - CAST(@start AS timestamptz))) <= CASE
			         -- an open checkpoint's duration is partial: scale on the tracker play only
			         WHEN COALESCE("PluginOpen", false) THEN `+samePlayWindow(`CAST(@dur AS bigint)`, `CAST(@dur AS bigint)`)+`
			         ELSE `+samePlayWindow(`"PlaybackDuration"`, `CAST(@dur AS bigint)`)+` END
			       OR (CAST(@start AS timestamptz) >= "StartedAt" - `+trackerOverlapSlack+`
			           AND CAST(@end AS timestamptz) <= COALESCE("EndedAt", "StartedAt") + `+trackerOverlapSlack+`))`,
			map[string]any{"plugin": SourceJellysticsPlugin, "user": *a.UserId, "item": *a.ItemId,
				"device": a.DeviceId, "start": *a.StartedAt, "end": end, "dur": a.PlaybackDuration}).Scan(&rows).Error
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
