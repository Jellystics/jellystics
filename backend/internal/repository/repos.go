package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/statsql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---- config ----

type configRepo struct{ db *gorm.DB }

func (r *configRepo) Get(ctx context.Context) (*models.AppConfig, error) {
	var cfg models.AppConfig
	if err := r.db.WithContext(ctx).First(&cfg, 1).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *configRepo) Save(ctx context.Context, cfg *models.AppConfig) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}

// ---- users ----

type userRepo struct{ db *gorm.DB }

func (r *userRepo) Upsert(ctx context.Context, users []models.JFUser) error {
	if len(users) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns([]string{"Name", "PrimaryImageTag", "LastLoginDate", "LastActivityDate", "IsAdministrator"}),
		}).
		CreateInBatches(users, 500).Error
}

func (r *userRepo) List(ctx context.Context) ([]models.JFUser, error) {
	var out []models.JFUser
	return out, r.db.WithContext(ctx).Find(&out).Error
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*models.JFUser, error) {
	var u models.JFUser
	err := r.db.WithContext(ctx).Where(`"Id" = ?`, id).First(&u).Error
	return &u, err
}

// ---- libraries ----

type libraryRepo struct{ db *gorm.DB }

func (r *libraryRepo) Upsert(ctx context.Context, libs []models.JFLibrary) error {
	if len(libs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns([]string{"Name", "ServerId", "IsFolder", "Type", "CollectionType", "ImageTagsPrimary", "archived"}),
		}).
		CreateInBatches(libs, 500).Error
}

func (r *libraryRepo) List(ctx context.Context) ([]models.JFLibrary, error) {
	var out []models.JFLibrary
	return out, r.db.WithContext(ctx).Where("archived = false").Find(&out).Error
}

func (r *libraryRepo) GetByID(ctx context.Context, id string) (*models.JFLibrary, error) {
	var lib models.JFLibrary
	err := r.db.WithContext(ctx).Where(`"Id" = ?`, id).First(&lib).Error
	return &lib, err
}

func (r *libraryRepo) ArchiveNotIn(ctx context.Context, ids []string) error {
	return r.db.WithContext(ctx).
		Model(&models.JFLibrary{}).
		Where(`"Id" NOT IN ?`, ids).
		Update("archived", true).Error
}

// ---- items ----

type itemRepo struct{ db *gorm.DB }

var itemUpdateCols = []string{
	"Name", "ServerId", "PremiereDate", "DateCreated", "EndDate",
	"CommunityRating", "RunTimeTicks", "ProductionYear", "IsFolder",
	"Type", "Status", "ImageTagsPrimary", "ImageTagsBanner", "ImageTagsLogo",
	"ImageTagsThumb", "BackdropImageTags", "ParentId", "PrimaryImageHash",
	"archived", "Genres", "AlbumArtist", "ArtistId",
}

func (r *itemRepo) Upsert(ctx context.Context, items []models.JFLibraryItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns(itemUpdateCols),
		}).
		CreateInBatches(items, 500).Error
}

func (r *itemRepo) ArchiveNotIn(ctx context.Context, parentId string, ids []string) error {
	return r.db.WithContext(ctx).
		Model(&models.JFLibraryItem{}).
		Where(`"ParentId" = ? AND "Id" NOT IN ?`, parentId, ids).
		Update("archived", true).Error
}

func (r *itemRepo) ListByParent(ctx context.Context, parentId string) ([]models.JFLibraryItem, error) {
	var out []models.JFLibraryItem
	return out, r.db.WithContext(ctx).
		Where(`"ParentId" = ? AND archived = false`, parentId).
		Find(&out).Error
}

func (r *itemRepo) GetByID(ctx context.Context, id string) (*models.JFLibraryItem, error) {
	var item models.JFLibraryItem
	err := r.db.WithContext(ctx).Where(`"Id" = ?`, id).First(&item).Error
	return &item, err
}

// ---- seasons ----

type seasonRepo struct{ db *gorm.DB }

var seasonUpdateCols = []string{
	"Name", "ServerId", "IndexNumber", "Type", "ParentLogoItemId",
	"ParentBackdropItemId", "ParentBackdropImageTags", "SeriesName",
	"SeriesId", "SeriesPrimaryImageTag", "archived",
}

func (r *seasonRepo) Upsert(ctx context.Context, seasons []models.JFLibrarySeason) error {
	if len(seasons) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns(seasonUpdateCols),
		}).
		CreateInBatches(seasons, 500).Error
}

func (r *seasonRepo) ArchiveNotIn(ctx context.Context, seriesId string, ids []string) error {
	return r.db.WithContext(ctx).
		Model(&models.JFLibrarySeason{}).
		Where(`"SeriesId" = ? AND "Id" NOT IN ?`, seriesId, ids).
		Update("archived", true).Error
}

