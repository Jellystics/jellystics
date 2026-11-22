// Package pluginsync imports data recorded by the Jellystics Jellyfin plugin
// (plays, library / user-data / login events) through its cursor API.
//
// The plugin is optional: without it (or when it is incompatible or
// unreachable) a run is a cheap no-op and the Jellystics tracker keeps
// recording plays. While the Plays table syncs successfully (state.Healthy),
// the tracker stops writing plays and the plugin's plays are authoritative.
package pluginsync

import (
	"context"
	"errors"
	"fmt"
	"log"
	gosync "sync"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
)

const (
	// TaskName is the scheduler background job name (settings key to disable it).
	TaskName = "Jellystics Plugin Sync"
	// Interval is how often the sync runs.
	Interval = 30 * time.Second
	// CronSpec is Interval as a scheduler spec.
	CronSpec = "@every 30s"

	// SupportedSchemaVersion is the plugin API schema this Jellystics speaks.
	SupportedSchemaVersion = jellyfin.JellysticsSchemaVersion

	pageLimit = 500
	// maxPagesPerTable bounds one run (500k rows); the rest follows next run.
	maxPagesPerTable = 1000
	// viewsRefreshEvery throttles the materialized statistics views refresh
	// after new plays (the live stats queries read jf_playback_activity
	// directly; only the legacy views lag, by at most this much).
	viewsRefreshEvery = 5 * time.Minute
	// recentDedupeWindow: every successful run re-checks tracker rows against
	// the plugin plays started within this window (tracker / sync race).
	recentDedupeWindow = 24 * time.Hour
	// staleOpenAfter: open plugin plays are closed once the plugin has been
	// unhealthy and their last checkpoint is older than this.
	staleOpenAfter = time.Hour
)

// errCursorBackwards: the plugin served a cursor behind ours (reinstall /
// lost database not reported by /Info): Jellystics is blind, so the plugin
// must not be considered healthy.
var errCursorBackwards = errors.New("plugin cursor went backwards")

// Service runs the plugin sync. Safe for concurrent use (runs are serialised).
type Service struct {
	repos *repository.Container
	now   func() time.Time
	// OnDataChanged is called after plays were written or tracker rows
	// removed (main wires the stats cache reset).
	OnDataChanged func()

	mu             gosync.Mutex
	lastRefresh    time.Time
	lastStaleCheck time.Time
	lastLoggedErr  string
}

func New(repos *repository.Container) *Service {
	return &Service{repos: repos, now: time.Now}
}

// Status is the read model of the sync state (settings card).
type Status struct {
	models.PluginSyncState
	Healthy bool `json:"healthy"`
}

// Status returns the persisted sync state and whether the plugin is
// currently the source of truth for plays.
func (s *Service) Status(ctx context.Context) (*Status, error) {
	st, err := s.repos.PluginSync.GetState(ctx)
	if err != nil {
		return nil, err
	}
	return &Status{PluginSyncState: *st, Healthy: st.Healthy(s.now())}, nil
}

// Result summarises one run.
type Result struct {
	Detected    bool
	Compatible  bool
	CursorReset bool  // the plugin cursor was behind ours: everything re-read
	Plays       int   // plays written
	Imported    int   // of which imports
	Deduped     int64 // tracker rows replaced by plugin plays
	Library     int
	UserData    int
	Logins      int
}

