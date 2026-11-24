-- 017: « Mes statistiques » page (Jellyfin plugin relay).
--
-- app_config.me_secret: random 32-byte hex secret shared with the Jellystics
-- plugin (POST /Jellystics/Link); the plugin sends it on every /api/me call.
-- Generated once by Jellystics, never returned by the API. Written with
-- targeted UPDATEs only (not part of the AppConfig model, so saving the
-- config never overwrites it).
ALTER TABLE app_config ADD COLUMN IF NOT EXISTS me_secret text NOT NULL DEFAULT '';

-- Link state (plugin sync): what was last pushed successfully (url, sha256
-- of the secret) and what the plugin reports (linked, reachable).
ALTER TABLE plugin_sync_state
  ADD COLUMN IF NOT EXISTS me_linked             boolean NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS me_reachable          boolean,
  ADD COLUMN IF NOT EXISTS me_linked_url         text,
  ADD COLUMN IF NOT EXISTS me_linked_secret_hash text,
  ADD COLUMN IF NOT EXISTS me_linked_at          timestamptz,
  ADD COLUMN IF NOT EXISTS me_link_error         text;
