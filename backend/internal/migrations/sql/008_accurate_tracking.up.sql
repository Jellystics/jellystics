-- 008: accurate playback tracking.
--
-- jf_playback_activity: one row per playback attempt ("click"). A row is a
-- "view" when the real watched time (PlaybackDuration, seconds, pauses and
-- seeks excluded) is >= 30 s.
ALTER TABLE jf_playback_activity
  ADD COLUMN IF NOT EXISTS "ItemId"             text,        -- the item actually played (movie / episode / track id)
  ADD COLUMN IF NOT EXISTS "MediaType"          text,        -- 'Movie' | 'Episode' | 'Audio' (anything else = 'Other', ignored by stats)
  ADD COLUMN IF NOT EXISTS "RunTimeTicks"       bigint,      -- runtime of ItemId at play time
  ADD COLUMN IF NOT EXISTS "PositionTicksStart" bigint,      -- first observed position (resume point)
  ADD COLUMN IF NOT EXISTS "PositionTicksEnd"   bigint,      -- last known position at stop
  ADD COLUMN IF NOT EXISTS "PausedSeconds"      bigint,      -- wall-clock time spent paused
  ADD COLUMN IF NOT EXISTS "StartedAt"          timestamptz, -- playback start (authoritative, replaces text ActivityDateInserted)
  ADD COLUMN IF NOT EXISTS "EndedAt"            timestamptz, -- playback end
  ADD COLUMN IF NOT EXISTS "IsView"             boolean NOT NULL DEFAULT false;

-- Backfill ItemId: episode/track id when present, else the movie id.
UPDATE jf_playback_activity
SET "ItemId" = COALESCE("EpisodeId", "NowPlayingItemId")
WHERE "ItemId" IS NULL;

-- Backfill MediaType from the catalog.
UPDATE jf_playback_activity a
SET "MediaType" = CASE
    WHEN EXISTS (SELECT 1 FROM jf_library_episodes e WHERE e."Id" = a."ItemId") THEN 'Episode'
    WHEN EXISTS (SELECT 1 FROM jf_music_tracks t WHERE t."Id" = a."ItemId") THEN 'Audio'
    WHEN EXISTS (SELECT 1 FROM jf_library_items i WHERE i."Id" = a."ItemId" AND i."Type" = 'Movie') THEN 'Movie'
    ELSE 'Other'
  END
WHERE "MediaType" IS NULL;

-- Backfill RunTimeTicks from the catalog.
UPDATE jf_playback_activity a
SET "RunTimeTicks" = COALESCE(
    (SELECT e."RunTimeTicks" FROM jf_library_episodes e WHERE e."Id" = a."ItemId"),
    (SELECT t."RunTimeTicks" FROM jf_music_tracks t WHERE t."Id" = a."ItemId"),
    (SELECT i."RunTimeTicks" FROM jf_library_items i WHERE i."Id" = a."ItemId"))
WHERE "RunTimeTicks" IS NULL;

-- Backfill StartedAt for rows written by Jellystics (text carries an offset).
-- Plugin-imported rows have no offset (server local time); they are backfilled
-- at startup by the Go code, which knows the configured TZ.
UPDATE jf_playback_activity
SET "StartedAt" = "ActivityDateInserted"::timestamptz
WHERE "StartedAt" IS NULL
  AND COALESCE(imported, false) = false
  AND "ActivityDateInserted" ~ '[+-][0-9]{2}(:?[0-9]{2})?$|Z$';

UPDATE jf_playback_activity
SET "EndedAt" = "StartedAt" + make_interval(secs => COALESCE("PlaybackDuration", 0))
WHERE "EndedAt" IS NULL AND "StartedAt" IS NOT NULL;

UPDATE jf_playback_activity
SET "IsView" = COALESCE("PlaybackDuration", 0) >= 30;

CREATE INDEX IF NOT EXISTS idx_playback_started_at ON jf_playback_activity ("StartedAt");
CREATE INDEX IF NOT EXISTS idx_playback_item_id    ON jf_playback_activity ("ItemId");
CREATE INDEX IF NOT EXISTS idx_playback_user_start ON jf_playback_activity ("UserId", "StartedAt");
CREATE INDEX IF NOT EXISTS idx_playback_media_type ON jf_playback_activity ("MediaType");

-- jf_activity_watchdog: live accumulation state (millisecond precision).
ALTER TABLE jf_activity_watchdog
  ADD COLUMN IF NOT EXISTS "ItemId"             text,
  ADD COLUMN IF NOT EXISTS "MediaType"          text,
  ADD COLUMN IF NOT EXISTS "RunTimeTicks"       bigint,
  ADD COLUMN IF NOT EXISTS "PositionTicksStart" bigint,
  ADD COLUMN IF NOT EXISTS "WatchedMs"          bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "PausedMs"           bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "StartedAt"          timestamptz,
  ADD COLUMN IF NOT EXISTS "LastCheckIn"        timestamptz, -- Jellyfin LastPlaybackCheckIn (real client reports only)
  ADD COLUMN IF NOT EXISTS "PeriodicReporter"   boolean NOT NULL DEFAULT false;

UPDATE jf_activity_watchdog SET "WatchedMs" = "WatchedSeconds" * 1000 WHERE "WatchedMs" = 0;