func (s *Service) client(ctx context.Context) (*jellyfin.Client, error) {
	cfg, err := s.repos.Config.Get(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.JFHost == nil || *cfg.JFHost == "" {
		return nil, nil
	}
	key := ""
	if cfg.JFApiKey != nil {
		key = *cfg.JFApiKey
	}
	return jellyfin.NewClient(*cfg.JFHost, key), nil
}

// Tick is the scheduler entry point: Run, errors only logged (on change).
func (s *Service) Tick(ctx context.Context) {
	_, _ = s.Run(ctx)
}

// Run performs one sync: detection, schema check, then every table from its
// cursor until hasMore = false.
func (s *Service) Run(ctx context.Context) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var res Result
	jf, err := s.client(ctx)
	if err != nil || jf == nil {
		return res, err // Jellyfin not configured yet: nothing to do
	}
	st, err := s.repos.PluginSync.GetState(ctx)
	if err != nil {
		return res, err
	}
	now := s.now()
	st.LastCheckedAt = &now

	info, err := jf.GetJellysticsInfo(ctx)
	if err != nil {
		s.closeStale(ctx, st, now)
		return res, s.fail(ctx, st, fmt.Errorf("plugin info: %w", err))
	}
	if info == nil {
		if st.Detected {
			log.Printf("[plugin-sync] Jellystics plugin no longer detected: the tracker records plays")
		}
		st.Detected, st.LastError = false, nil
		s.lastLoggedErr = ""
		s.closeStale(ctx, st, now)
		return res, s.repos.PluginSync.SaveState(ctx, st)
	}
	res.Detected = true
	if !st.Detected {
		log.Printf("[plugin-sync] Jellystics plugin %s detected (schema %d)", info.PluginVersion, info.SchemaVersion)
	}
	st.Detected = true
	pv, sv := info.PluginVersion, info.SchemaVersion
	st.PluginVersion, st.SchemaVersion = &pv, &sv
	if sv != SupportedSchemaVersion {
		s.closeStale(ctx, st, now)
		return res, s.fail(ctx, st, fmt.Errorf("incompatible plugin schema version %d (supported: %d): plugin ignored, update Jellystics or the plugin", sv, SupportedSchemaVersion))
	}
	res.Compatible = true

	// Plugin database reset (reinstall, lost DB): its cursor is behind ours.
	// Re-read everything; upserts are idempotent and dedupe runs again.
	if maxCur := max(st.CursorPlays, st.CursorLibraryEvents, st.CursorUserDataEvents, st.CursorLogins); info.Cursor < maxCur {
		log.Printf("[plugin-sync] plugin cursor %d is behind ours (%d): plugin data was reset, resyncing from 0", info.Cursor, maxCur)
		st.CursorPlays, st.CursorLibraryEvents, st.CursorUserDataEvents, st.CursorLogins = 0, 0, 0, 0
		res.CursorReset = true
	}

	// Plays decide health.
	var pluginIds []string
	written := repository.UpsertResult{}
	err = s.syncTable(ctx, jf, st, jellyfin.JellysticsPlays, &st.CursorPlays, func(p *jellyfin.JellysticsPage) error {
		plays, err := p.Plays()
		if err != nil {
			return err
		}
		acts, err := s.mapPlays(ctx, plays)
		if err != nil {
			return err
		}
		for _, a := range acts {
			if a.Source == repository.SourceJellysticsPlugin {
				pluginIds = append(pluginIds, a.Id)
			}
		}
		r, err := s.repos.PluginSync.UpsertPlays(ctx, acts)
		written.Plugin += r.Plugin
		written.Imported += r.Imported
		return err
	})
	res.Plays, res.Imported = written.Plugin+written.Imported, written.Imported
	s.afterPlays(ctx, pluginIds, written, err == nil, &res)
	if err != nil {
		if errors.Is(err, errCursorBackwards) {
			st.LastSuccessAt = nil // blind: let the tracker record plays now
		}
		return res, s.fail(ctx, st, err)
	}
	done := s.now()
	st.LastSuccessAt, st.LastError = &done, nil

	// Annex tables: failures are reported but do not affect health.
	annex := []struct {
		table  jellyfin.JellysticsTable
		cursor *int64
		apply  func(*jellyfin.JellysticsPage) error
	}{
		{jellyfin.JellysticsLibraryEvents, &st.CursorLibraryEvents, func(p *jellyfin.JellysticsPage) error {
			rows, err := p.LibraryEvents()
			if err != nil {
				return err
			}
			res.Library += len(rows)
			return s.repos.PluginSync.UpsertLibraryEvents(ctx, mapLibraryEvents(rows))
		}},
		{jellyfin.JellysticsUserDataEvents, &st.CursorUserDataEvents, func(p *jellyfin.JellysticsPage) error {
			rows, err := p.UserDataEvents()
			if err != nil {
				return err
			}
			res.UserData += len(rows)
			return s.repos.PluginSync.UpsertUserDataEvents(ctx, mapUserDataEvents(rows))
		}},
		{jellyfin.JellysticsLogins, &st.CursorLogins, func(p *jellyfin.JellysticsPage) error {
			rows, err := p.Logins()
			if err != nil {
				return err
			}
			res.Logins += len(rows)
			return s.repos.PluginSync.UpsertLogins(ctx, mapLogins(rows))
		}},
	}
	var annexErr error
	for _, a := range annex {
		if err := s.syncTable(ctx, jf, st, a.table, a.cursor, a.apply); err != nil {
			annexErr = errors.Join(annexErr, err)
		}
	}
	if annexErr != nil {
		return res, s.fail(ctx, st, annexErr)
	}
	if s.lastLoggedErr != "" {
		log.Printf("[plugin-sync] sync recovered")
		s.lastLoggedErr = ""
	}
	return res, s.repos.PluginSync.SaveState(ctx, st)
}

