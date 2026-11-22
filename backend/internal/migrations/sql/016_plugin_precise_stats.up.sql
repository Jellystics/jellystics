-- 016: precise statistics from the Jellystics plugin.
--
-- jf_library_events."Kind": "added" / "removed" = live events (At = when the
-- plugin saw them); "present" = first-run catalog snapshot (At = snapshot
-- time), never an addition. "DateCreated" = Jellyfin item DateCreated
-- (added / present rows).
ALTER TABLE jf_library_events ADD COLUMN IF NOT EXISTS "DateCreated" timestamptz;
CREATE INDEX IF NOT EXISTS idx_library_events_library_at ON jf_library_events ("LibraryId", "At");

-- First plugin play (pluginSince) without scanning the tracker history.
CREATE INDEX IF NOT EXISTS idx_playback_plugin_started_at
  ON jf_playback_activity ("StartedAt") WHERE "PluginOpen" IS NOT NULL;