func (r *seasonRepo) ListBySeries(ctx context.Context, seriesId string) ([]models.JFLibrarySeason, error) {
	var out []models.JFLibrarySeason
	return out, r.db.WithContext(ctx).
		Where(`"SeriesId" = ? AND archived = false`, seriesId).
		Find(&out).Error
}

// ---- episodes ----

type episodeRepo struct{ db *gorm.DB }

var episodeUpdateCols = []string{
	"EpisodeId", "Name", "ServerId", "PremiereDate", "DateCreated",
	"OfficialRating", "CommunityRating", "RunTimeTicks", "ProductionYear",
	"IndexNumber", "ParentIndexNumber", "Type", "ParentLogoItemId",
	"ParentBackdropItemId", "ParentBackdropImageTags", "SeriesId", "SeasonId",
	"SeasonName", "SeriesName", "PrimaryImageHash", "archived",
}

func (r *episodeRepo) Upsert(ctx context.Context, eps []models.JFLibraryEpisode) error {
	if len(eps) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns(episodeUpdateCols),
		}).
		CreateInBatches(eps, 500).Error
}

func (r *episodeRepo) ArchiveNotIn(ctx context.Context, seriesId string, ids []string) error {
	return r.db.WithContext(ctx).
		Model(&models.JFLibraryEpisode{}).
		Where(`"SeriesId" = ? AND "Id" NOT IN ?`, seriesId, ids).
		Update("archived", true).Error
}

func (r *episodeRepo) ListBySeries(ctx context.Context, seriesId string) ([]models.JFLibraryEpisode, error) {
	var out []models.JFLibraryEpisode
	return out, r.db.WithContext(ctx).
		Where(`"SeriesId" = ? AND archived = false`, seriesId).
		Find(&out).Error
}

func (r *episodeRepo) ListBySeason(ctx context.Context, seasonId string) ([]models.JFLibraryEpisode, error) {
	var out []models.JFLibraryEpisode
	return out, r.db.WithContext(ctx).
		Where(`"SeasonId" = ? AND archived = false`, seasonId).
		Find(&out).Error
}

// ---- music artists ----

type musicArtistRepo struct{ db *gorm.DB }

var artistUpdateCols = []string{"LibraryId", "Name", "Overview", "ImageTagsPrimary", "Genres", "archived"}

func (r *musicArtistRepo) Upsert(ctx context.Context, artists []models.JFMusicArtist) error {
	if len(artists) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns(artistUpdateCols),
		}).
		CreateInBatches(artists, 500).Error
}

func (r *musicArtistRepo) ArchiveNotIn(ctx context.Context, libraryId string, ids []string) error {
	return r.db.WithContext(ctx).
		Model(&models.JFMusicArtist{}).
		Where(`"LibraryId" = ? AND "Id" NOT IN ?`, libraryId, ids).
		Update("archived", true).Error
}

func (r *musicArtistRepo) ListByLibrary(ctx context.Context, libraryId string) ([]models.JFMusicArtist, error) {
	var out []models.JFMusicArtist
	return out, r.db.WithContext(ctx).
		Where(`"LibraryId" = ? AND archived = false`, libraryId).
		Find(&out).Error
}

// ---- music tracks ----

type musicTrackRepo struct{ db *gorm.DB }

var trackUpdateCols = []string{
	"LibraryId", "AlbumId", "ArtistId", "Name", "AlbumName", "AlbumArtist",
	"IndexNumber", "DiscNumber", "RunTimeTicks", "DateCreated", "ProductionYear",
	"ImageTagsPrimary", "Genres", "archived",
}

func (r *musicTrackRepo) Upsert(ctx context.Context, tracks []models.JFMusicTrack) error {
	if len(tracks) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns(trackUpdateCols),
		}).
		CreateInBatches(tracks, 500).Error
}

func (r *musicTrackRepo) ArchiveNotIn(ctx context.Context, libraryId string, ids []string) error {
	return r.db.WithContext(ctx).
		Model(&models.JFMusicTrack{}).
		Where(`"LibraryId" = ? AND "Id" NOT IN ?`, libraryId, ids).
		Update("archived", true).Error
}

func (r *musicTrackRepo) ListByLibrary(ctx context.Context, libraryId string) ([]models.JFMusicTrack, error) {
	var out []models.JFMusicTrack
	return out, r.db.WithContext(ctx).
		Where(`"LibraryId" = ? AND archived = false`, libraryId).
		Find(&out).Error
}

func (r *musicTrackRepo) ListByAlbum(ctx context.Context, albumId string) ([]models.JFMusicTrack, error) {
	var out []models.JFMusicTrack
	return out, r.db.WithContext(ctx).
		Where(`"AlbumId" = ? AND archived = false`, albumId).
		Order(`"DiscNumber" ASC, "IndexNumber" ASC`).
		Find(&out).Error
}

