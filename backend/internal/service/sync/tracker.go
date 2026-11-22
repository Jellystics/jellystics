package sync

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/netutil"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Playback tracking.
//
// Jellystics polls GET /Sessions every PollInterval and accumulates, per
// session, the real playback time of the item being played. How Jellyfin
// behaves (verified in jellyfin/jellyfin source):
//
//   - PlayState.PositionTicks is whatever the client last reported, *plus* a
//     server-side extrapolation of +1 s every second while not paused
//     (SessionInfo.StartAutomaticProgress). So the position advances smoothly
//     between client reports, and stays frozen while paused.
//   - The extrapolation continues when a client dies silently, until the idle
//     check (every 5 min, >5 min without a real report) stops the playback.
//     LastPlaybackCheckIn is only updated by *real* client reports while
//     LastActivityDate is also bumped by the extrapolation, so their gap tells
//     whether a client that normally reports periodically went silent.
//   - When playback stops, NowPlayingItem is cleared immediately, and the stop
//     position is saved to the user's item data (PlaybackPositionTicks, reset
//     to 0 with Played=true past 90%).
//
// Accounting rules:
//   - Watched time = forward progress of the media position, in milliseconds,
//     each step clamped to the elapsed wall time (+ a small slack absorbing the
//     1 s granularity of the extrapolation). Pauses, buffering and backward
//     seeks add nothing; a forward seek adds at most the elapsed time.
//   - The total is also clamped to wall time since the playback started.
//   - Time is not counted while a periodically-reporting client is silent.
//   - When a play ends (stop, item switch, replay), the time between the last
//     poll and the actual stop is recovered from the stop position saved in
//     the user's item data, when it is consistent.
//   - Every playback attempt is recorded (a "click"); it is a "view" when the
//     watched time is >= MinViewSeconds.
//   - Only movies, episodes and music tracks are tracked.

const (
	// PollInterval is how often SessionTick should be called.
	PollInterval = 5 * time.Second

	MinViewSeconds = 30

	ticksPerMs = 10_000 // PositionTicks are 100 ns units

	stepSlackMs       = 2000             // tolerance per step (1 s extrapolation granularity + poll jitter)
	fallbackMaxStepMs = 15000            // cap per step when no position is available
	periodicMaxGap    = 30 * time.Second // check-ins closer than this ⇒ client reports periodically
	silentClientAfter = 45 * time.Second // periodic client silent this long ⇒ stop counting
	userDataTimeout   = 5 * time.Second

	// maxTailMs bounds the time credited between the last poll and the stop.
	maxTailMs = 2*int64(PollInterval/time.Millisecond) + int64(silentClientAfter/time.Millisecond)
)

// trackedTypes maps Jellyfin item types to the MediaType stored on plays.
var trackedTypes = map[string]string{
	"Movie":   "Movie",
	"Episode": "Episode",
	"Audio":   "Audio",
}

// observation is the tracking-relevant view of one live Jellyfin session.
type observation struct {
	sess      jellyfin.SessionInfo
	itemId    string  // movie / episode / track id
	parentId  string  // movie id, series id or album id (legacy NowPlayingItemId)
	childId   *string // episode / track id (legacy EpisodeId), nil for movies
	mediaType string
	seasonId  *string
	groupName *string // series or album name
	runtime   *int64
	pos       *int64
	paused    bool
	checkIn   time.Time // LastPlaybackCheckIn (zero if unknown)
	activity  time.Time // LastActivityDate (server clock, bumped every second while playing)
	silentFor time.Duration
	playlist  string
	capture   capture
}

