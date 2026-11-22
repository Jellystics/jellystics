package jellyfin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// Jellystics plugin API (routes under /Jellystics/, admin or API key).
// JSON is camelCase; times are UTC ISO-8601.

// JellysticsTable is a table exposed by the plugin cursor API.
type JellysticsTable string

const (
	JellysticsPlays          JellysticsTable = "Plays"
	JellysticsLibraryEvents  JellysticsTable = "LibraryEvents"
	JellysticsUserDataEvents JellysticsTable = "UserDataEvents"
	JellysticsLogins         JellysticsTable = "Logins"
)

// JellysticsMaxPush is the maximum number of plays per POST /Jellystics/Import/Plays.
const JellysticsMaxPush = 500

// JellysticsSchemaVersion is the plugin API schema version this Jellystics speaks.
const JellysticsSchemaVersion = 1

// JellysticsInfo is GET /Jellystics/Info.
type JellysticsInfo struct {
	PluginVersion  string    `json:"pluginVersion"`
	SchemaVersion  int       `json:"schemaVersion"`
	Cursor         int64     `json:"cursor"`
	DatabaseId     string    `json:"databaseId"` // changes when the plugin data is reset; empty if unsupported
	FirstPlayAt    *FlexTime `json:"firstPlayAt"`
	ServerTimeZone string    `json:"serverTimeZone"`
	Counts         struct {
		Plays          int64 `json:"plays"`
		LibraryEvents  int64 `json:"libraryEvents"`
		UserDataEvents int64 `json:"userDataEvents"`
		Logins         int64 `json:"logins"`
	} `json:"counts"`
}

// JellysticsPage is one page of GET /Jellystics/<Table>?since=&limit=:
// rows with Version > since, ordered by Version. Items stays raw; decode it
// with Plays / LibraryEvents / UserDataEvents / Logins.
type JellysticsPage struct {
	Items   json.RawMessage `json:"items"`
	Next    int64           `json:"next"`
	HasMore bool            `json:"hasMore"`
}

func decodeItems[T any](p *JellysticsPage) ([]T, error) {
	var out []T
	if len(p.Items) == 0 || string(p.Items) == "null" {
		return out, nil
	}
	err := json.Unmarshal(p.Items, &out)
	return out, err
}

func (p *JellysticsPage) Plays() ([]JellysticsPlay, error) { return decodeItems[JellysticsPlay](p) }
func (p *JellysticsPage) LibraryEvents() ([]JellysticsLibraryEvent, error) {
	return decodeItems[JellysticsLibraryEvent](p)
}
func (p *JellysticsPage) UserDataEvents() ([]JellysticsUserDataEvent, error) {
	return decodeItems[JellysticsUserDataEvent](p)
}
func (p *JellysticsPage) Logins() ([]JellysticsLogin, error) { return decodeItems[JellysticsLogin](p) }

// EmbeddedJSON is a JSON document the plugin stores as text (StreamChanges,
// TranscodeReasons): it is sent as a JSON string containing JSON. Decoding
// also accepts the raw JSON value; encoding always produces the string form.
type EmbeddedJSON json.RawMessage

func (e *EmbeddedJSON) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*e = nil
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		if s == "" || !json.Valid([]byte(s)) {
			*e = nil
			return nil
		}
		*e = EmbeddedJSON(s)
		return nil
	}
	*e = append((*e)[:0], b...)
	return nil
}

func (e EmbeddedJSON) MarshalJSON() ([]byte, error) {
	if len(e) == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(string(e))
}

// JellysticsPlay is one playback recorded by the plugin (spec §3.1).
type JellysticsPlay struct {
	Id                  string       `json:"id"`
	PlaySessionId       *string      `json:"playSessionId"`
	UserId              string       `json:"userId"`
	UserName            *string      `json:"userName"`
	ItemId              string       `json:"itemId"`
	ItemName            *string      `json:"itemName"`
	ItemType            *string      `json:"itemType"`
	SeriesId            *string      `json:"seriesId"`
	SeriesName          *string      `json:"seriesName"`
	SeasonId            *string      `json:"seasonId"`
	AlbumId             *string      `json:"albumId"`
	AlbumName           *string      `json:"albumName"`
	ParentIndexNumber   *int         `json:"parentIndexNumber"`
	IndexNumber         *int         `json:"indexNumber"`
	MediaSourceId       *string      `json:"mediaSourceId"`
	LibraryId           *string      `json:"libraryId"`
	StartedAt           FlexTime     `json:"startedAt"`
	EndedAt             *FlexTime    `json:"endedAt"`
	WatchedMs           int64        `json:"watchedMs"`
	PausedMs            int64        `json:"pausedMs"`
	PauseCount          int          `json:"pauseCount"`
	SeekCount           int          `json:"seekCount"`
	SkippedMs           int64        `json:"skippedMs"`
	PositionStartTicks  *int64       `json:"positionStartTicks"`
	PositionEndTicks    *int64       `json:"positionEndTicks"`
	RunTimeTicks        *int64       `json:"runTimeTicks"`
	PlayedToCompletion  *bool        `json:"playedToCompletion"`
	DeviceId            *string      `json:"deviceId"`
	DeviceName          *string      `json:"deviceName"`
	Client              *string      `json:"client"`
	ApplicationVersion  *string      `json:"applicationVersion"`
	RemoteEndPoint      *string      `json:"remoteEndPoint"`
	IsRemote            *bool        `json:"isRemote"`
	PlayMethod          *string      `json:"playMethod"`
	AudioStreamIndex    *int         `json:"audioStreamIndex"`
	AudioLanguage       *string      `json:"audioLanguage"`
	AudioCodec          *string      `json:"audioCodec"`
	SubtitleStreamIndex *int         `json:"subtitleStreamIndex"`
	SubtitleLanguage    *string      `json:"subtitleLanguage"`
	StreamChanges       EmbeddedJSON `json:"streamChanges"`
	VideoCodecFrom      *string      `json:"videoCodecFrom"`
	VideoCodecTo        *string      `json:"videoCodecTo"`
	AudioCodecFrom      *string      `json:"audioCodecFrom"`
	AudioCodecTo        *string      `json:"audioCodecTo"`
	Width               *int         `json:"width"`
	Height              *int         `json:"height"`
	Bitrate             *int64       `json:"bitrate"`
	Framerate           *float64     `json:"framerate"`
	HwAccel             *string      `json:"hwAccel"`
	TranscodeReasons    EmbeddedJSON `json:"transcodeReasons"`
	VideoRange          *string      `json:"videoRange"`
	// Contract extension (12): Jellyfin TranscodingInfo direct flags and the
	// source video stream. All optional (null = unknown).
	IsVideoDirect  *bool    `json:"isVideoDirect"`
	IsAudioDirect  *bool    `json:"isAudioDirect"`
	SourceWidth    *int     `json:"sourceWidth"`
	SourceHeight   *int     `json:"sourceHeight"`
	VideoRangeType *string  `json:"videoRangeType"` // SDR, HDR10, HLG, DOVI...
	State          string   `json:"state"`          // open | closed
	LastEventAt    FlexTime `json:"lastEventAt"`
	Source         string   `json:"source"` // plugin | import-pbr | import-jellystics
	Version        int64    `json:"version"`
}