func (r *musicTrackRepo) ListByArtist(ctx context.Context, artistId string) ([]models.JFMusicTrack, error) {
	var out []models.JFMusicTrack
	return out, r.db.WithContext(ctx).
		Where(`"ArtistId" = ? AND archived = false`, artistId).
		Find(&out).Error
}

// ---- item info ----

type itemInfoRepo struct{ db *gorm.DB }

func (r *itemInfoRepo) Upsert(ctx context.Context, infos []models.JFItemInfo) error {
	if len(infos) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns([]string{"Path", "Name", "Size", "Bitrate", "MediaStreams", "Type"}),
		}).
		CreateInBatches(infos, 500).Error
}

func (r *itemInfoRepo) RemoveOrphaned(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec(`
		DELETE FROM jf_item_info
		WHERE "Id" NOT IN (SELECT "Id" FROM jf_library_items)
		  AND "Id" NOT IN (SELECT "Id" FROM jf_library_episodes)
		  AND "Id" NOT IN (SELECT "Id" FROM jf_music_tracks)
	`).Error
}

// ---- playback ----

// playbackUpsertColumns are the columns refreshed when a play is upserted.
var playbackUpsertColumns = []string{
	"IsPaused", "UserId", "UserName", "Client", "DeviceName", "DeviceId",
	"ApplicationVersion", "NowPlayingItemId", "NowPlayingItemName",
	"EpisodeId", "SeasonId", "SeriesName", "PlaybackDuration", "PlayMethod",
	"ActivityDateInserted", "MediaStreams", "TranscodingInfo", "PlayState",
	"OriginalContainer", "RemoteEndPoint", "ServerId", "source", "imported",
	"ItemId", "MediaType", "RunTimeTicks", "PositionTicksStart", "PositionTicksEnd",
	"PausedSeconds", "StartedAt", "EndedAt", "IsView",
	"AudioStreamIndex", "AudioLanguage", "SubtitleStreamIndex", "SubtitleLanguage",
	"TranscodeReasons", "Bitrate", "HwAccel", "VideoRange", "PauseCount", "IsRemote",
	"PlaySessionId", "SeekCount", "SkippedSeconds", "PlayedToCompletion", "PluginOpen",
	"DurationRepaired",
}

type playbackRepo struct{ db *gorm.DB }

func (r *playbackRepo) Upsert(ctx context.Context, acts []models.JFPlaybackActivity) error {
	if len(acts) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns(playbackUpsertColumns),
		}).
		CreateInBatches(acts, 500).Error
}

func (r *playbackRepo) List(ctx context.Context, limit int) ([]models.JFPlaybackActivity, error) {
	var out []models.JFPlaybackActivity
	q := r.db.WithContext(ctx).Order(`"ActivityDateInserted" DESC`)
	if limit > 0 {
		q = q.Limit(limit)
	}
	return out, q.Find(&out).Error
}

func (r *playbackRepo) ListByUser(ctx context.Context, userId string) ([]models.JFPlaybackActivity, error) {
	var out []models.JFPlaybackActivity
	return out, r.db.WithContext(ctx).
		Where(`"UserId" = ?`, userId).
		Order(`"ActivityDateInserted" DESC`).
		Find(&out).Error
}

// ---- watchdog ----

type watchdogRepo struct{ db *gorm.DB }

func (r *watchdogRepo) Upsert(ctx context.Context, entries []models.JFActivityWatchdog) error {
	if len(entries) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"ActivityId", "IsPaused", "UserId", "UserName", "Client", "DeviceName",
				"DeviceId", "ApplicationVersion", "NowPlayingItemId", "NowPlayingItemName",
				"EpisodeId", "SeasonId", "SeriesName", "PlaybackDuration", "PlayMethod",
				"ActivityDateInserted", "MediaStreams", "TranscodingInfo", "PlayState",
				"OriginalContainer", "RemoteEndPoint", "ServerId",
				"WatchedSeconds", "LastTickAt",
				"ItemId", "MediaType", "RunTimeTicks", "PositionTicksStart", "WatchedMs", "PausedMs",
				"StartedAt", "LastCheckIn", "PeriodicReporter", "LastServerActivity",
				"AudioStreamIndex", "AudioLanguage", "SubtitleStreamIndex", "SubtitleLanguage",
				"TranscodeReasons", "Bitrate", "HwAccel", "VideoRange", "PauseCount", "IsRemote",
			}),
		}).
		CreateInBatches(entries, 500).Error
}

func (r *watchdogRepo) List(ctx context.Context) ([]models.JFActivityWatchdog, error) {
	var out []models.JFActivityWatchdog
	return out, r.db.WithContext(ctx).Find(&out).Error
}

func (r *watchdogRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.JFActivityWatchdog{}, `"Id" = ?`, id).Error
}

// ---- plugin data ----

type pluginDataRepo struct{ db *gorm.DB }

