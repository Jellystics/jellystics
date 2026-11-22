package sync

import (
	"testing"

	"github.com/Jellystics/Jellystics/internal/jellyfin"
)

func sp(s string) *string { return &s }

func TestTitleCase(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"rock", "Rock"},
		{"HIP HOP", "Hip Hop"},
		{"science fiction", "Science Fiction"},
		{"  spaced   out  ", "Spaced Out"},
		{"éléctronique", "Éléctronique"},
	}
	for _, tt := range tests {
		if got := titleCase(tt.in); got != tt.want {
			t.Fatalf("titleCase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestStrPtr(t *testing.T) {
	if strPtr("") != nil {
		t.Fatal("strPtr(\"\") should be nil")
	}
	p := strPtr("x")
	if p == nil || *p != "x" {
		t.Fatalf("strPtr(\"x\") = %v, want pointer to \"x\"", p)
	}
}

func TestDeref(t *testing.T) {
	if deref(nil) != "" {
		t.Fatal("deref(nil) should be empty string")
	}
	if deref(sp("y")) != "y" {
		t.Fatal("deref(&\"y\") should be \"y\"")
	}
}

// TestMapPluginRow covers the PlaybackReporting plugin row parser: JSON numbers
// arrive as float64, missing rowid rejects the row, short rows are ignored, and
// PlayDuration is preserved as-is (already seconds).
func TestMapPluginRow(t *testing.T) {
	t.Run("valid row", func(t *testing.T) {
		row := jellyfin.PlaybackReportingRow{
			"42", "2026-07-12 10:00:00", "user-1", "item-1", "Movie",
			"Big Buck Bunny", "DirectPlay", "Jellyfin Web", "Chrome", float64(600),
		}
		got := mapPluginRow(row)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.RowId != "42" {
			t.Fatalf("RowId = %q, want 42", got.RowId)
		}
		if got.PlayDuration == nil || *got.PlayDuration != 600 {
			t.Fatalf("PlayDuration = %v, want 600 (seconds preserved)", got.PlayDuration)
		}
		if got.ItemName == nil || *got.ItemName != "Big Buck Bunny" {
			t.Fatalf("ItemName = %v", got.ItemName)
		}
	})

	t.Run("short row rejected", func(t *testing.T) {
		if got := mapPluginRow(jellyfin.PlaybackReportingRow{"1", "2", "3"}); got != nil {
			t.Fatalf("short row should be nil, got %+v", got)
		}
	})

	t.Run("empty rowid rejected", func(t *testing.T) {
		row := jellyfin.PlaybackReportingRow{
			"", "d", "u", "i", "t", "n", "m", "c", "dev", float64(10),
		}
		if got := mapPluginRow(row); got != nil {
			t.Fatalf("empty rowid should be nil, got %+v", got)
		}
	})

	t.Run("nil fields become nil pointers", func(t *testing.T) {
		row := jellyfin.PlaybackReportingRow{
			"7", nil, nil, nil, nil, nil, nil, nil, nil, nil,
		}
		got := mapPluginRow(row)
		if got == nil {
			t.Fatal("row with valid rowid should not be nil")
		}
		if got.DateCreated != nil || got.UserId != nil || got.ItemName != nil {
			t.Fatal("nil source fields should map to nil pointers")
		}
		if got.PlayDuration == nil || *got.PlayDuration != 0 {
			t.Fatalf("nil duration should default to 0, got %v", got.PlayDuration)
		}
	})

	t.Run("negative duration clamped to 0", func(t *testing.T) {
		row := jellyfin.PlaybackReportingRow{
			"9", "d", "u", "i", "t", "n", "m", "c", "dev", float64(-5),
		}
		got := mapPluginRow(row)
		if got == nil || got.PlayDuration == nil || *got.PlayDuration != 0 {
			t.Fatalf("negative duration should clamp to 0, got %v", got)
		}
	})
}

const ticksPerSecond = 10_000_000
