package repository_test

import (
	"testing"

	"github.com/Jellystics/Jellystics/internal/migrations"
)

// Migration 013 can be rolled back and re-applied cleanly.
func TestMigration013_DownUp(t *testing.T) {
	db := setupTestDB(t)
	exists := func(sql string, args ...any) bool {
		var ok bool
		if err := db.Raw(sql, args...).Scan(&ok).Error; err != nil {
			t.Fatal(err)
		}
		return ok
	}
	table := func(name string) bool { return exists(`SELECT to_regclass(?) IS NOT NULL`, name) }
	column := func(name string) bool {
		return exists(`SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'jf_playback_activity' AND column_name = ?)`, name)
	}
	run := func(file string) {
		b, err := migrations.SQL.ReadFile("sql/" + file)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(string(b)).Error; err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}

	if !table("plugin_sync_state") || !column("PluginOpen") || !table("idx_playback_user_item_start") {
		t.Fatal("013 not applied")
	}
	run("013_plugin_sync.down.sql")
	for _, tb := range []string{"plugin_sync_state", "jf_library_events", "jf_user_data_events", "jf_user_logins", "idx_playback_user_item_start"} {
		if table(tb) {
			t.Errorf("%s still exists after down", tb)
		}
	}
	for _, c := range []string{"PlaySessionId", "SeekCount", "SkippedSeconds", "PlayedToCompletion", "PluginOpen"} {
		if column(c) {
			t.Errorf("column %s still exists after down", c)
		}
	}
	if !column("DeviceId") {
		t.Error("DeviceId (001) must be kept")
	}
	run("013_plugin_sync.up.sql")
	if !table("plugin_sync_state") || !column("PluginOpen") || !exists(`SELECT EXISTS (SELECT 1 FROM plugin_sync_state WHERE id = 1)`) {
		t.Fatal("013 not re-applied")
	}
}