func (r *pluginDataRepo) Upsert(ctx context.Context, rows []models.JFPluginData) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "rowid"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"DateCreated", "UserId", "ItemId", "ItemType", "ItemName",
				"PlaybackMethod", "ClientName", "DeviceName", "PlayDuration", "imported", "source",
			}),
		}).
		CreateInBatches(rows, 500).Error
}

func (r *pluginDataRepo) ListUnimported(ctx context.Context) ([]models.JFPluginData, error) {
	var out []models.JFPluginData
	return out, r.db.WithContext(ctx).Where("imported = false").Find(&out).Error
}

func (r *pluginDataRepo) GetMaxRowId(ctx context.Context) (int64, error) {
	type result struct{ MaxRowId int64 }
	var res result
	err := r.db.WithContext(ctx).Raw(
		`SELECT COALESCE(MAX("rowid"::bigint), 0) AS "MaxRowId" FROM jf_playback_reporting_plugin_data`,
	).Scan(&res).Error
	return res.MaxRowId, err
}

func displayTZ() string {
	name, _ := statsql.Timezone()
	return name
}

// MergeIntoPlaybackActivity imports staged PlaybackReporting rows as plays.
//
//   - Only movies, episodes and music tracks are imported.
//   - The plugin writes its row ~20 s after playback start (DateCreated, in
//     the Jellyfin server's local time, no offset) and keeps updating its
//     duration while the play goes on. Rows younger than a day are therefore
//     not imported yet (their duration may still grow); they are imported by a
//     later run once final.
//   - A plugin row is skipped when the same play is already recorded by the
//     Jellystics tracker (or is being tracked right now): same user and item,
//     and either overlapping in time (TZ-interpreted), or — robust to a
//     missing/wrong TZ setting — starting within ±14 h with a similar duration.
func (r *pluginDataRepo) MergeIntoPlaybackActivity(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec(`
		WITH src AS (
			SELECT
				p.*,
				(p."DateCreated"::timestamp AT TIME ZONE CAST(@tz AS text)) - interval '20 seconds' AS started_at,
				COALESCE(p."PlayDuration", 0) AS dur,
				e."Id"        AS ep_id,
				e."SeriesId"  AS series_id,
				e."SeasonId"  AS season_id,
				e."RunTimeTicks" AS ep_rt,
				s."Name"      AS series_name,
				mt."Id"       AS track_id,
				mt."AlbumId"  AS album_id,
				mt."AlbumName" AS album_name,
				mt."RunTimeTicks" AS track_rt,
				i."RunTimeTicks" AS item_rt,
				CASE
					WHEN e."Id" IS NOT NULL THEN 'Episode'
					WHEN mt."Id" IS NOT NULL THEN 'Audio'
					WHEN i."Type" = 'Movie' THEN 'Movie'
					WHEN p."ItemType" IN ('Movie', 'Episode', 'Audio') THEN p."ItemType"
					ELSE 'Other'
				END AS media_type
			FROM jf_playback_reporting_plugin_data p
			LEFT JOIN jf_library_episodes e ON e."Id" = p."ItemId"
			LEFT JOIN jf_library_items s ON s."Id" = e."SeriesId"
			LEFT JOIN jf_music_tracks mt ON mt."Id" = p."ItemId"
			LEFT JOIN jf_library_items i ON i."Id" = p."ItemId"
			WHERE p."DateCreated" ~ '^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}'
		)
		INSERT INTO jf_playback_activity (
			"Id", "IsPaused", "UserId", "UserName", "Client", "DeviceName",
			"NowPlayingItemId", "NowPlayingItemName", "SeasonId", "SeriesName", "EpisodeId",
			"PlaybackDuration", "ActivityDateInserted", "PlayMethod",
			"ItemId", "MediaType", "RunTimeTicks", "StartedAt", "EndedAt", "IsView",
			source, imported
		)
		SELECT
			src."rowid", false, src."UserId", COALESCE(u."Name", src."UserId"),
			src."ClientName", src."DeviceName",
			COALESCE(src.series_id, src.album_id, src."ItemId"),
			src."ItemName",
			src.season_id,
			COALESCE(src.series_name, src.album_name),
			CASE WHEN src.ep_id IS NOT NULL OR src.track_id IS NOT NULL THEN src."ItemId" END,
			src.dur,
			src."DateCreated",
			src."PlaybackMethod",
			src."ItemId",
			src.media_type,
			COALESCE(src.ep_rt, src.track_rt, src.item_rt),
			src.started_at,
			src.started_at + make_interval(secs => src.dur),
			src.dur >= 30,
			'plugin', true
		FROM src
		LEFT JOIN jf_users u ON u."Id" = src."UserId"
		WHERE src.media_type IN ('Movie', 'Episode', 'Audio')
		  AND src.started_at < now() - interval '1 day'
		  AND NOT EXISTS (
			SELECT 1 FROM jf_activity_watchdog w
			WHERE w."UserId" = src."UserId" AND w."ItemId" = src."ItemId"
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM jf_playback_activity a
			WHERE COALESCE(a.imported, false) = false
			  AND a."UserId" = src."UserId"
			  AND a."ItemId" = src."ItemId"
			  AND (
				(a."StartedAt" <= src.started_at + make_interval(secs => src.dur) + interval '2 minutes'
				 AND COALESCE(a."EndedAt", a."StartedAt") >= src.started_at - interval '2 minutes')
				OR
				(a."StartedAt" BETWEEN src.started_at - interval '14 hours' AND src.started_at + interval '14 hours'
				 AND abs(COALESCE(a."PlaybackDuration", 0) - src.dur) <= GREATEST(60, 0.15 * GREATEST(COALESCE(a."PlaybackDuration", 0), src.dur)))
			  )
		  )
		  -- One play = one row: also skipped when an imported row of another
		  -- origin (plugin PBR import, Jellystics history, TSV import) records it.
		  AND NOT EXISTS (
			SELECT 1 FROM jf_playback_activity a
			WHERE a.imported AND a.source IN ('import-pbr', 'import-jellystics', 'import')
			  AND a."UserId" = src."UserId" AND a."ItemId" = src."ItemId"
			  AND a."StartedAt" <= src.started_at + make_interval(secs => src.dur) + interval '2 minutes'
			  AND COALESCE(a."EndedAt", a."StartedAt") >= src.started_at - interval '2 minutes'
		  )
		ON CONFLICT ("Id") DO NOTHING
	`, sql.Named("tz", displayTZ())).Error
}

