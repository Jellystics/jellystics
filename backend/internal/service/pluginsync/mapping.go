package pluginsync

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"gorm.io/datatypes"
)

// minViewMs: a play is a view from 30 s of real watched time (as the tracker).
const minViewMs = 30_000

// trackedTypes maps the plugin ItemType to jf_playback_activity.MediaType.
// Other item types are not stored (the tracker ignores them too).
var trackedTypes = map[string]string{"Movie": "Movie", "Episode": "Episode", "Audio": "Audio"}

// normID returns Jellyfin ids in the N format (32 lowercase hex, no dashes)
// used everywhere in Jellystics; other values are returned unchanged.
func normID(s string) string {
	t := strings.ToLower(strings.ReplaceAll(s, "-", ""))
	if len(t) != 32 {
		return s
	}
	for _, c := range t {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return s
		}
	}
	return t
}

func normPtr(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	v := normID(*s)
	return &v
}

func nonEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

func roundSec(ms int64) int64 {
	if ms <= 0 {
		return 0
	}
	return (ms + 500) / 1000
}

// MapPlay converts a plugin play to a jf_playback_activity row (spec §7.3),
// filling the legacy columns exactly like the tracker does:
//   - Movie: NowPlayingItemId = movie, EpisodeId = nil
//   - Episode: NowPlayingItemId = series, EpisodeId = episode, SeasonId, SeriesName = series name
//   - Audio: NowPlayingItemId = album (else track), EpisodeId = track, SeriesName = album name
//
// ok is false for item types Jellystics does not track.
func MapPlay(p jellyfin.JellysticsPlay) (models.JFPlaybackActivity, bool) {
	mt, tracked := "", false
	if p.ItemType != nil {
		mt, tracked = trackedTypes[*p.ItemType]
	}
	if !tracked || p.Id == "" || p.ItemId == "" || p.StartedAt.IsZero() {
		return models.JFPlaybackActivity{}, false
	}

	itemId := normID(p.ItemId)
	userId := normID(p.UserId)
	parentId := itemId
	var childId, seasonId, groupName *string
	switch mt {
	case "Episode":
		if s := normPtr(p.SeriesId); s != nil {
			parentId = *s
			childId = &itemId
		}
		seasonId = normPtr(p.SeasonId)
		groupName = nonEmpty(p.SeriesName)
	case "Audio":
		childId = &itemId
		if a := normPtr(p.AlbumId); a != nil {
			parentId = *a
			groupName = nonEmpty(p.AlbumName)
		}
	}

	source := p.Source
	switch source {
	case repository.SourcePlugin, repository.SourceImportPBR, repository.SourceImportJellystics:
	default:
		source = repository.SourcePlugin
	}

	started := p.StartedAt.UTC()
	ended := p.LastEventAt.UTC() // open play: last checkpoint
	if p.EndedAt != nil && !p.EndedAt.IsZero() {
		ended = p.EndedAt.UTC()
	}
	if ended.IsZero() || ended.Before(started) {
		ended = started.Add(time.Duration(p.WatchedMs) * time.Millisecond)
	}
	startedStr := started.Format("2006-01-02 15:04:05.000-07:00")

	secs := roundSec(p.WatchedMs)
	paused := roundSec(p.PausedMs)
	skipped := int(roundSec(p.SkippedMs))
	seeks := p.SeekCount
	userName := nonEmpty(p.UserName)
	if userName == nil {
		userName = &userId
	}
	isPaused := false
	name := nonEmpty(p.ItemName)

	a := models.JFPlaybackActivity{
		Id:                   p.Id,
		IsPaused:             &isPaused,
		UserId:               &userId,
		UserName:             userName,
		Client:               nonEmpty(p.Client),
		DeviceName:           nonEmpty(p.DeviceName),
		DeviceId:             nonEmpty(p.DeviceId),
		ApplicationVersion:   nonEmpty(p.ApplicationVersion),
		NowPlayingItemId:     &parentId,
		NowPlayingItemName:   name,
		EpisodeId:            childId,
		SeasonId:             seasonId,
		SeriesName:           groupName,
		PlaybackDuration:     &secs,
		PlayMethod:           nonEmpty(p.PlayMethod),
		ActivityDateInserted: &startedStr,
		RemoteEndPoint:       nonEmpty(p.RemoteEndPoint),
		Source:               source,
		Imported:             source != repository.SourcePlugin,
		ItemId:               &itemId,
		MediaType:            &mt,
		RunTimeTicks:         p.RunTimeTicks,
		PositionTicksStart:   p.PositionStartTicks,
		PositionTicksEnd:     p.PositionEndTicks,
		PausedSeconds:        &paused,
		StartedAt:            &started,
		EndedAt:              &ended,
		IsView:               p.WatchedMs >= minViewMs,
		AudioStreamIndex:     p.AudioStreamIndex,
		AudioLanguage:        nonEmpty(p.AudioLanguage),
		SubtitleStreamIndex:  p.SubtitleStreamIndex,
		SubtitleLanguage:     nonEmpty(p.SubtitleLanguage),
		TranscodeReasons:     reasonsJSON(p.TranscodeReasons),
		Bitrate:              p.Bitrate,
		HwAccel:              hwAccel(p.HwAccel),
		VideoRange:           videoRange(p.VideoRange),
		PauseCount:           p.PauseCount,
		IsRemote:             p.IsRemote,
		PlaySessionId:        nonEmpty(p.PlaySessionId),
		SeekCount:            &seeks,
		SkippedSeconds:       &skipped,
		PlayedToCompletion:   p.PlayedToCompletion,
		PlayState:            playState(p),
		TranscodingInfo:      transcodingInfo(p),
		MediaStreams:         mediaStreams(p),
	}
	if a.SubtitleStreamIndex != nil && *a.SubtitleStreamIndex < 0 {
		a.SubtitleStreamIndex = nil // -1 = none
	}
	return a, true
}

