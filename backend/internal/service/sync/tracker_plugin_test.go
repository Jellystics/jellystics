package sync

import (
	"context"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service/pluginsync"
)

// Tracker ↔ Jellystics plugin switch-over (spec §7.2).

func (d *driver) setPlugin(healthy bool) {
	d.t.Helper()
	age := "0 seconds"
	if !healthy {
		age = "3 minutes" // older than models.PluginHealthyWindow
	}
	if err := d.db.Exec(`UPDATE plugin_sync_state SET detected = true, last_success_at = now() - CAST(? AS interval)`, age).Error; err != nil {
		d.t.Fatal(err)
	}
}

// pluginDelivers stores plugin plays the way pluginsync.Service does (upsert
// + tracker dedupe).
func (d *driver) pluginDelivers(ps ...jellyfin.JellysticsPlay) {
	d.t.Helper()
	repos := repository.New(d.db)
	var acts []models.JFPlaybackActivity
	var ids []string
	for _, p := range ps {
		a, ok := pluginsync.MapPlay(p)
		if !ok {
			d.t.Fatalf("play %s not mapped", p.Id)
		}
		acts = append(acts, a)
		ids = append(ids, a.Id)
	}
	ctx := context.Background()
	if _, err := repos.PluginSync.UpsertPlays(ctx, acts); err != nil {
		d.t.Fatal(err)
	}
	if _, err := repos.PluginSync.DedupeTrackerRows(ctx, ids); err != nil {
		d.t.Fatal(err)
	}
}

// pluginPlay is the plugin's record of a movie play on dev-1 by user-1.
func pluginPlay(id, item string, start time.Time, watched time.Duration, open bool) jellyfin.JellysticsPlay {
	end := jellyfin.FlexTime{Time: start.Add(watched)}
	p := jellyfin.JellysticsPlay{
		Id: id, UserId: "user-1", ItemId: item, ItemType: sp("Movie"), DeviceId: sp("dev-1"),
		StartedAt: jellyfin.FlexTime{Time: start}, LastEventAt: end, WatchedMs: watched.Milliseconds(),
		State: "closed", Source: "plugin",
	}
	if open {
		p.State = "open"
	} else {
		p.EndedAt = &end
	}
	return p
}

func onDevice(s jellyfin.SessionInfo) jellyfin.SessionInfo {
	s.DeviceId = "dev-1"
	s.DeviceName = "TV"
	return s
}

// playFor drives a 60 s play of item on session sess (13 ticks + stop).
func (d *driver) playFor(sess, item string) {
	d.tick(0, onDevice(movieAt(sess, item, 0, false)))
	for i := 1; i <= 12; i++ {
		d.tick(5, onDevice(movieAt(sess, item, int64(i)*5000, false)))
	}
	d.tick(5) // stopped
}

func (d *driver) rowsFor(item string) []models.JFPlaybackActivity {
	var out []models.JFPlaybackActivity
	for _, p := range d.plays() {
		if p.ItemId != nil && *p.ItemId == item {
			out = append(out, p)
		}
	}
	return out
}

func TestTracker_WritesWithoutPluginAndRecordsDeviceId(t *testing.T) {
	d := newDriver(t)
	d.playFor("s1", "m1")
	p := d.plays()
	if len(p) != 1 || p[0].DeviceId == nil || *p[0].DeviceId != "dev-1" || p[0].Source != "watchdog" {
		t.Fatalf("plays = %+v", p)
	}
}

func TestTracker_HealthyPluginPausesWrites(t *testing.T) {
	d := newDriver(t)
	d.setPlugin(true)
	d.tick(0, onDevice(movieAt("s1", "m1", 0, false)))
	d.tick(5, onDevice(movieAt("s1", "m1", 5000, false)))
	if wd := d.watchdog("s1"); wd.WatchedMs < 5000 {
		t.Fatalf("tracker must keep observing: watched %d ms", wd.WatchedMs)
	}
	d.tick(5) // stopped
	if n := len(d.plays()); n != 0 {
		t.Fatalf("plays = %d, want 0 while the plugin is healthy", n)
	}
	var n int64
	d.db.Model(&models.JFActivityWatchdog{}).Count(&n)
	if n != 0 {
		t.Fatal("watchdog row must still be cleared")
	}
}

func TestTracker_UnhealthyPluginWritesAsToday(t *testing.T) {
	d := newDriver(t)
	d.setPlugin(false)
	d.playFor("s1", "m1")
	if p := d.plays(); len(p) != 1 || *p[0].PlaybackDuration < 59 {
		t.Fatalf("plays = %+v", p)
	}
}