// BackfillTracking fills the accurate-tracking columns (migration 008) of
// rows that lack them: rows recorded before the upgrade, imported plugin
// rows, and rows coming from a restored backup. Safe to run repeatedly; it
// only touches rows whose ItemId or StartedAt is NULL.
//
// Dates with an offset are parsed as-is; dates without one are server-local
// time in TZ (imported plugin rows are also shifted by the plugin's 20 s
// delay). Items no longer in the catalog keep a best-effort media type from
// the legacy columns (episode/track rows had EpisodeId set).
func (r *playbackRepo) BackfillTracking(ctx context.Context) error {
	err := r.db.WithContext(ctx).Exec(`
		UPDATE jf_playback_activity a SET
			"ItemId" = COALESCE(a."ItemId", a."EpisodeId", a."NowPlayingItemId"),
			"MediaType" = COALESCE(a."MediaType", CASE
				WHEN x.ep_id IS NOT NULL THEN 'Episode'
				WHEN x.track_id IS NOT NULL THEN 'Audio'
				WHEN x.item_type = 'Movie' THEN 'Movie'
				WHEN x.item_type IS NOT NULL THEN 'Other'
				WHEN a."EpisodeId" IS NOT NULL AND a."SeasonId" IS NOT NULL THEN 'Episode'
				WHEN a."EpisodeId" IS NOT NULL THEN 'Audio'
				ELSE 'Movie'
			END),
			"RunTimeTicks" = COALESCE(a."RunTimeTicks", x.rt),
			"StartedAt" = COALESCE(a."StartedAt", CASE
				WHEN a."ActivityDateInserted" ~ '\d{2}:\d{2}(:\d{2}(\.\d+){0,1}){0,1}\s*(Z|[+-]\d{2}(:{0,1}\d{2}){0,1})$'
					THEN a."ActivityDateInserted"::timestamptz
				WHEN a."ActivityDateInserted" ~ '^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}'
					THEN (a."ActivityDateInserted"::timestamp AT TIME ZONE CAST(@tz AS text))
					     - CASE WHEN COALESCE(a.imported, false) THEN interval '20 seconds' ELSE interval '0' END
			END),
			"IsView" = CASE WHEN a."ItemId" IS NULL THEN COALESCE(a."PlaybackDuration", 0) >= 30 ELSE a."IsView" END
		FROM (
			SELECT b."Id",
				e."Id" AS ep_id, t."Id" AS track_id, i."Type" AS item_type,
				COALESCE(e."RunTimeTicks", t."RunTimeTicks", i."RunTimeTicks") AS rt
			FROM jf_playback_activity b
			LEFT JOIN jf_library_episodes e ON e."Id" = COALESCE(b."ItemId", b."EpisodeId", b."NowPlayingItemId")
			LEFT JOIN jf_music_tracks t ON t."Id" = COALESCE(b."ItemId", b."EpisodeId", b."NowPlayingItemId")
			LEFT JOIN jf_library_items i ON i."Id" = COALESCE(b."ItemId", b."EpisodeId", b."NowPlayingItemId")
			WHERE b."ItemId" IS NULL OR b."StartedAt" IS NULL
		) x
		WHERE x."Id" = a."Id"
	`, sql.Named("tz", displayTZ())).Error
	if err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Exec(`
		UPDATE jf_playback_activity
		SET "EndedAt" = "StartedAt" + make_interval(secs => COALESCE("PlaybackDuration", 0))
		WHERE "EndedAt" IS NULL AND "StartedAt" IS NOT NULL
	`).Error; err != nil {
		return err
	}
	return r.sanitizeImported(ctx)
}

