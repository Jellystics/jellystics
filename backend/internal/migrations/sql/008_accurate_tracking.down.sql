DROP INDEX IF EXISTS idx_playback_started_at;
DROP INDEX IF EXISTS idx_playback_item_id;
DROP INDEX IF EXISTS idx_playback_user_start;
DROP INDEX IF EXISTS idx_playback_media_type;

ALTER TABLE jf_playback_activity
  DROP COLUMN IF EXISTS "ItemId",
  DROP COLUMN IF EXISTS "MediaType",
  DROP COLUMN IF EXISTS "RunTimeTicks",
  DROP COLUMN IF EXISTS "PositionTicksStart",
  DROP COLUMN IF EXISTS "PositionTicksEnd",
  DROP COLUMN IF EXISTS "PausedSeconds",
  DROP COLUMN IF EXISTS "StartedAt",
  DROP COLUMN IF EXISTS "EndedAt",
  DROP COLUMN IF EXISTS "IsView";

ALTER TABLE jf_activity_watchdog
  DROP COLUMN IF EXISTS "ItemId",
  DROP COLUMN IF EXISTS "MediaType",
  DROP COLUMN IF EXISTS "RunTimeTicks",
  DROP COLUMN IF EXISTS "PositionTicksStart",
  DROP COLUMN IF EXISTS "WatchedMs",
  DROP COLUMN IF EXISTS "PausedMs",
  DROP COLUMN IF EXISTS "StartedAt",
  DROP COLUMN IF EXISTS "LastCheckIn",
  DROP COLUMN IF EXISTS "PeriodicReporter";