func hwAccel(s *string) *string {
	if s == nil || *s == "" || *s == "None" {
		return nil
	}
	return s
}

func videoRange(s *string) *string {
	if s == nil || *s == "" || *s == "Unknown" {
		return nil
	}
	return s
}

// reasonsJSON keeps a non-empty JSON array of transcode reasons.
func reasonsJSON(e jellyfin.EmbeddedJSON) datatypes.JSON {
	var arr []string
	if len(e) == 0 || json.Unmarshal(e, &arr) != nil || len(arr) == 0 {
		return nil
	}
	b, _ := json.Marshal(arr)
	return datatypes.JSON(b)
}

func marshal(v any) datatypes.JSON {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return datatypes.JSON(b)
}

// playState keeps the plugin play state: State "open" marks a play still in
// progress (repository.PluginPlayOpen), used by the tracker switch-over.
func playState(p jellyfin.JellysticsPlay) datatypes.JSON {
	st := p.State
	if st == "" {
		st = "closed"
	}
	m := map[string]any{
		"State":         st,
		"Version":       p.Version,
		"PositionTicks": p.PositionEndTicks,
		"IsPaused":      false,
		"PlayMethod":    p.PlayMethod,
	}
	if !p.LastEventAt.IsZero() {
		m["LastEventAt"] = p.LastEventAt.UTC().Format(time.RFC3339Nano)
	}
	if len(p.StreamChanges) > 0 && string(p.StreamChanges) != "[]" {
		m["StreamChanges"] = json.RawMessage(p.StreamChanges)
	}
	return marshal(m)
}

// transcodingInfo rebuilds the Jellyfin TranscodingInfo subset read by the
// playback quality analyses, when the play was transcoded / remuxed.
func transcodingInfo(p jellyfin.JellysticsPlay) datatypes.JSON {
	if p.VideoCodecTo == nil && p.AudioCodecTo == nil && p.Width == nil {
		return nil
	}
	m := map[string]any{}
	if p.VideoCodecTo != nil {
		m["VideoCodec"] = *p.VideoCodecTo
		m["IsVideoDirect"] = p.VideoCodecFrom != nil && strings.EqualFold(*p.VideoCodecFrom, *p.VideoCodecTo)
	}
	if p.AudioCodecTo != nil {
		m["AudioCodec"] = *p.AudioCodecTo
		m["IsAudioDirect"] = p.AudioCodecFrom != nil && strings.EqualFold(*p.AudioCodecFrom, *p.AudioCodecTo)
	}
	if p.Width != nil {
		m["Width"] = *p.Width
	}
	if p.Height != nil {
		m["Height"] = *p.Height
	}
	if p.Bitrate != nil {
		m["Bitrate"] = *p.Bitrate
	}
	if p.Framerate != nil {
		m["Framerate"] = *p.Framerate
	}
	if h := hwAccel(p.HwAccel); h != nil {
		m["HardwareAccelerationType"] = *h
	}
	if r := reasonsJSON(p.TranscodeReasons); r != nil {
		m["TranscodeReasons"] = json.RawMessage(r)
	}
	return marshal(m)
}