// SanitizeImported repairs impossible durations of imported rows (see sanitizeImported).
func (r *playbackRepo) SanitizeImported(ctx context.Context) error { return r.sanitizeImported(ctx) }

// Longest believable imported play when the item's runtime is unknown.
const maxImportedPlaySeconds = 12 * 3600

// sanitizeImported repairs impossible durations coming from imported
// Playback Reporting data (the plugin can store a negative / int32-overflowed
// PlayDuration, or keep a session open for days): negative → 0, more than
// 1.5× the item's runtime → the runtime, more than 12 h with no known runtime
// → 12 h. EndedAt and IsView follow the repaired duration and the row is
// flagged DurationRepaired (data quality). Tracked rows are never touched
// (the tracker already clamps to wall time).
func (r *playbackRepo) sanitizeImported(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec(`
		WITH fixed AS (
			SELECT "Id",
				CASE
					WHEN "PlaybackDuration" < 0 THEN 0
					WHEN COALESCE("RunTimeTicks", 0) > 0 AND "PlaybackDuration" > ("RunTimeTicks" / 10000000.0) * 1.5
						THEN CEIL("RunTimeTicks" / 10000000.0)::bigint
					WHEN COALESCE("RunTimeTicks", 0) <= 0 AND "PlaybackDuration" > @max THEN @max
				END AS dur
			FROM jf_playback_activity
			WHERE imported AND "PlaybackDuration" IS NOT NULL
		)
		UPDATE jf_playback_activity a SET
			"PlaybackDuration" = f.dur,
			"EndedAt" = CASE WHEN a."StartedAt" IS NULL THEN a."EndedAt" ELSE a."StartedAt" + make_interval(secs => f.dur) END,
			"IsView" = f.dur >= 30,
			"DurationRepaired" = true
		FROM fixed f
		WHERE f."Id" = a."Id" AND f.dur IS NOT NULL
	`, sql.Named("max", maxImportedPlaySeconds)).Error
}

// ---- logging ----

type logRepo struct{ db *gorm.DB }

func (r *logRepo) Insert(ctx context.Context, entry *models.JFLogging) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *logRepo) Upsert(ctx context.Context, entry *models.JFLogging) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns([]string{"Name", "Type", "ExecutionType", "Duration", "TimeRun", "Log", "Result"}),
		}).
		Create(entry).Error
}

func (r *logRepo) List(ctx context.Context, limit int) ([]models.JFLogging, error) {
	var out []models.JFLogging
	q := r.db.WithContext(ctx).Order(`"TimeRun" DESC`)
	if limit > 0 {
		q = q.Limit(limit)
	}
	return out, q.Find(&out).Error
}

// ---- webhooks ----

type webhookRepo struct{ db *gorm.DB }

func (r *webhookRepo) List(ctx context.Context) ([]models.Webhook, error) {
	var out []models.Webhook
	return out, r.db.WithContext(ctx).Find(&out).Error
}

func (r *webhookRepo) GetByID(ctx context.Context, id int) (*models.Webhook, error) {
	var wh models.Webhook
	err := r.db.WithContext(ctx).First(&wh, id).Error
	return &wh, err
}

func (r *webhookRepo) Create(ctx context.Context, wh *models.Webhook) error {
	return r.db.WithContext(ctx).Create(wh).Error
}

func (r *webhookRepo) Update(ctx context.Context, wh *models.Webhook) error {
	return r.db.WithContext(ctx).Save(wh).Error
}

func (r *webhookRepo) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&models.Webhook{}, id).Error
}

// ---- stats ----
//
// All aggregates follow package statsql: Movie / Episode / Audio rows only,
// plays = IsView rows, watch time = SUM(PlaybackDuration), local days in the
// display time zone.

type statsRepo struct{ db *gorm.DB }

func (r *statsRepo) RefreshViews(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec(`CALL ju_update_library_stats_data()`).Error
}

