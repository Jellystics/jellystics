-- 014: identity of the Jellystics plugin database the sync cursors refer to.
-- A different id reported by /Jellystics/Info means the plugin data was reset
-- (reinstall, lost database): the cursors restart at 0, even when the new
-- plugin cursor is already past ours.
ALTER TABLE plugin_sync_state ADD COLUMN IF NOT EXISTS plugin_database_id text;