// observe extracts an observation from a session, or ok=false when the
// session is not playing a tracked item.
func observe(sess jellyfin.SessionInfo) (observation, bool) {
	it := sess.NowPlayingItem
	if it == nil || it.Id == "" {
		return observation{}, false
	}
	mt, ok := trackedTypes[it.Type]
	if !ok {
		return observation{}, false
	}
	if it.ExtraType != nil && *it.ExtraType != "" {
		return observation{}, false // theme songs / videos, trailers
	}
	o := observation{
		sess:      sess,
		itemId:    it.Id,
		parentId:  it.Id,
		mediaType: mt,
		seasonId:  it.SeasonId,
		groupName: it.SeriesName,
		runtime:   it.RunTimeTicks,
	}
	switch {
	case mt == "Episode" && it.SeriesId != nil && *it.SeriesId != "":
		o.parentId = *it.SeriesId
		o.childId = &o.itemId
	case mt == "Audio" && it.AlbumId != nil && *it.AlbumId != "":
		o.parentId = *it.AlbumId
		o.childId = &o.itemId
		o.groupName = it.Album
		o.seasonId = nil
	case mt == "Audio":
		o.childId = &o.itemId // track without album
		o.seasonId = nil
	}
	if sess.PlayState != nil {
		o.paused = sess.PlayState.IsPaused
		o.pos = sess.PlayState.PositionTicks
	}
	o.checkIn = sess.LastPlaybackCheckIn.Time
	o.activity = sess.LastActivityDate.Time
	if !o.checkIn.IsZero() && !sess.LastActivityDate.IsZero() {
		o.silentFor = sess.LastActivityDate.Sub(o.checkIn)
	}
	if sess.PlaylistItemId != nil {
		o.playlist = *sess.PlaylistItemId
	}
	o.capture = captureFrom(sess)
	return o, true
}

// capture holds the per-poll playback fields recorded for analytics (audio /
// subtitle choice, video range, network locality). It never feeds the
// time accounting.
type capture struct {
	audioIdx, subIdx   *int
	audioLang, subLang *string
	videoRange         *string
	remote             *bool
	reasons            []string
	bitrate            *int64
	bitrateTranscode   bool // bitrate comes from TranscodingInfo (authoritative)
	hwAccel            *string
}

func captureFrom(sess jellyfin.SessionInfo) capture {
	var c capture
	streams := sess.NowPlayingItem.MediaStreams
	var audio *jellyfin.MediaStream
	if ps := sess.PlayState; ps != nil {
		if ps.AudioStreamIndex != nil && *ps.AudioStreamIndex >= 0 {
			audio = findStream(streams, "Audio", *ps.AudioStreamIndex)
			c.audioIdx = ps.AudioStreamIndex
		}
		if ps.SubtitleStreamIndex != nil && *ps.SubtitleStreamIndex >= 0 {
			c.subIdx = ps.SubtitleStreamIndex // never falls back: none means none
			if st := findStream(streams, "Subtitle", *ps.SubtitleStreamIndex); st != nil {
				c.subLang = langPtr(st)
			}
		}
	}
	if c.audioIdx != nil && audio != nil {
		c.audioLang = langPtr(audio)
	}
	if c.audioIdx == nil {
		// Client did not report its audio track: it stays unknown (NULL) so the
		// VO / VF analyses never guess. The default (else first) stream only
		// feeds the bitrate estimate below.
		audio = defaultAudio(streams)
	}
	video := pickVideo(streams)
	if video != nil {
		// VideoRangeType (SDR, HDR10, HLG, DOVI...) is finer than VideoRange (SDR/HDR).
		r := string(video.VideoRangeType)
		if r == "" || r == "Unknown" {
			r = string(video.VideoRange)
		}
		if r != "" && r != "Unknown" {
			c.videoRange = &r
		}
	}
	if ti := sess.TranscodingInfo; ti != nil {
		c.reasons = ti.TranscodeReasons
		if ti.Bitrate != nil && *ti.Bitrate > 0 {
			c.bitrate, c.bitrateTranscode = ti.Bitrate, true
		}
		if hw := string(ti.HardwareAccelerationType); hw != "" && hw != "None" {
			c.hwAccel = &hw
		}
	}
	if c.bitrate == nil {
		c.bitrate = mediaBitrate(sess.NowPlayingItem, video, audio)
	}
	if sess.RemoteEndPoint != nil {
		c.remote = isRemoteEndpoint(*sess.RemoteEndPoint)
	}
	return c
}

func langPtr(st *jellyfin.MediaStream) *string {
	if st == nil || st.Language == nil || *st.Language == "" {
		return nil
	}
	l := *st.Language
	return &l
}

// findStream returns the stream of the given type with the given Index.
func findStream(streams []jellyfin.MediaStream, typ string, idx int) *jellyfin.MediaStream {
	for i := range streams {
		if streams[i].Index == idx && streams[i].Type == typ {
			return &streams[i]
		}
	}
	return nil
}

