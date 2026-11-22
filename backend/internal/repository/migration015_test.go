package repository_test

import (
	"testing"

	"github.com/Jellystics/Jellystics/internal/migrations"
)

// Migration 015 rolls back / re-applies cleanly and flags the legacy
// Playback Reporting imports repaired before it (original duration still in
// the staging table).
func TestMigration015_DownUpBackfill(t *testing.T) {
	db := setupTestDB(t)
	run := func(file string) {
		b, err := migrations.SQL.ReadFile("sql/" + file)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(string(b)).Error; err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
	run("015_data_quality.down.sql")
	for _, q := range []string{
		`INSERT INTO jf_playback_reporting_plugin_data (rowid, "PlayDuration") VALUES ('10', -2147483648), ('11', 3000), ('12', 166552)`,
		`INSERT INTO jf_playback_activity ("Id", "PlaybackDuration", source, imported) VALUES
			('10', 0, 'plugin', true),      -- repaired negative
			('11', 3000, 'plugin', true),   -- untouched
			('12', 2700, 'plugin', true),   -- repaired overlong
			('13', 10, 'plugin', true),     -- no staging row
			('w1', 10, 'watchdog', false)`,
	} {
		if err := db.Exec(q).Error; err != nil {
			t.Fatal(err)
		}
	}
	run("015_data_quality.up.sql")
	var repaired []string
	if err := db.Raw(`SELECT "Id" FROM jf_playback_activity WHERE "DurationRepaired" ORDER BY "Id"`).Scan(&repaired).Error; err != nil {
		t.Fatal(err)
	}
	if len(repaired) != 2 || repaired[0] != "10" || repaired[1] != "12" {
		t.Fatalf("repaired = %v", repaired)
	}
	var state string
	if err := db.Raw(`SELECT push_state FROM plugin_sync_state WHERE id = 1`).Scan(&state).Error; err != nil || state != "idle" {
		t.Fatalf("push_state = %q, %v", state, err)
	}
}