// mapPlays maps a page of plays, completing the source video stream from the
// catalog when the plugin did not send it.
func (s *Service) mapPlays(ctx context.Context, plays []jellyfin.JellysticsPlay) ([]models.JFPlaybackActivity, error) {
	var need []string
	for _, p := range plays {
		if p.SourceWidth == nil || p.VideoRangeType == nil {
			need = append(need, normID(p.ItemId))
		}
	}
	fallback, err := s.repos.PluginSync.ItemSourceVideo(ctx, need)
	if err != nil {
		return nil, err
	}
	acts := make([]models.JFPlaybackActivity, 0, len(plays))
	for _, p := range plays {
		var fb *repository.SourceVideo
		if v, ok := fallback[normID(p.ItemId)]; ok {
			fb = &v
		}
		if a, ok := MapPlayWith(p, fb); ok {
			acts = append(acts, a)
		}
	}
	return acts, nil
}

// syncTable paginates one table from *cursor, saving the cursor after every page.
func (s *Service) syncTable(ctx context.Context, jf *jellyfin.Client, st *models.PluginSyncState,
	table jellyfin.JellysticsTable, cursor *int64, apply func(*jellyfin.JellysticsPage) error) error {
	for i := 0; i < maxPagesPerTable; i++ {
		page, err := jf.GetJellysticsPage(ctx, table, *cursor, pageLimit)
		if err != nil {
			return fmt.Errorf("%s: %w", table, err)
		}
		if page.Next < *cursor {
			return fmt.Errorf("%s: %w (since %d, next %d)", table, errCursorBackwards, *cursor, page.Next)
		}
		if err := apply(page); err != nil {
			return fmt.Errorf("%s: store page since %d: %w", table, *cursor, err)
		}
		if page.Next > *cursor {
			*cursor = page.Next
			if err := s.repos.PluginSync.SaveState(ctx, st); err != nil {
				return err
			}
		} else if page.HasMore {
			return fmt.Errorf("%s: cursor did not advance (since %d, next %d)", table, *cursor, page.Next)
		}
		if !page.HasMore {
			return nil
		}
	}
	return nil
}

// afterPlays replaces tracker rows covered by plugin plays (the new ones,
// and — after a successful sync — every plugin play of the last day), repairs
// imported durations, refreshes the views (throttled) and resets the stats
// caches. No BackfillTracking: plugin rows carry ItemId / StartedAt / EndedAt.
func (s *Service) afterPlays(ctx context.Context, pluginIds []string, written repository.UpsertResult, synced bool, res *Result) {
	if n, err := s.repos.PluginSync.DedupeTrackerRows(ctx, pluginIds); err != nil {
		log.Printf("[plugin-sync] dedupe tracker rows: %v", err)
	} else {
		res.Deduped += n
	}
	if synced {
		if n, err := s.repos.PluginSync.DedupeRecentTrackerRows(ctx, s.now().Add(-recentDedupeWindow)); err != nil {
			log.Printf("[plugin-sync] dedupe recent tracker rows: %v", err)
		} else {
			res.Deduped += n
		}
	}
	if written.Imported > 0 {
		if err := s.repos.Playback.SanitizeImported(ctx); err != nil {
			log.Printf("[plugin-sync] sanitize imported plays: %v", err)
		}
	}
	if written.Plugin+written.Imported == 0 && res.Deduped == 0 {
		return
	}
	if s.OnDataChanged != nil {
		s.OnDataChanged()
	}
	if written.Plugin+written.Imported > 0 {
		if now := s.now(); now.Sub(s.lastRefresh) >= viewsRefreshEvery {
			s.lastRefresh = now
			if err := s.repos.Stats.RefreshViews(ctx); err != nil {
				log.Printf("[plugin-sync] view refresh: %v", err)
			}
		}
	}
}

// closeStale closes open plugin plays left behind by a plugin that has been
// unhealthy for more than staleOpenAfter (checked at most every 10 minutes).
func (s *Service) closeStale(ctx context.Context, st *models.PluginSyncState, now time.Time) {
	if st.LastSuccessAt == nil || now.Sub(*st.LastSuccessAt) <= staleOpenAfter || now.Sub(s.lastStaleCheck) < 10*time.Minute {
		return
	}
	s.lastStaleCheck = now
	n, err := s.repos.PluginSync.CloseStalePluginPlays(ctx, now.Add(-staleOpenAfter))
	if err != nil {
		log.Printf("[plugin-sync] close stale open plays: %v", err)
	} else if n > 0 {
		log.Printf("[plugin-sync] closed %d stale open plugin play(s)", n)
	}
}

// fail records err in the state (cursors already saved stay) and logs it once.
func (s *Service) fail(ctx context.Context, st *models.PluginSyncState, err error) error {
	msg := err.Error()
	st.LastError = &msg
	if msg != s.lastLoggedErr {
		log.Printf("[plugin-sync] %s", msg)
		s.lastLoggedErr = msg
	}
	if serr := s.repos.PluginSync.SaveState(ctx, st); serr != nil {
		return errors.Join(err, serr)
	}
	return err
}