// defaultAudio returns the IsDefault audio stream, else the first audio stream.
func defaultAudio(streams []jellyfin.MediaStream) *jellyfin.MediaStream {
	var first *jellyfin.MediaStream
	for i := range streams {
		if streams[i].Type != "Audio" {
			continue
		}
		if streams[i].IsDefault {
			return &streams[i]
		}
		if first == nil {
			first = &streams[i]
		}
	}
	return first
}

// pickVideo prefers an embedded (non-external) stream, then IsDefault, then
// the lowest Index.
func pickVideo(streams []jellyfin.MediaStream) *jellyfin.MediaStream {
	var best *jellyfin.MediaStream
	better := func(a, b *jellyfin.MediaStream) bool { // a better than b
		if a.IsExternal != b.IsExternal {
			return !a.IsExternal
		}
		if a.IsDefault != b.IsDefault {
			return a.IsDefault
		}
		return a.Index < b.Index
	}
	for i := range streams {
		if streams[i].Type != "Video" {
			continue
		}
		if best == nil || better(&streams[i], best) {
			best = &streams[i]
		}
	}
	return best
}

// mediaBitrate is the bitrate of the media being direct-played: the first
// MediaSource's, else video + selected audio stream bitrates.
func mediaBitrate(it *jellyfin.SessionItem, video, audio *jellyfin.MediaStream) *int64 {
	if len(it.MediaSources) > 0 && it.MediaSources[0].Bitrate != nil && *it.MediaSources[0].Bitrate > 0 {
		b := *it.MediaSources[0].Bitrate
		return &b
	}
	var sum int64
	for _, st := range []*jellyfin.MediaStream{video, audio} {
		if st != nil && st.BitRate != nil && *st.BitRate > 0 {
			sum += *st.BitRate
		}
	}
	if sum == 0 {
		return nil
	}
	return &sum
}

// isRemoteEndpoint classifies a session's RemoteEndPoint (IP, optionally with
// a port or IPv4-mapped IPv6): RFC1918, loopback, link-local and ULA are
// local. Empty or unparseable returns nil (unknown).
func isRemoteEndpoint(s string) *bool {
	addr, ok := netutil.Parse(s)
	if !ok {
		return nil
	}
	remote := !netutil.IsLocal(addr) // CGNAT (100.64/10) counts as remote
	return &remote
}

func ms(d time.Duration) int64 { return d.Milliseconds() }

func minI64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// elapsedMs returns the wall time since the previous poll of this entry.
func elapsedMs(prev *models.JFActivityWatchdog, now time.Time) int64 {
	if prev.LastTickAt == nil {
		return 0
	}
	if e := ms(now.Sub(*prev.LastTickAt)); e > 0 {
		return e
	}
	return 0
}

func samePlay(prev *models.JFActivityWatchdog, o observation, now time.Time) bool {
	if prev.ItemId == nil || *prev.ItemId != o.itemId {
		return false
	}
	if prev.UserId != nil && *prev.UserId != o.sess.UserId {
		return false // shared device (10.10 session key is app+device): user switched
	}
	// Same item: a jump back to the very beginning is a new play when the
	// previous one was near its end (repeat-one / play again) or the client
	// started a new queue entry.
	if o.pos == nil || prev.PlaybackDuration == nil || *o.pos >= *prev.PlaybackDuration {
		return true
	}
	if *o.pos/ticksPerMs > elapsedMs(prev, now)+stepSlackMs {
		return true // a backward seek, not a restart
	}
	// Looped: the rest of the previous play plus the new position fits in the
	// elapsed time (exact for any track length and poll interval).
	nearEnd := prev.RunTimeTicks != nil && *prev.RunTimeTicks > 0 &&
		(*prev.RunTimeTicks-*prev.PlaybackDuration)/ticksPerMs+*o.pos/ticksPerMs <= elapsedMs(prev, now)+stepSlackMs
	var prevPlaylist string
	if prev.PlayState != nil {
		var ps struct{ PlaylistItemId string }
		_ = json.Unmarshal(prev.PlayState, &ps)
		prevPlaylist = ps.PlaylistItemId
	}
	playlistChanged := o.playlist != "" && prevPlaylist != "" && o.playlist != prevPlaylist
	return !(nearEnd || playlistChanged)
}

