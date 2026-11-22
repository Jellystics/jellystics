-- 011: capture more playback fields (audio/subtitle choice, transcode reasons,
-- bitrate, hardware acceleration, video range, pause count, local/remote) and
-- an IP geolocation cache.
-- No backfill: rows recorded before this migration keep NULL (PauseCount 0).
ALTER TABLE jf_activity_watchdog
  ADD COLUMN IF NOT EXISTS "AudioStreamIndex" int,
  ADD COLUMN IF NOT EXISTS "AudioLanguage" text,
  ADD COLUMN IF NOT EXISTS "SubtitleStreamIndex" int,
  ADD COLUMN IF NOT EXISTS "SubtitleLanguage" text,
  ADD COLUMN IF NOT EXISTS "TranscodeReasons" jsonb,
  ADD COLUMN IF NOT EXISTS "Bitrate" bigint,
  ADD COLUMN IF NOT EXISTS "HwAccel" text,
  ADD COLUMN IF NOT EXISTS "VideoRange" text,
  ADD COLUMN IF NOT EXISTS "PauseCount" int NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "IsRemote" boolean;

ALTER TABLE jf_playback_activity
  ADD COLUMN IF NOT EXISTS "AudioStreamIndex" int,
  ADD COLUMN IF NOT EXISTS "AudioLanguage" text,
  ADD COLUMN IF NOT EXISTS "SubtitleStreamIndex" int,
  ADD COLUMN IF NOT EXISTS "SubtitleLanguage" text,
  ADD COLUMN IF NOT EXISTS "TranscodeReasons" jsonb,
  ADD COLUMN IF NOT EXISTS "Bitrate" bigint,
  ADD COLUMN IF NOT EXISTS "HwAccel" text,
  ADD COLUMN IF NOT EXISTS "VideoRange" text,
  ADD COLUMN IF NOT EXISTS "PauseCount" int NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "IsRemote" boolean;

CREATE TABLE IF NOT EXISTS ip_geo_cache (
  ip         text PRIMARY KEY,
  country    text,
  city       text,
  updated_at timestamptz NOT NULL DEFAULT now()
);