func (r *statsRepo) GetGlobalStats(ctx context.Context) (*GlobalStats, error) {
	var stats GlobalStats
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FILTER (WHERE a."IsView") FROM jf_playback_activity a WHERE `+statsql.MediaFilter("a")+`) AS total_plays,
			(SELECT COALESCE(SUM(a."PlaybackDuration"), 0) FROM jf_playback_activity a WHERE `+statsql.MediaFilter("a")+`) AS total_duration,
			(SELECT COUNT(*) FROM jf_users) AS total_users,
			(SELECT COUNT(*) FROM jf_libraries WHERE archived = false) AS total_libraries
	`).Scan(&stats).Error
	return &stats, err
}

func (r *statsRepo) GetMostViewedLibraries(ctx context.Context, limit int) ([]LibraryPlayStat, error) {
	var out []LibraryPlayStat
	err := r.db.WithContext(ctx).Raw(`
		WITH plays AS (
			SELECT `+statsql.LibraryID+` AS library_id, COUNT(*) AS total_plays
			FROM jf_playback_activity a`+statsql.CatalogJoins+`
			WHERE a."IsView" AND `+statsql.MediaFilter("a")+`
			GROUP BY 1
		)
		SELECT l."Id" AS id, l."Name" AS name, l."CollectionType" AS collection_type,
		       COALESCE(p.total_plays, 0)::bigint AS total_plays
		FROM jf_libraries l
		LEFT JOIN plays p ON p.library_id = l."Id"
		WHERE l.archived = false
		ORDER BY total_plays DESC, l."Name"
		LIMIT ?
	`, limit).Scan(&out).Error
	return out, err
}

func (r *statsRepo) GetLibraryStats(ctx context.Context, libraryId string) (*LibraryStats, error) {
	var stats LibraryStats
	err := r.db.WithContext(ctx).Raw(`
		WITH plays AS (
			SELECT COUNT(*) FILTER (WHERE a."IsView")::int AS total_plays,
			       MAX(`+statsql.StartedAt("a")+`) AS last_activity
			FROM jf_playback_activity a`+statsql.CatalogJoins+`
			WHERE `+statsql.MediaFilter("a")+` AND `+statsql.LibraryID+` = ?
		)
		SELECT
			l."Id" AS id,
			((SELECT COUNT(*) FROM jf_library_items WHERE "ParentId" = l."Id" AND archived = false AND "Type" = 'Movie')
			 + (SELECT COUNT(*) FROM jf_library_episodes e JOIN jf_library_items s ON s."Id" = e."SeriesId" AND s.archived = false
			    WHERE s."ParentId" = l."Id" AND e.archived = false)
			 + (SELECT COUNT(*) FROM jf_music_tracks WHERE "LibraryId" = l."Id" AND archived = false))::int AS total_items,
			(SELECT COUNT(*) FROM jf_library_episodes e JOIN jf_library_items s ON s."Id" = e."SeriesId" AND s.archived = false
			 WHERE s."ParentId" = l."Id" AND e.archived = false)::int AS total_episodes,
			(SELECT total_plays FROM plays) AS total_plays,
			(SELECT `+statsql.ISO("last_activity")+` FROM plays) AS last_activity
		FROM jf_libraries l
		WHERE l."Id" = ?
	`, libraryId, libraryId).Scan(&stats).Error
	return &stats, err
}

func (r *statsRepo) GetActivityOverTime(ctx context.Context, days int) ([]DailyActivity, error) {
	var out []DailyActivity
	loc := statsql.Location()
	since := statsql.StartOfDay(time.Now(), loc).AddDate(0, 0, -days)
	err := r.db.WithContext(ctx).Raw(`
		SELECT TO_CHAR(`+statsql.LocalTS("a")+`::date, 'YYYY-MM-DD') AS date, COUNT(*) AS count
		FROM jf_playback_activity a
		WHERE a."IsView" AND `+statsql.MediaFilter("a")+` AND `+statsql.StartedAt("a")+` >= ?
		GROUP BY 1
		ORDER BY 1 ASC
	`, since).Scan(&out).Error
	return out, err
}

// GetTopUsers returns one row per user id, labelled with the user's current
// name (a rename must not split a user into two rows / label sets).
func (r *statsRepo) GetTopUsers(ctx context.Context, limit int) ([]UserStat, error) {
	var out []UserStat
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			a."UserId" AS user_id,
			COALESCE(MAX(u."Name"), MAX(a."UserName"), a."UserId") AS user_name,
			COUNT(*) FILTER (WHERE a."IsView") AS total_plays,
			COALESCE(SUM(a."PlaybackDuration"), 0)::float8 / 3600.0 AS total_hours
		FROM jf_playback_activity a
		LEFT JOIN jf_users u ON u."Id" = a."UserId"
		WHERE a."UserId" IS NOT NULL AND `+statsql.MediaFilter("a")+`
		GROUP BY a."UserId"
		ORDER BY total_plays DESC, total_hours DESC
		LIMIT ?
	`, limit).Scan(&out).Error
	return out, err
}

