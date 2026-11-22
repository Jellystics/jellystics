package statsql

import (
	"strings"
	"testing"
)

func TestTimezone(t *testing.T) {
	cases := map[string]string{
		"":                       "UTC",
		"Local":                  "UTC",
		"Europe/Paris":           "Europe/Paris",
		":America/New_York":      "America/New_York",
		"Nowhere/Land":           "UTC",
		"Europe/Paris'; DROP --": "UTC",
		"../../etc/passwd":       "UTC",
	}
	for in, want := range cases {
		t.Setenv("TZ", in)
		if got, loc := Timezone(); got != want || loc.String() != want {
			t.Errorf("TZ=%q → %q (%s), want %q", in, got, loc, want)
		}
	}
}

func TestLocalTSEmbedsValidatedZone(t *testing.T) {
	t.Setenv("TZ", "Europe/Paris")
	if got := LocalTS("a"); !strings.Contains(got, `AT TIME ZONE 'Europe/Paris'`) || !strings.Contains(got, `a."StartedAt"`) {
		t.Errorf("LocalTS = %s", got)
	}
}
