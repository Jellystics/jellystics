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

-- Existing rows are backfilled at startup by the Go code (playbackRepo.
-- BackfillTracking): it knows the configured TZ, needed for dates stored
-- without an offset, and it also runs after a backup restore / import.

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
