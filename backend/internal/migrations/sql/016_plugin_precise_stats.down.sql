DROP INDEX IF EXISTS idx_playback_plugin_started_at;
DROP INDEX IF EXISTS idx_library_events_library_at;
ALTER TABLE jf_library_events DROP COLUMN IF EXISTS "DateCreated";