// accumulate advances an entry for the same play by one poll.
func accumulate(prev models.JFActivityWatchdog, o observation, now time.Time) models.JFActivityWatchdog {
	el := elapsedMs(&prev, now)
	next := prev

	// Does this client report progress periodically? (sticky per session)
	if !o.checkIn.IsZero() && prev.LastCheckIn != nil && o.checkIn.After(*prev.LastCheckIn) &&
		o.checkIn.Sub(*prev.LastCheckIn) <= periodicMaxGap {
		next.PeriodicReporter = true
	}
	silent := next.PeriodicReporter && !o.paused && o.silentFor > silentClientAfter

	var inc int64
	switch {
	case silent:
		inc = 0
	case o.pos != nil && prev.PlaybackDuration != nil:
		if d := (*o.pos - *prev.PlaybackDuration) / ticksPerMs; d > 0 {
			inc = minI64(d, el+stepSlackMs)
		}
	case !o.paused:
		inc = minI64(el, fallbackMaxStepMs)
	}
	if prev.StartedAt != nil {
		budget := ms(now.Sub(*prev.StartedAt)) + stepSlackMs - prev.WatchedMs
		if inc > budget {
			inc = budget
		}
	}
	if inc < 0 {
		inc = 0
	}
	next.WatchedMs = prev.WatchedMs + inc
	if o.paused {
		next.PausedMs = prev.PausedMs + el
		if prev.IsPaused != nil && !*prev.IsPaused {
			next.PauseCount = prev.PauseCount + 1 // playing -> paused transition
		}
	}
	return next
}

// headCreditMs is the playback time already elapsed when a play is first
// seen: its position, if it plausibly started from the beginning since the
// previous poll (sinceMs), else 0 (resumed mid-way: start unknown).
func headCreditMs(o observation, sinceMs int64) int64 {
	if o.pos == nil || *o.pos <= 0 {
		return 0
	}
	p := *o.pos / ticksPerMs
	if p > sinceMs+stepSlackMs {
		return 0
	}
	return minI64(p, sinceMs)
}

// headFromCheckIn is the head credit for a play first seen mid-way (resumed):
// the time since the client's "Playing" report, i.e. LastActivityDate minus
// LastPlaybackCheckIn, both on the Jellyfin server clock. Bounded by capMs.
func headFromCheckIn(o observation, capMs int64) int64 {
	if o.paused || o.silentFor <= 0 || capMs <= 0 {
		return 0
	}
	return minI64(ms(o.silentFor), capMs)
}

// firstHead combines both head estimates.
func firstHead(o observation, sinceMs int64) int64 {
	if h := headCreditMs(o, sinceMs); h > 0 {
		return h
	}
	return headFromCheckIn(o, sinceMs+stepSlackMs)
}

// newEntry starts tracking a new play.
func newEntry(o observation, now time.Time, headMs int64, periodic bool) models.JFActivityWatchdog {
	s := o.sess
	started := now.Add(-time.Duration(headMs) * time.Millisecond)
	startedStr := started.Format("2006-01-02 15:04:05.000-07:00")
	actId := uuid.NewString()
	e := models.JFActivityWatchdog{
		Id:                   s.Id,
		ActivityId:           &actId,
		UserId:               strPtr(s.UserId),
		UserName:             strPtr(s.UserName),
		NowPlayingItemId:     &o.parentId,
		EpisodeId:            o.childId,
		SeasonId:             o.seasonId,
		SeriesName:           o.groupName,
		ActivityDateInserted: &startedStr,
		ItemId:               &o.itemId,
		MediaType:            &o.mediaType,
		PositionTicksStart:   o.pos,
		WatchedMs:            headMs,
		StartedAt:            &started,
		PeriodicReporter:     periodic,
	}
	return e
}

