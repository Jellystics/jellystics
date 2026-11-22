package pluginsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
)

// History push (spec §6.2): the plays Jellystics recorded itself (tracker,
// legacy Playback Reporting / TSV imports, restores) are sent to the plugin
// in batches of 500 (POST /Jellystics/Import/Plays, idempotent upsert on Id,
// source import-jellystics). The progress lives in plugin_sync_state: an
// interrupted or failed push resumes after the last batch acknowledged.

// Push states (plugin_sync_state.push_state). PushInterrupted is never
// stored: it is a running state whose job is not running in this process
// (Jellystics restarted during the push).
const (
	PushIdle        = "idle"
	PushRunning     = "running"
	PushDone        = "done"
	PushError       = "error"
	PushInterrupted = "interrupted"
)

var (
	ErrPushRunning        = errors.New("a history push is already running")
	ErrNotConfigured      = errors.New("Jellyfin is not configured")
	ErrPluginAbsent       = errors.New("the Jellystics plugin is not installed")
	ErrPluginIncompatible = errors.New("the Jellystics plugin schema version is not supported")
)

// PushStatus is the read model of the history push.
type PushStatus struct {
	State      string     `json:"state"`
	Total      int64      `json:"total"`
	Sent       int64      `json:"sent"`
	Imported   int64      `json:"imported"`
	Skipped    int64      `json:"skipped"`
	Rejected   int64      `json:"rejected"`
	StartedAt  *time.Time `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
	Error      *string    `json:"error"`
	// Resumable: the next start continues after the last acknowledged batch.
	Resumable bool `json:"resumable"`
}

func (s *Service) pushStatusOf(st *models.PluginSyncState) PushStatus {
	state := st.PushState
	if state == "" {
		state = PushIdle
	}
	if state == PushRunning && !s.pushing {
		state = PushInterrupted
	}
	return PushStatus{
		State: state, Total: st.PushTotal, Sent: st.PushSent, Imported: st.PushImported,
		Skipped: st.PushSkipped, Rejected: st.PushRejected, StartedAt: st.PushStartedAt,
		FinishedAt: st.PushFinishedAt, Error: st.PushError,
		Resumable: (state == PushInterrupted || state == PushError) && st.PushCursor != "",
	}
}

// PushStatus returns the history push progress.
func (s *Service) PushStatus(ctx context.Context) (PushStatus, error) {
	st, err := s.repos.PluginSync.GetState(ctx)
	if err != nil {
		return PushStatus{}, err
	}
	s.pushMu.Lock()
	defer s.pushMu.Unlock()
	return s.pushStatusOf(st), nil
}

// StartPush checks the plugin and starts the history push in the background
// (resuming an interrupted / failed one). WaitPush waits for it to finish.
func (s *Service) StartPush(ctx context.Context) (PushStatus, error) {
	s.pushMu.Lock()
	defer s.pushMu.Unlock()
	if s.pushing {
		return PushStatus{}, ErrPushRunning
	}
	jf, err := s.client(ctx)
	if err != nil {
		return PushStatus{}, err
	}
	if jf == nil {
		return PushStatus{}, ErrNotConfigured
	}
	info, err := jf.GetJellysticsInfo(ctx)
	if err != nil {
		return PushStatus{}, fmt.Errorf("plugin info: %w", err)
	}
	if info == nil {
		return PushStatus{}, ErrPluginAbsent
	}
	if info.SchemaVersion != SupportedSchemaVersion {
		return PushStatus{}, ErrPluginIncompatible
	}
	st, err := s.repos.PluginSync.GetState(ctx)
	if err != nil {
		return PushStatus{}, err
	}
	resume := (st.PushState == PushRunning || st.PushState == PushError) && st.PushCursor != ""
	now := s.now()
	if !resume {
		st.PushCursor, st.PushSent, st.PushImported, st.PushSkipped, st.PushRejected = "", 0, 0, 0, 0
		st.PushStartedAt = &now
	}
	if st.PushTotal, err = s.repos.PluginSync.CountPushable(ctx); err != nil {
		return PushStatus{}, err
	}
	st.PushState, st.PushFinishedAt, st.PushError = PushRunning, nil, nil
	if err := s.repos.PluginSync.SavePushState(ctx, st); err != nil {
		return PushStatus{}, err
	}
	s.pushing = true
	out := s.pushStatusOf(st)
	s.pushWG.Add(1)
	go s.push(jf, st)
	return out, nil
}

// WaitPush blocks until the running history push (if any) has finished.
func (s *Service) WaitPush() { s.pushWG.Wait() }

func (s *Service) push(jf *jellyfin.Client, st *models.PluginSyncState) {
	defer s.pushWG.Done()
	ctx := context.Background()
	err := s.pushBatches(ctx, jf, st)
	now := s.now()
	st.PushFinishedAt = &now
	if err != nil {
		msg := err.Error()
		st.PushState, st.PushError = PushError, &msg
		log.Printf("[plugin-push] history push stopped after %d plays: %s (it resumes from there)", st.PushSent, msg)
	} else {
		st.PushState = PushDone
		log.Printf("[plugin-push] history push done: %d sent, %d imported, %d skipped, %d rejected",
			st.PushSent, st.PushImported, st.PushSkipped, st.PushRejected)
	}
	if serr := s.repos.PluginSync.SavePushState(ctx, st); serr != nil {
		log.Printf("[plugin-push] save state: %v", serr)
	}
	s.pushMu.Lock()
	s.pushing = false
	s.pushMu.Unlock()
}

func (s *Service) pushBatches(ctx context.Context, jf *jellyfin.Client, st *models.PluginSyncState) error {
	for {
		rows, err := s.repos.PluginSync.ListPushable(ctx, st.PushCursor, jellyfin.JellysticsMaxPush)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		items := make([]jellyfin.JellysticsPlay, len(rows))
		for i, a := range rows {
			items[i] = ToPlay(a)
		}
		res, err := jf.PostJellysticsPlays(ctx, items)
		if err != nil {
			return fmt.Errorf("push after %q: %w", st.PushCursor, err)
		}
		st.PushCursor = rows[len(rows)-1].Id
		st.PushSent += int64(len(rows))
		st.PushImported += int64(res.Imported)
		st.PushSkipped += int64(res.Skipped)
		st.PushRejected += int64(res.Rejected)
		if st.PushSent > st.PushTotal { // plays recorded during the push
			st.PushTotal = st.PushSent
		}
		if err := s.repos.PluginSync.SavePushState(ctx, st); err != nil {
			return err
		}
	}
}

// ToPlay maps a Jellystics play to the plugin Play contract (source
// import-jellystics, Id kept, watchedMs = PlaybackDuration × 1000, closed).
// Legacy columns follow the tracker layout: an episode's NowPlayingItemId is
// its series, a track's is its album.
func ToPlay(a models.JFPlaybackActivity) jellyfin.JellysticsPlay {
	str := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	itemId := str(a.ItemId)
	p := jellyfin.JellysticsPlay{
		Id: a.Id, PlaySessionId: a.PlaySessionId, UserId: str(a.UserId), UserName: a.UserName,
		ItemId: itemId, ItemName: a.NowPlayingItemName, ItemType: a.MediaType,
		PauseCount: a.PauseCount, PositionStartTicks: a.PositionTicksStart, PositionEndTicks: a.PositionTicksEnd,
		RunTimeTicks: a.RunTimeTicks, PlayedToCompletion: a.PlayedToCompletion,
		DeviceId: a.DeviceId, DeviceName: a.DeviceName, Client: a.Client, ApplicationVersion: a.ApplicationVersion,
		RemoteEndPoint: a.RemoteEndPoint, IsRemote: a.IsRemote, PlayMethod: a.PlayMethod,
		AudioStreamIndex: a.AudioStreamIndex, AudioLanguage: a.AudioLanguage,
		SubtitleStreamIndex: a.SubtitleStreamIndex, SubtitleLanguage: a.SubtitleLanguage,
		Bitrate: a.Bitrate, HwAccel: a.HwAccel, VideoRange: a.VideoRange,
		State: "closed", Source: "import-jellystics",
	}
	if len(a.TranscodeReasons) > 0 && string(a.TranscodeReasons) != "null" {
		p.TranscodeReasons = jellyfin.EmbeddedJSON(a.TranscodeReasons)
	}
	parent := str(a.NowPlayingItemId)
	if parent != "" && parent != itemId {
		switch str(a.MediaType) {
		case "Episode":
			p.SeriesId, p.SeriesName = &parent, a.SeriesName
		case "Audio":
			p.AlbumId, p.AlbumName = &parent, a.SeriesName
		}
	}
	if str(a.MediaType) == "Episode" {
		p.SeasonId = a.SeasonId
		if p.SeriesName == nil {
			p.SeriesName = a.SeriesName
		}
	}

	if a.PlaybackDuration != nil && *a.PlaybackDuration > 0 {
		p.WatchedMs = *a.PlaybackDuration * 1000
	}
	if a.PausedSeconds != nil && *a.PausedSeconds > 0 {
		p.PausedMs = *a.PausedSeconds * 1000
	}
	if a.SeekCount != nil {
		p.SeekCount = *a.SeekCount
	}
	if a.SkippedSeconds != nil && *a.SkippedSeconds > 0 {
		p.SkippedMs = int64(*a.SkippedSeconds) * 1000
	}
	if a.StartedAt != nil {
		p.StartedAt = jellyfin.FlexTime{Time: a.StartedAt.UTC()}
		end := a.StartedAt.Add(time.Duration(p.WatchedMs) * time.Millisecond)
		if a.EndedAt != nil && !a.EndedAt.Before(*a.StartedAt) {
			end = *a.EndedAt
		}
		e := jellyfin.FlexTime{Time: end.UTC()}
		p.EndedAt, p.LastEventAt = &e, e
	}

	// Transcoding target (TranscodingInfo) and source streams (MediaStreams).
	var ti struct {
		VideoCodec, AudioCodec       *string
		IsVideoDirect, IsAudioDirect *bool
		Width, Height                *int
		Framerate                    *float64
		Bitrate                      *int64
		HardwareAccelerationType     *string
		TranscodeReasons             json.RawMessage
	}
	if len(a.TranscodingInfo) > 0 && json.Unmarshal(a.TranscodingInfo, &ti) == nil {
		p.VideoCodecTo, p.AudioCodecTo = nonEmpty(ti.VideoCodec), nonEmpty(ti.AudioCodec)
		p.IsVideoDirect, p.IsAudioDirect = ti.IsVideoDirect, ti.IsAudioDirect
		p.Width, p.Height, p.Framerate = ti.Width, ti.Height, ti.Framerate
		if p.Bitrate == nil {
			p.Bitrate = ti.Bitrate
		}
		if p.HwAccel == nil {
			p.HwAccel = nonEmpty(ti.HardwareAccelerationType)
		}
		if len(p.TranscodeReasons) == 0 {
			p.TranscodeReasons = embeddedArray(ti.TranscodeReasons)
		}
	}
	var streams []struct {
		Type           string
		Codec          *string
		Index          *int
		IsDefault      bool
		IsExternal     bool
		Width, Height  *int
		VideoRange     *string
		VideoRangeType *string
	}
	if len(a.MediaStreams) > 0 && json.Unmarshal(a.MediaStreams, &streams) == nil {
		// Best stream of each type: the selected audio track, else the
		// default one, else the first (external video streams ignored).
		videoRank, audioRank := 0, 0
		for _, s := range streams {
			rank := 1
			if s.IsDefault {
				rank = 2
			}
			switch {
			case s.Type == "Video" && !s.IsExternal && rank > videoRank:
				videoRank = rank
				p.VideoCodecFrom, p.SourceWidth, p.SourceHeight = nonEmpty(s.Codec), s.Width, s.Height
				p.VideoRangeType = videoRange(s.VideoRangeType)
				if a.VideoRange == nil {
					p.VideoRange = videoRange(s.VideoRange)
				}
			case s.Type == "Audio":
				if a.AudioStreamIndex != nil && s.Index != nil && *s.Index == *a.AudioStreamIndex {
					rank = 3
				}
				if rank > audioRank {
					audioRank = rank
					p.AudioCodecFrom = nonEmpty(s.Codec)
				}
			}
		}
		p.AudioCodec = p.AudioCodecFrom
	}
	if len(a.PlayState) > 0 {
		var ps struct{ StreamChanges json.RawMessage }
		if json.Unmarshal(a.PlayState, &ps) == nil {
			p.StreamChanges = embeddedArray(ps.StreamChanges)
		}
	}
	return p
}

// embeddedArray keeps a non-empty JSON array.
func embeddedArray(raw json.RawMessage) jellyfin.EmbeddedJSON {
	t := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(t, "[") || t == "[]" {
		return nil
	}
	return jellyfin.EmbeddedJSON(t)
}
