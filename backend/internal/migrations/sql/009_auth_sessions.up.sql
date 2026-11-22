-- 009: Jellystics login sessions backed by a Jellyfin access token.
-- A session stays valid exactly as long as its Jellyfin token does (revoked on
-- logout, or when the device is removed in the Jellyfin dashboard).
CREATE TABLE IF NOT EXISTS auth_sessions (
  id                text PRIMARY KEY,
  user_id           text NOT NULL,
  user_name         text,
  jf_token          text NOT NULL,
  device_id         text NOT NULL,
  device_name       text,
  created_at        timestamptz NOT NULL DEFAULT now(),
  last_validated_at timestamptz NOT NULL DEFAULT now(),
  last_seen_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_user ON auth_sessions (user_id);
