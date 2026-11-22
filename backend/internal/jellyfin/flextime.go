package jellyfin

import (
	"strings"
	"time"
)

// FlexTime parses Jellyfin DateTime values, which may come with or without a
// zone suffix and with 7 fractional digits (.NET). Unparseable or default
// values (0001-01-01) decode to the zero time instead of failing the whole
// response.
type FlexTime struct{ time.Time }

func (t *FlexTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.9999999", "2006-01-02T15:04:05"} {
		if v, err := time.Parse(layout, s); err == nil {
			if v.Year() > 1 {
				t.Time = v
			}
			return nil
		}
	}
	return nil
}

func (t FlexTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + t.UTC().Format(time.RFC3339Nano) + `"`), nil
}