// refresh copies the per-poll session state onto an entry.
func refresh(e *models.JFActivityWatchdog, o observation, now time.Time) {
	s := o.sess
	name := s.NowPlayingItem.Name
	e.NowPlayingItemName = &name
	e.Client = strPtr(s.Client)
	e.DeviceName = strPtr(s.DeviceName)
	e.DeviceId = strPtr(s.DeviceId)
	e.ApplicationVersion = strPtr(s.ApplicationVersion)
	e.RemoteEndPoint = s.RemoteEndPoint
	e.ServerId = s.ServerId
	e.RunTimeTicks = o.runtime
	e.PlaybackDuration = o.pos // last observed position (ticks)
	paused := o.paused
	e.IsPaused = &paused
	e.LastTickAt = &now
	e.WatchedSeconds = (e.WatchedMs + 500) / 1000
	if !o.checkIn.IsZero() {
		ci := o.checkIn
		e.LastCheckIn = &ci
	}
	if !o.activity.IsZero() {
		act := o.activity
		e.LastServerActivity = &act
	}
	if s.PlayState != nil {
		e.PlayMethod = s.PlayState.PlayMethod
		ps := map[string]any{
			"PositionTicks":  s.PlayState.PositionTicks,
			"IsPaused":       s.PlayState.IsPaused,
			"PlayMethod":     s.PlayState.PlayMethod,
			"PlaylistItemId": o.playlist,
		}
		if b, err := json.Marshal(ps); err == nil {
			e.PlayState = datatypes.JSON(b)
		}
	}
	if len(s.NowPlayingItem.MediaStreams) > 0 {
		if b, err := json.Marshal(s.NowPlayingItem.MediaStreams); err == nil {
			e.MediaStreams = datatypes.JSON(b)
		}
	}
	if s.TranscodingInfo != nil {
		if b, err := json.Marshal(s.TranscodingInfo); err == nil {
			e.TranscodingInfo = datatypes.JSON(b)
		}
	}
	e.OriginalContainer = s.NowPlayingItem.Container
	refreshCapture(e, o.capture)
}

// refreshCapture copies the capture fields onto an entry. Audio / subtitle /
// range / locality follow the current poll (the viewer can switch tracks
// mid-play, subtitles can be turned off); transcode details keep the last
// non-empty values since TranscodingInfo disappears when playback pauses.
func refreshCapture(e *models.JFActivityWatchdog, c capture) {
	if c.audioIdx != nil { // keep the last known audio track
		e.AudioStreamIndex, e.AudioLanguage = c.audioIdx, c.audioLang
	}
	e.SubtitleStreamIndex, e.SubtitleLanguage = c.subIdx, c.subLang
	if c.videoRange != nil {
		e.VideoRange = c.videoRange
	}
	if c.remote != nil {
		e.IsRemote = c.remote
	}
	if len(c.reasons) > 0 {
		if b, err := json.Marshal(c.reasons); err == nil {
			e.TranscodeReasons = datatypes.JSON(b)
		}
	}
	// A transcode bitrate always wins; the media bitrate only fills a gap.
	if c.bitrate != nil && *c.bitrate > 0 && (c.bitrateTranscode || e.Bitrate == nil) {
		e.Bitrate = c.bitrate
	}
	if c.hwAccel != nil {
		e.HwAccel = c.hwAccel
	}
}

// closedPlay is a play that ended during this poll.
type closedPlay struct {
	wd           models.JFActivityWatchdog
	tailBudgetMs int64      // max time that may have been played after the last poll
	deleteRow    bool       // the session is gone: remove its watchdog row
	stopActivity *time.Time // LastActivityDate of the still-listed, now idle session (= stop time)
}

