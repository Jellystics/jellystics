DROP TABLE IF EXISTS ip_geo_cache;
ALTER TABLE jf_playback_activity
  DROP COLUMN IF EXISTS "AudioStreamIndex", DROP COLUMN IF EXISTS "AudioLanguage",
  DROP COLUMN IF EXISTS "SubtitleStreamIndex", DROP COLUMN IF EXISTS "SubtitleLanguage",
  DROP COLUMN IF EXISTS "TranscodeReasons", DROP COLUMN IF EXISTS "Bitrate",
  DROP COLUMN IF EXISTS "HwAccel", DROP COLUMN IF EXISTS "VideoRange",
  DROP COLUMN IF EXISTS "PauseCount", DROP COLUMN IF EXISTS "IsRemote";
ALTER TABLE jf_activity_watchdog
  DROP COLUMN IF EXISTS "AudioStreamIndex", DROP COLUMN IF EXISTS "AudioLanguage",
  DROP COLUMN IF EXISTS "SubtitleStreamIndex", DROP COLUMN IF EXISTS "SubtitleLanguage",
  DROP COLUMN IF EXISTS "TranscodeReasons", DROP COLUMN IF EXISTS "Bitrate",
  DROP COLUMN IF EXISTS "HwAccel", DROP COLUMN IF EXISTS "VideoRange",
  DROP COLUMN IF EXISTS "PauseCount", DROP COLUMN IF EXISTS "IsRemote";
