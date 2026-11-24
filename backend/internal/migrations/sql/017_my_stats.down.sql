ALTER TABLE plugin_sync_state
  DROP COLUMN IF EXISTS me_link_error,
  DROP COLUMN IF EXISTS me_linked_at,
  DROP COLUMN IF EXISTS me_linked_secret_hash,
  DROP COLUMN IF EXISTS me_linked_url,
  DROP COLUMN IF EXISTS me_reachable,
  DROP COLUMN IF EXISTS me_linked;

ALTER TABLE app_config DROP COLUMN IF EXISTS me_secret;