// SessionTick polls Jellyfin sessions once: broadcasts them, accumulates real
// playback time and records finished plays. Safe to call concurrently.
func (s *Service) SessionTick(ctx context.Context) {
	s.tickMu.Lock()
	defer s.tickMu.Unlock()

	if err := s.refreshClient(ctx); err != nil {
		return
	}
	sessions, err := s.jf.GetSessions(ctx)
	if err != nil {
		return // Jellyfin unreachable: keep state, never close plays on errors
	}
	s.hub.Emit("sessions", sessions)

	existing, err := s.repos.Watchdog.List(ctx)
	if err != nil {
		return
	}
	prevByID := make(map[string]models.JFActivityWatchdog, len(existing))
	for _, wd := range existing {
		prevByID[wd.Id] = wd
	}

	now := time.Now()
	live := make(map[string]bool, len(sessions))
	listed := make(map[string]jellyfin.SessionInfo, len(sessions))
	for _, ss := range sessions {
		listed[ss.Id] = ss
	}
	var entries []models.JFActivityWatchdog
	var closed []closedPlay

	for _, sess := range sessions {
		o, ok := observe(sess)
		if !ok {
			continue
		}
		live[sess.Id] = true

		prev, had := prevByID[sess.Id]
		var e models.JFActivityWatchdog
		switch {
		case !had:
			e = newEntry(o, now, firstHead(o, ms(PollInterval)), false)
		case prev.ItemId == nil:
			// Row written by an older Jellystics version: close it, start fresh.
			closed = append(closed, closedPlay{wd: prev})
			e = newEntry(o, now, 0, false)
		case samePlay(&prev, o, now):
			e = accumulate(prev, o, now)
		default:
			el := elapsedMs(&prev, now)
			head := minI64(firstHead(o, el), el)
			closed = append(closed, closedPlay{wd: prev, tailBudgetMs: el - head})
			e = newEntry(o, now, head, prev.PeriodicReporter)
		}
		refresh(&e, o, now)
		entries = append(entries, e)
	}

	for _, wd := range existing {
		if !live[wd.Id] {
			c := closedPlay{wd: wd, tailBudgetMs: elapsedMs(&wd, now), deleteRow: true}
			if ss, ok := listed[wd.Id]; ok && !ss.LastActivityDate.IsZero() {
				act := ss.LastActivityDate.Time
				c.stopActivity = &act
			}
			closed = append(closed, c)
		}
	}

	if len(closed) > 0 {
		acts := make([]models.JFPlaybackActivity, 0, len(closed))
		for _, c := range closed {
			if a, ok := s.finalize(ctx, c, now); ok {
				acts = append(acts, a)
			}
		}
		if err := s.repos.Playback.Upsert(ctx, acts); err != nil {
			log.Printf("[tracker] cannot record %d finished play(s), will retry: %v", len(acts), err)
			return // keep the watchdog untouched so nothing is lost
		}
		if len(acts) > 0 {
			log.Printf("[tracker] recorded %d finished play(s)", len(acts))
		}
	}

	if err := s.repos.Watchdog.Upsert(ctx, entries); err != nil {
		log.Printf("[tracker] watchdog upsert failed: %v", err)
	}
	for _, c := range closed {
		if c.deleteRow {
			_ = s.repos.Watchdog.Delete(ctx, c.wd.Id)
		}
	}
}

// tailMs recovers the playback time between the last poll and the actual stop
// from the stop position Jellyfin saved in the user's item data. It returns 0
// when the saved state is not consistent with this play.
func (s *Service) tailMs(ctx context.Context, wd models.JFActivityWatchdog, budgetMs int64) (int64, *int64) {
	// After a long Jellystics outage the budget would be the whole downtime; a
	// stop position saved much later (another session) must not be credited.
	if budgetMs > maxTailMs {
		budgetMs = maxTailMs
	}
	if budgetMs <= 0 || wd.UserId == nil || wd.ItemId == nil || wd.PlaybackDuration == nil {
		return 0, nil
	}
	if wd.IsPaused != nil && *wd.IsPaused {
		return 0, nil
	}
	cctx, cancel := context.WithTimeout(ctx, userDataTimeout)
	defer cancel()
	ud, err := s.jf.GetUserItemData(cctx, *wd.UserId, *wd.ItemId)
	if err != nil {
		return 0, nil
	}
	last := *wd.PlaybackDuration
	var end int64
	switch {
	case ud.PlaybackPositionTicks > 0:
		end = ud.PlaybackPositionTicks
	case ud.Played && wd.RunTimeTicks != nil && *wd.RunTimeTicks > 0:
		end = *wd.RunTimeTicks // played to completion (position reset by Jellyfin)
	default:
		return 0, nil
	}
	if end < last {
		return 0, nil
	}
	t := (end - last) / ticksPerMs
	if t > budgetMs+stepSlackMs {
		return 0, nil // saved position belongs to another playback
	}
	return minI64(t, budgetMs), &end
}

