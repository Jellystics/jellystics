DROP TABLE IF EXISTS plugin_sync_state;
DROP TABLE IF EXISTS jf_user_logins;
DROP TABLE IF EXISTS jf_user_data_events;
DROP TABLE IF EXISTS jf_library_events;
DROP INDEX IF EXISTS idx_playback_user_item_start;
-- DeviceId predates 013 (001) and is kept.
ALTER TABLE jf_playback_activity
  DROP COLUMN IF EXISTS "PlaySessionId",
  DROP COLUMN IF EXISTS "SeekCount",
  DROP COLUMN IF EXISTS "SkippedSeconds",
  DROP COLUMN IF EXISTS "PlayedToCompletion",
  DROP COLUMN IF EXISTS "PluginOpen";