// Tracker → plugin: the tracker recorded plays while the plugin was down;
// when it comes back, its plays replace the tracker rows; a play in progress
// during the switch ends up recorded once (by the plugin).
func TestSwitchOver_TrackerToPlugin(t *testing.T) {
	d := newDriver(t)
	d.setPlugin(false)

	d.playFor("s1", "m1") // recorded by the tracker
	tr := d.rowsFor("m1")
	if len(tr) != 1 {
		t.Fatalf("tracker rows = %d", len(tr))
	}

	// m2 starts while the plugin is down, the plugin recovers mid-play.
	d.tick(0, onDevice(movieAt("s1", "m2", 0, false)))
	d.tick(5, onDevice(movieAt("s1", "m2", 5000, false)))
	start2 := *d.watchdog("s1").StartedAt
	d.setPlugin(true)
	d.tick(5, onDevice(movieAt("s1", "m2", 10000, false)))
	d.tick(5) // m2 stopped while healthy: not written by the tracker

	// The plugin delivers both plays (it was recording all along).
	d.pluginDelivers(
		pluginPlay("plug-m1", "m1", *tr[0].StartedAt, 60*time.Second, false),
		pluginPlay("plug-m2", "m2", start2, 15*time.Second, false),
	)
	m1, m2 := d.rowsFor("m1"), d.rowsFor("m2")
	if len(m1) != 1 || m1[0].Id != "plug-m1" || m1[0].Source != "jellystics-plugin" {
		t.Fatalf("m1 rows = %+v (want the plugin row only)", m1)
	}
	if len(m2) != 1 || m2[0].Id != "plug-m2" {
		t.Fatalf("m2 rows = %+v (want exactly one, the plugin's)", m2)
	}
}

// Plugin → tracker: the plugin goes away mid-play. The tracker records the
// play (replacing the stale open checkpoint), never duplicates a play the
// plugin closed, and records new plays. When the plugin comes back with the
// closed version, it replaces the tracker row.
func TestSwitchOver_PluginToTracker(t *testing.T) {
	d := newDriver(t)
	d.setPlugin(true)

	// m1: seen by both, closed by the plugin; m2 in progress when the plugin dies.
	d.tick(0, onDevice(movieAt("s1", "m1", 0, false)))
	d.tick(0, onDevice(movieAt("s2", "m2", 0, false)), onDevice(movieAt("s1", "m1", 0, false)))
	for i := 1; i <= 6; i++ {
		d.tick(5, onDevice(movieAt("s1", "m1", int64(i)*5000, false)), onDevice(movieAt("s2", "m2", int64(i)*5000, false)))
	}
	start1, start2 := *d.watchdog("s1").StartedAt, *d.watchdog("s2").StartedAt
	d.pluginDelivers(
		pluginPlay("plug-m1", "m1", start1, 30*time.Second, false), // m1 closed by the plugin
		pluginPlay("plug-m2", "m2", start2, 20*time.Second, true),  // m2 open checkpoint
	)
	d.setPlugin(false) // plugin unreachable for more than 2 minutes

	// m1 stops (the tracker saw it late): covered by the closed plugin play → dropped.
	// m2 keeps playing then stops: the tracker records it, replacing the open checkpoint.
	d.tick(5, onDevice(movieAt("s2", "m2", 35000, false)))
	d.tick(5, onDevice(movieAt("s2", "m2", 40000, false)))
	d.tick(5)
	if m1 := d.rowsFor("m1"); len(m1) != 1 || m1[0].Id != "plug-m1" {
		t.Fatalf("m1 rows = %+v (want the plugin row only)", m1)
	}
	m2 := d.rowsFor("m2")
	if len(m2) != 1 || m2[0].Source != "watchdog" || *m2[0].PlaybackDuration < 39 {
		t.Fatalf("m2 rows = %+v (want one full tracker row)", m2)
	}

	// m3 entirely while the plugin is down: tracker row.
	d.playFor("s1", "m3")
	if m3 := d.rowsFor("m3"); len(m3) != 1 || m3[0].Source != "watchdog" {
		t.Fatalf("m3 rows = %+v", m3)
	}

	// The plugin comes back and re-sends m2 closed: it replaces the tracker row.
	d.setPlugin(true)
	d.pluginDelivers(pluginPlay("plug-m2", "m2", start2, 45*time.Second, false))
	if m2 := d.rowsFor("m2"); len(m2) != 1 || m2[0].Id != "plug-m2" || *m2[0].PlaybackDuration != 45 {
		t.Fatalf("m2 rows after plugin recovery = %+v", m2)
	}
	if n := len(d.plays()); n != 3 {
		t.Fatalf("total plays = %d, want 3", n)
	}
}
