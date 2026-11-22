-- 012: partial index for the first captured playback row (collectedSince of
-- the playback / languages analyses). The predicate must stay identical to
-- capturedCond in internal/handler/v2_playback.go.
CREATE INDEX IF NOT EXISTS idx_playback_captured_started_at ON jf_playback_activity ("StartedAt")
  WHERE ("AudioStreamIndex" IS NOT NULL OR "AudioLanguage" IS NOT NULL OR "Bitrate" IS NOT NULL OR "IsRemote" IS NOT NULL
         OR "VideoRange" IS NOT NULL OR "TranscodeReasons" IS NOT NULL OR "PauseCount" > 0);
