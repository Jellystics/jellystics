-- 015: data quality tracking and Jellystics history push to the plugin.

-- DurationRepaired: Jellystics replaced an impossible imported duration
-- (negative int32 overflow, longer than 1.5x the runtime, more than 12 h
-- without a known runtime). Set by sanitizeImported and by the Playback
-- Reporting TSV import.
ALTER TABLE jf_playback_activity
  ADD COLUMN IF NOT EXISTS "DurationRepaired" boolean NOT NULL DEFAULT false;

-- Legacy Playback Reporting imports repaired before this migration: the
-- staging table still holds the original duration (row id = rowid, or
-- 'plugin-' || rowid for older imports). TSV imports have no staging row:
-- their earlier repairs cannot be identified.
UPDATE jf_playback_activity a SET "DurationRepaired" = true
FROM jf_playback_reporting_plugin_data p
WHERE a.source = 'plugin' AND a.imported AND (a."Id" = p."rowid" OR a."Id" = 'plugin-' || p."rowid")
  AND p."PlayDuration" IS NOT NULL AND a."PlaybackDuration" IS DISTINCT FROM p."PlayDuration";

-- Push of the Jellystics history to the plugin (POST /Jellystics/Import/Plays),
-- in ("StartedAt", "Id") order, resumable from (push_cursor_at, push_cursor):
-- the last pushed play.
ALTER TABLE plugin_sync_state
  ADD COLUMN IF NOT EXISTS push_state       text NOT NULL DEFAULT 'idle', -- idle | running | done | error
  ADD COLUMN IF NOT EXISTS push_cursor      text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS push_cursor_at   timestamptz,
  ADD COLUMN IF NOT EXISTS push_total       bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS push_sent        bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS push_imported    bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS push_skipped     bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS push_rejected    bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS push_started_at  timestamptz,
  ADD COLUMN IF NOT EXISTS push_finished_at timestamptz,
  ADD COLUMN IF NOT EXISTS push_error       text;