// mediaStreams rebuilds the source streams known by the plugin (codecs,
// selected audio / subtitle) in the Jellyfin MediaStream shape.
func mediaStreams(p jellyfin.JellysticsPlay) datatypes.JSON {
	var streams []map[string]any
	if p.VideoCodecFrom != nil {
		v := map[string]any{"Type": "Video", "Codec": *p.VideoCodecFrom, "IsDefault": true}
		if r := videoRange(p.VideoRange); r != nil {
			v["VideoRange"] = *r
		}
		streams = append(streams, v)
	}
	audioCodec := p.AudioCodecFrom
	if audioCodec == nil {
		audioCodec = p.AudioCodec
	}
	if audioCodec != nil || p.AudioStreamIndex != nil {
		a := map[string]any{"Type": "Audio", "IsDefault": true}
		if audioCodec != nil {
			a["Codec"] = *audioCodec
		}
		if p.AudioStreamIndex != nil {
			a["Index"] = *p.AudioStreamIndex
		}
		if l := nonEmpty(p.AudioLanguage); l != nil {
			a["Language"] = *l
		}
		streams = append(streams, a)
	}
	if p.SubtitleStreamIndex != nil && *p.SubtitleStreamIndex >= 0 {
		s := map[string]any{"Type": "Subtitle", "Index": *p.SubtitleStreamIndex}
		if l := nonEmpty(p.SubtitleLanguage); l != nil {
			s["Language"] = *l
		}
		streams = append(streams, s)
	}
	if len(streams) == 0 {
		return nil
	}
	return marshal(streams)
}

func mapLibraryEvents(rows []jellyfin.JellysticsLibraryEvent) []models.JFLibraryEvent {
	out := make([]models.JFLibraryEvent, 0, len(rows))
	for _, r := range rows {
		if r.Id == "" {
			continue
		}
		out = append(out, models.JFLibraryEvent{
			Id: r.Id, Kind: r.Kind, At: r.At.UTC(),
			ItemId: normPtr(r.ItemId), ItemType: nonEmpty(r.ItemType), ItemName: nonEmpty(r.ItemName),
			SeriesId: normPtr(r.SeriesId), AlbumId: normPtr(r.AlbumId), LibraryId: normPtr(r.LibraryId),
			RunTimeTicks: r.RunTimeTicks, SizeBytes: r.SizeBytes, Version: r.Version,
		})
	}
	return out
}

func mapUserDataEvents(rows []jellyfin.JellysticsUserDataEvent) []models.JFUserDataEvent {
	out := make([]models.JFUserDataEvent, 0, len(rows))
	for _, r := range rows {
		if r.Id == "" {
			continue
		}
		out = append(out, models.JFUserDataEvent{
			Id: r.Id, At: r.At.UTC(), UserId: normPtr(r.UserId), ItemId: normPtr(r.ItemId),
			Kind: r.Kind, Reason: nonEmpty(r.Reason), Version: r.Version,
		})
	}
	return out
}

func mapLogins(rows []jellyfin.JellysticsLogin) []models.JFUserLogin {
	out := make([]models.JFUserLogin, 0, len(rows))
	for _, r := range rows {
		if r.Id == "" {
			continue
		}
		out = append(out, models.JFUserLogin{
			Id: r.Id, At: r.At.UTC(), UserId: normPtr(r.UserId), UserName: nonEmpty(r.UserName),
			DeviceId: nonEmpty(r.DeviceId), DeviceName: nonEmpty(r.DeviceName), Client: nonEmpty(r.Client),
			ApplicationVersion: nonEmpty(r.ApplicationVersion), RemoteEndPoint: nonEmpty(r.RemoteEndPoint),
			Success: r.Success, Version: r.Version,
		})
	}
	return out
}
