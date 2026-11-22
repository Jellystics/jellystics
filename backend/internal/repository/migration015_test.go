package repository_test

import (
	"fmt"
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
		`INSERT INTO jf_playback_reporting_plugin_data (rowid, "PlayDuration") VALUES ('10', -2147483648), ('11', 3000), ('12', 166552), ('14', 90000), ('15', 600)`,
		`INSERT INTO jf_playback_activity ("Id", "PlaybackDuration", source, imported) VALUES
			('10', 0, 'plugin', true),          -- repaired negative
			('11', 3000, 'plugin', true),       -- untouched
			('12', 2700, 'plugin', true),       -- repaired overlong
			('13', 10, 'plugin', true),         -- no staging row
			('plugin-14', 43200, 'plugin', true), -- older id format, repaired
			('plugin-15', 600, 'plugin', true),   -- older id format, untouched
			('import-u-i-2026', 0, 'import', true), -- TSV import: no staging row, unknown
			('15', 10, 'import', true),         -- TSV row whose id collides with a staging rowid: not legacy
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
	if fmt.Sprint(repaired) != "[10 12 plugin-14]" {
		t.Fatalf("repaired = %v", repaired)
	}
	var state string
	if err := db.Raw(`SELECT push_state FROM plugin_sync_state WHERE id = 1`).Scan(&state).Error; err != nil || state != "idle" {
		t.Fatalf("push_state = %q, %v", state, err)
	}
}