// JellysticsLibraryEvent is a library addition / removal (spec §3.2).
type JellysticsLibraryEvent struct {
	Id           string   `json:"id"`
	Kind         string   `json:"kind"`
	At           FlexTime `json:"at"`
	ItemId       *string  `json:"itemId"`
	ItemType     *string  `json:"itemType"`
	ItemName     *string  `json:"itemName"`
	SeriesId     *string  `json:"seriesId"`
	AlbumId      *string  `json:"albumId"`
	LibraryId    *string  `json:"libraryId"`
	RunTimeTicks *int64   `json:"runTimeTicks"`
	SizeBytes    *int64   `json:"sizeBytes"`
	Version      int64    `json:"version"`
}

// JellysticsUserDataEvent is a played / favorite change (spec §3.2).
type JellysticsUserDataEvent struct {
	Id      string   `json:"id"`
	At      FlexTime `json:"at"`
	UserId  *string  `json:"userId"`
	ItemId  *string  `json:"itemId"`
	Kind    string   `json:"kind"`
	Reason  *string  `json:"reason"`
	Version int64    `json:"version"`
}

// JellysticsLogin is a login attempt (spec §3.2).
type JellysticsLogin struct {
	Id                 string   `json:"id"`
	At                 FlexTime `json:"at"`
	UserId             *string  `json:"userId"`
	UserName           *string  `json:"userName"`
	DeviceId           *string  `json:"deviceId"`
	DeviceName         *string  `json:"deviceName"`
	Client             *string  `json:"client"`
	ApplicationVersion *string  `json:"applicationVersion"`
	RemoteEndPoint     *string  `json:"remoteEndPoint"`
	Success            bool     `json:"success"`
	Version            int64    `json:"version"`
}

// StatusError is an HTTP error status returned by Jellyfin.
type StatusError struct {
	Method, Path string
	Code         int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("%s %s: status %d", e.Method, e.Path, e.Code)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", AuthorizationHeader(c.apiKey))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return &StatusError{Method: method, Path: path, Code: resp.StatusCode}
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// GetJellysticsInfo returns the plugin info, or (nil, nil) when the plugin is
// not installed (Jellyfin answers 404 on /Jellystics/Info).
func (c *Client) GetJellysticsInfo(ctx context.Context) (*JellysticsInfo, error) {
	var info JellysticsInfo
	err := c.do(ctx, http.MethodGet, "/Jellystics/Info", nil, nil, &info)
	if se, ok := err.(*StatusError); ok && se.Code == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// GetJellysticsPage returns the rows of table with Version > since (limit
// 1..1000; 0 = plugin default 500).
func (c *Client) GetJellysticsPage(ctx context.Context, table JellysticsTable, since int64, limit int) (*JellysticsPage, error) {
	switch table {
	case JellysticsPlays, JellysticsLibraryEvents, JellysticsUserDataEvents, JellysticsLogins:
	default:
		return nil, fmt.Errorf("unknown Jellystics table %q", table)
	}
	q := url.Values{"since": []string{strconv.FormatInt(since, 10)}}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var page JellysticsPage
	if err := c.do(ctx, http.MethodGet, "/Jellystics/"+string(table), q, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// JellysticsImportResult is the response of POST /Jellystics/Import/Plays:
// plays received, imported (inserted or updated), skipped (already known /
// covered by a plugin play) and rejected (invalid).
type JellysticsImportResult struct {
	Received int `json:"received"`
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Rejected int `json:"rejected"`
}

// PostJellysticsPlays pushes Jellystics history to the plugin
// (POST /Jellystics/Import/Plays, idempotent upsert on Id, ≤ 500 plays).
// An empty 2xx body yields a zero result.
func (c *Client) PostJellysticsPlays(ctx context.Context, items []JellysticsPlay) (JellysticsImportResult, error) {
	var res JellysticsImportResult
	if len(items) > JellysticsMaxPush {
		return res, fmt.Errorf("too many plays in one push: %d (max %d)", len(items), JellysticsMaxPush)
	}
	if items == nil {
		items = []JellysticsPlay{}
	}
	err := c.do(ctx, http.MethodPost, "/Jellystics/Import/Plays", nil, map[string]any{"items": items}, &res)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return res, err
}