// GetMostPlayedItems lists the library's movies / series / albums by views
// (series and albums aggregate their episodes / tracks).
func (r *statsRepo) GetMostPlayedItems(ctx context.Context, libraryId string, limit int) ([]ItemPlayStat, error) {
	var out []ItemPlayStat
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			i."Id" AS id, i."Name" AS name, i."Type" AS type,
			COUNT(*) FILTER (WHERE a."IsView")::bigint AS times_played,
			COALESCE(SUM(a."PlaybackDuration"), 0) AS total_duration
		FROM jf_playback_activity a`+statsql.CatalogJoins+`
		JOIN jf_library_items i ON i."Id" = COALESCE(cm."Id", cs."Id", cal."Id")
		WHERE `+statsql.MediaFilter("a")+` AND i."ParentId" = ? AND i.archived = false
		GROUP BY i."Id", i."Name", i."Type"
		HAVING COUNT(*) FILTER (WHERE a."IsView") > 0
		ORDER BY times_played DESC
		LIMIT ?
	`, libraryId, limit).Scan(&out).Error
	return out, err
}

// trackViews counts views per track id.
const trackViews = `(
	SELECT "ItemId" AS track_id, COUNT(*) AS plays
	FROM jf_playback_activity
	WHERE "MediaType" = 'Audio' AND "IsView"
	GROUP BY "ItemId"
)`

func (r *statsRepo) GetMostPlayedArtists(ctx context.Context, libraryId string, limit int) ([]ArtistPlayStat, error) {
	var out []ArtistPlayStat
	err := r.db.WithContext(ctx).Raw(`
		SELECT ar."Id" AS artist_id, ar."Name" AS artist_name, SUM(v.plays)::bigint AS times_played
		FROM jf_music_artists ar
		JOIN jf_music_tracks t ON t."ArtistId" = ar."Id"
		JOIN `+trackViews+` v ON v.track_id = t."Id"
		WHERE ar."LibraryId" = ? AND ar.archived = false
		GROUP BY ar."Id", ar."Name"
		ORDER BY times_played DESC
		LIMIT ?
	`, libraryId, limit).Scan(&out).Error
	return out, err
}

func (r *statsRepo) GetMostPlayedAlbums(ctx context.Context, libraryId, artistId string, limit int) ([]AlbumPlayStat, error) {
	var out []AlbumPlayStat
	sql := `
		SELECT i."Id" AS album_id, i."Name" AS album_name, SUM(v.plays)::bigint AS times_played
		FROM jf_library_items i
		JOIN jf_music_tracks t ON t."AlbumId" = i."Id"
		JOIN ` + trackViews + ` v ON v.track_id = t."Id"
		WHERE i."Type" = 'MusicAlbum' AND i.archived = false
	`
	args := []any{}
	if libraryId != "" {
		sql += ` AND i."ParentId" = ?`
		args = append(args, libraryId)
	}
	if artistId != "" {
		sql += ` AND EXISTS (
			SELECT 1 FROM jf_music_tracks t2
			WHERE t2."AlbumId" = i."Id" AND t2."ArtistId" = ?
		)`
		args = append(args, artistId)
	}
	sql += ` GROUP BY i."Id", i."Name" ORDER BY times_played DESC LIMIT ?`
	args = append(args, limit)
	return out, r.db.WithContext(ctx).Raw(sql, args...).Scan(&out).Error
}

func (r *statsRepo) GetMostPlayedTracks(ctx context.Context, libraryId, albumId string, limit int) ([]TrackPlayStat, error) {
	sql := `
		SELECT t."Id" AS track_id, t."Name" AS track_name, t."AlbumName" AS album_name,
		       v.plays::bigint AS times_played
		FROM jf_music_tracks t
		JOIN ` + trackViews + ` v ON v.track_id = t."Id"
		WHERE t.archived = false
	`
	args := []any{}
	if libraryId != "" {
		sql += ` AND t."LibraryId" = ?`
		args = append(args, libraryId)
	}
	if albumId != "" {
		sql += ` AND t."AlbumId" = ?`
		args = append(args, albumId)
	}
	sql += ` ORDER BY times_played DESC, t."Name" LIMIT ?`
	args = append(args, limit)
	var result []TrackPlayStat
	return result, r.db.WithContext(ctx).Raw(sql, args...).Scan(&result).Error
}

func (r *statsRepo) GetUserHistory(ctx context.Context, userId string, page, pageSize int) ([]ActivityEntry, int64, error) {
	var total int64
	r.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM jf_playback_activity a WHERE a."UserId" = ? AND `+statsql.MediaFilter("a"), userId).Scan(&total)

	var rows []ActivityEntry
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			a."Id" AS id,
			a."UserId" AS user_id,
			COALESCE(u."Name", a."UserName", a."UserId") AS user_name,
			a."NowPlayingItemId" AS now_playing_item_id,
			a."NowPlayingItemName" AS now_playing_item_name,
			a."SeriesName" AS series_name,
			a."EpisodeId" AS episode_id,
			a."PlaybackDuration" AS playback_duration,
			a."PlayMethod" AS play_method,
			a."Client" AS client,
			a."DeviceName" AS device_name,
			a."ActivityDateInserted" AS activity_date_inserted
		FROM jf_playback_activity a
		LEFT JOIN jf_users u ON u."Id" = a."UserId"
		WHERE a."UserId" = ? AND `+statsql.MediaFilter("a")+`
		ORDER BY `+statsql.StartedAt("a")+` DESC
		LIMIT ? OFFSET ?
	`, userId, pageSize, (page-1)*pageSize).Scan(&rows).Error
	return rows, total, err
}
