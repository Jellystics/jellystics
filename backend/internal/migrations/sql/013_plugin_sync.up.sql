-- 013: Jellystics plugin sync.
--
-- The Jellystics Jellyfin plugin records every playback server-side and
-- exposes it (plus library / user-data / login events) through a cursor API
-- (GET /Jellystics/<Table>?since=<version>). Jellystics stays autonomous
-- without it: these tables simply stay empty.

-- Extra playback fields provided by the plugin. DeviceId already exists
-- (001) but is re-declared for safety; it is now also the dedupe key between
-- tracker and plugin rows.
ALTER TABLE jf_playback_activity
  ADD COLUMN IF NOT EXISTS "PlaySessionId"      text,
  ADD COLUMN IF NOT EXISTS "DeviceId"           text,
  ADD COLUMN IF NOT EXISTS "SeekCount"          int,
  ADD COLUMN IF NOT EXISTS "SkippedSeconds"     int,
  ADD COLUMN IF NOT EXISTS "PlayedToCompletion" boolean;

CREATE INDEX IF NOT EXISTS idx_playback_device_user_item_start
  ON jf_playback_activity ("DeviceId", "UserId", "ItemId", "StartedAt");

-- Library additions / removals (plugin LibraryEvents).
CREATE TABLE IF NOT EXISTS jf_library_events (
  "Id"           text PRIMARY KEY,
  "Kind"         text NOT NULL,          -- added | removed
  "At"           timestamptz NOT NULL,
  "ItemId"       text,
  "ItemType"     text,
  "ItemName"     text,
  "SeriesId"     text,
  "AlbumId"      text,
  "LibraryId"    text,
  "RunTimeTicks" bigint,
  "SizeBytes"    bigint,
  "Version"      bigint NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_library_events_at   ON jf_library_events ("At");
CREATE INDEX IF NOT EXISTS idx_library_events_item ON jf_library_events ("ItemId");

-- Played / favorite changes (plugin UserDataEvents).
CREATE TABLE IF NOT EXISTS jf_user_data_events (
  "Id"      text PRIMARY KEY,
  "At"      timestamptz NOT NULL,
  "UserId"  text,
  "ItemId"  text,
  "Kind"    text NOT NULL,               -- played | unplayed | favorite | unfavorite
  "Reason"  text,                        -- Jellyfin UserDataSaveReason
  "Version" bigint NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_user_data_events_user_at ON jf_user_data_events ("UserId", "At");
CREATE INDEX IF NOT EXISTS idx_user_data_events_item    ON jf_user_data_events ("ItemId");

-- Login attempts (plugin Logins).
CREATE TABLE IF NOT EXISTS jf_user_logins (
  "Id"                 text PRIMARY KEY,
  "At"                 timestamptz NOT NULL,
  "UserId"             text,
  "UserName"           text,
  "DeviceId"           text,
  "DeviceName"         text,
  "Client"             text,
  "ApplicationVersion" text,
  "RemoteEndPoint"     text,
  "Success"            boolean NOT NULL DEFAULT false,
  "Version"            bigint NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_user_logins_user_at ON jf_user_logins ("UserId", "At");
CREATE INDEX IF NOT EXISTS idx_user_logins_at      ON jf_user_logins ("At");

-- Sync state (single row, id = 1).
CREATE TABLE IF NOT EXISTS plugin_sync_state (
  id                      int PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  detected                boolean NOT NULL DEFAULT false,
  plugin_version          text,
  schema_version          int,
  cursor_plays            bigint NOT NULL DEFAULT 0,
  cursor_library_events   bigint NOT NULL DEFAULT 0,
  cursor_user_data_events bigint NOT NULL DEFAULT 0,
  cursor_logins           bigint NOT NULL DEFAULT 0,
  last_checked_at         timestamptz,
  last_success_at         timestamptz,
  last_error              text
);
INSERT INTO plugin_sync_state (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
