ALTER TABLE plugin_sync_state
  DROP COLUMN IF EXISTS push_state,
  DROP COLUMN IF EXISTS push_cursor,
  DROP COLUMN IF EXISTS push_total,
  DROP COLUMN IF EXISTS push_sent,
  DROP COLUMN IF EXISTS push_imported,
  DROP COLUMN IF EXISTS push_skipped,
  DROP COLUMN IF EXISTS push_rejected,
  DROP COLUMN IF EXISTS push_started_at,
  DROP COLUMN IF EXISTS push_finished_at,
  DROP COLUMN IF EXISTS push_error;
ALTER TABLE jf_playback_activity DROP COLUMN IF EXISTS "DurationRepaired";
