-- 010: remember Jellyfin's LastActivityDate (server clock) at each poll.
-- After a stop, the session's LastActivityDate is the stop time; the gap with
-- the value stored at the last poll is the playback time after that poll,
-- measured entirely on the Jellyfin server clock (no clock-skew issue).
ALTER TABLE jf_activity_watchdog
  ADD COLUMN IF NOT EXISTS "LastServerActivity" timestamptz;
