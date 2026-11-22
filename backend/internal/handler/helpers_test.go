package handler_test

import (
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/testutil"
)

func sp(s string) *string       { return &s }
func i64(v int64) *int64        { return &v }
func ip(n int) *int             { return &n }
func tp(t time.Time) *time.Time { return &t }

const secTicks = 10_000_000 // ticks per second

// trackedRow builds a playback row with the accurate-tracking columns set.
func trackedRow(id, user, itemId, mediaType string, start time.Time, durSec, runtimeSec int64, posEndSec *int64) models.JFPlaybackActivity {
	r := models.JFPlaybackActivity{
		Id: id, UserId: sp(user), UserName: sp(user),
		NowPlayingItemId: sp(itemId), NowPlayingItemName: sp(itemId),
		ItemId: sp(itemId), MediaType: sp(mediaType),
		PlaybackDuration: i64(durSec), RunTimeTicks: i64(runtimeSec * secTicks),
		StartedAt: tp(start), EndedAt: tp(start.Add(time.Duration(durSec) * time.Second)),
		ActivityDateInserted: sp(testutil.TS(start)),
		IsView:               durSec >= 30,
		Source:               "watchdog",
	}
	if posEndSec != nil {
		r.PositionTicksEnd = i64(*posEndSec * secTicks)
	}
	return r
}
