package sync

import (
	"context"
	"testing"
)

func pbrRow() []any {
	return []any{"1", "2026-11-01 20:00:00", "user-1", "m1", "Movie", "Movie", "DirectPlay", "Jellyfin Web", "TV", float64(600)}
}

func TestSyncPlaybackPlugin_RunsWithoutJellysticsPlugin(t *testing.T) {
	mock := newMockJF(t)
	mock.pbrRows = [][]any{pbrRow()}
	svc, db := newTestService(t, mock)
	if err := svc.SyncPlaybackPlugin(context.Background()); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Raw(`SELECT COUNT(*) FROM jf_playback_reporting_plugin_data`).Scan(&n)
	if mock.pbrQueries != 1 || n != 1 {
		t.Fatalf("queries = %d, staged = %d; want the legacy sync unchanged", mock.pbrQueries, n)
	}
}

func TestSyncPlaybackPlugin_SkippedWhenJellysticsPluginDetected(t *testing.T) {
	mock := newMockJF(t)
	mock.pbrRows = [][]any{pbrRow()}
	mock.jellysticsPlugin = true
	svc, db := newTestService(t, mock)
	if err := svc.SyncPlaybackPlugin(context.Background()); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Raw(`SELECT COUNT(*) FROM jf_playback_reporting_plugin_data`).Scan(&n)
	if mock.pbrQueries != 0 || n != 0 {
		t.Fatalf("queries = %d, staged = %d; want the legacy sync skipped", mock.pbrQueries, n)
	}
}