// finalize turns a finished watchdog entry into a playback row.
func (s *Service) finalize(ctx context.Context, c closedPlay, now time.Time) (models.JFPlaybackActivity, bool) {
	wd := c.wd
	tail, endPos := s.tailMs(ctx, wd, c.tailBudgetMs)
	if tail == 0 {
		tail, endPos = tailFromActivity(c)
	}
	watchedMs := wd.WatchedMs + tail
	if wd.ItemId == nil {
		watchedMs = wd.WatchedSeconds * 1000 // legacy row
	}
	// Every observed playback is a click, even with no progress (started then
	// stopped while buffering or paused). Legacy rows keep the old filter.
	if wd.ItemId == nil && watchedMs < 1000 {
		return models.JFPlaybackActivity{}, false
	}
	secs := (watchedMs + 500) / 1000
	paused := (wd.PausedMs + 500) / 1000

	if endPos == nil {
		endPos = wd.PlaybackDuration
	}
	var ended time.Time
	if wd.LastTickAt != nil {
		ended = wd.LastTickAt.Add(time.Duration(tail) * time.Millisecond)
	} else {
		ended = now
	}

	id := wd.Id
	if wd.ActivityId != nil {
		id = *wd.ActivityId
	}
	startedAt := wd.StartedAt
	if startedAt == nil && wd.ActivityDateInserted != nil {
		if t, err := time.Parse("2006-01-02 15:04:05.000-07:00", *wd.ActivityDateInserted); err == nil {
			startedAt = &t
		}
	}
	itemId := wd.ItemId
	if itemId == nil {
		if wd.EpisodeId != nil {
			itemId = wd.EpisodeId
		} else {
			itemId = wd.NowPlayingItemId
		}
	}

	return models.JFPlaybackActivity{
		Id:                   id,
		IsPaused:             wd.IsPaused,
		UserId:               wd.UserId,
		UserName:             wd.UserName,
		Client:               wd.Client,
		DeviceName:           wd.DeviceName,
		DeviceId:             wd.DeviceId,
		ApplicationVersion:   wd.ApplicationVersion,
		NowPlayingItemId:     wd.NowPlayingItemId,
		NowPlayingItemName:   wd.NowPlayingItemName,
		EpisodeId:            wd.EpisodeId,
		SeasonId:             wd.SeasonId,
		SeriesName:           wd.SeriesName,
		PlaybackDuration:     &secs,
		PlayMethod:           wd.PlayMethod,
		ActivityDateInserted: wd.ActivityDateInserted,
		MediaStreams:         wd.MediaStreams,
		TranscodingInfo:      wd.TranscodingInfo,
		PlayState:            wd.PlayState,
		OriginalContainer:    wd.OriginalContainer,
		RemoteEndPoint:       wd.RemoteEndPoint,
		ServerId:             wd.ServerId,
		Source:               "watchdog",
		ItemId:               itemId,
		MediaType:            wd.MediaType,
		RunTimeTicks:         wd.RunTimeTicks,
		PositionTicksStart:   wd.PositionTicksStart,
		PositionTicksEnd:     endPos,
		PausedSeconds:        &paused,
		StartedAt:            startedAt,
		EndedAt:              &ended,
		IsView:               watchedMs >= MinViewSeconds*1000,
		AudioStreamIndex:     wd.AudioStreamIndex,
		AudioLanguage:        wd.AudioLanguage,
		SubtitleStreamIndex:  wd.SubtitleStreamIndex,
		SubtitleLanguage:     wd.SubtitleLanguage,
		TranscodeReasons:     wd.TranscodeReasons,
		Bitrate:              wd.Bitrate,
		HwAccel:              wd.HwAccel,
		VideoRange:           wd.VideoRange,
		PauseCount:           wd.PauseCount,
		IsRemote:             wd.IsRemote,
	}, true
}

// SyncSessions is the manual "SessionSync" task: one tracking poll.
func (s *Service) SyncSessions(ctx context.Context) error {
	s.SessionTick(ctx)
	return nil
}

// tailFromActivity recovers the playback time after the last poll when the
// stop position is not usable (Jellyfin saves position 0 / Played=true for
// items shorter than MinResumeDurationSeconds, 5 min by default — i.e. nearly
// every music track). After a stop the session stays listed and its
// LastActivityDate is the stop time; the gap with the LastActivityDate stored
// at the last poll is measured on the Jellyfin server clock only.
func tailFromActivity(c closedPlay) (int64, *int64) {
	wd := c.wd
	if c.stopActivity == nil || wd.LastServerActivity == nil || (wd.IsPaused != nil && *wd.IsPaused) {
		return 0, nil
	}
	budget := minI64(c.tailBudgetMs, maxTailMs)
	t := ms(c.stopActivity.Sub(*wd.LastServerActivity))
	if t <= 0 || budget <= 0 {
		return 0, nil
	}
	t = minI64(t, budget)
	if wd.PlaybackDuration == nil {
		return t, nil
	}
	end := *wd.PlaybackDuration + t*ticksPerMs
	if wd.RunTimeTicks != nil && *wd.RunTimeTicks > 0 && end > *wd.RunTimeTicks {
		end = *wd.RunTimeTicks
	}
	return t, &end
}
