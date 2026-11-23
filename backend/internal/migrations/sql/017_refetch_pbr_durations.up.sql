-- The Playback Reporting import read every duration as 0 (the plugin returns
-- its query cells as text). When no staged row has a duration, drop the
-- staged rows: the next import re-reads the whole history and fills the
-- durations of the plays already merged.
DELETE FROM jf_playback_reporting_plugin_data
WHERE NOT EXISTS (
	SELECT 1 FROM jf_playback_reporting_plugin_data WHERE "PlayDuration" > 0
);
