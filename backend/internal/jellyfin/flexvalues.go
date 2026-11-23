package jellyfin

import (
	"bytes"
	"encoding/json"
	"strings"
)

// FlexString decodes a JSON string, number or bool into a string and any
// other value (null, object, array) into "", so a surprising enum encoding
// never fails the whole /Sessions decode.
type FlexString string

func (s *FlexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil
	}
	switch b[0] {
	case '"':
		var v string
		if err := json.Unmarshal(b, &v); err == nil {
			*s = FlexString(v)
		}
	case '{', '[':
	default:
		if string(b) != "null" {
			*s = FlexString(b)
		}
	}
	return nil
}

// FlexStringList decodes a [Flags] enum as Jellyfin serialises it: an array
// of strings (10.10+) or a comma-separated string (older). Anything else
// (null, a bare number) decodes to an empty list.
type FlexStringList []string

func (l *FlexStringList) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil
	}
	var out []string
	switch b[0] {
	case '[':
		var raw []json.RawMessage
		if err := json.Unmarshal(b, &raw); err != nil {
			return nil
		}
		for _, r := range raw {
			var s FlexString
			_ = s.UnmarshalJSON(r)
			if v := strings.TrimSpace(string(s)); v != "" {
				out = append(out, v)
			}
		}
	case '"':
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return nil
		}
		for _, p := range strings.Split(s, ",") {
			if v := strings.TrimSpace(p); v != "" && v != "None" {
				out = append(out, v)
			}
		}
	}
	*l = out
	return nil
}
