// Package pluginsync imports data recorded by the Jellystics Jellyfin plugin
// (plays, library / user-data / login events) through its cursor API.
//
// The plugin is optional: without it (or when it is incompatible or
// unreachable) a run is a cheap no-op and the Jellystics tracker keeps
// recording plays. While the plugin syncs successfully (state.Healthy), the
// tracker stops writing plays and the plugin's plays are authoritative.
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
	SupportedSchemaVersion = 1

	pageLimit = 500
	// maxPagesPerTable bounds one run (500k rows); the rest follows next run.
	maxPagesPerTable = 1000
	// viewsRefreshEvery throttles the statistics views refresh after new plays.
	viewsRefreshEvery = 5 * time.Minute
)

// Service runs the plugin sync. Safe for concurrent use (runs are serialised).
type Service struct {
	repos *repository.Container
	now   func() time.Time

	mu            gosync.Mutex
	lastRefresh   time.Time
	lastLoggedErr string
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

// Detected reports whether the Jellystics plugin answered the last check.
func (s *Service) Detected(ctx context.Context) bool {
	st, err := s.repos.PluginSync.GetState(ctx)
	return err == nil && st.Detected
}

// Result summarises one run.
type Result struct {
	Detected   bool
	Compatible bool
	Plays      int // plays written
	Deduped    int64
	Library    int
	UserData   int
	Logins     int
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
		return res, s.fail(ctx, st, fmt.Errorf("plugin info: %w", err))
	}
	if info == nil {
		if st.Detected {
			log.Printf("[plugin-sync] Jellystics plugin no longer detected: the tracker records plays")
		}
		st.Detected, st.LastError = false, nil
		s.lastLoggedErr = ""
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
		return res, s.fail(ctx, st, fmt.Errorf("incompatible plugin schema version %d (supported: %d): plugin ignored, update Jellystics or the plugin", sv, SupportedSchemaVersion))
	}
	res.Compatible = true

	var pluginIds []string
	newPlays := 0
	steps := []struct {
		table  jellyfin.JellysticsTable
		cursor *int64
		apply  func(*jellyfin.JellysticsPage) error
	}{
		{jellyfin.JellysticsPlays, &st.CursorPlays, func(p *jellyfin.JellysticsPage) error {
			plays, err := p.Plays()
			if err != nil {
				return err
			}
			acts := make([]models.JFPlaybackActivity, 0, len(plays))
			for _, pl := range plays {
				if a, ok := MapPlay(pl); ok {
					acts = append(acts, a)
					if a.Source == repository.SourcePlugin {
						pluginIds = append(pluginIds, a.Id)
					}
				}
			}
			n, err := s.repos.PluginSync.UpsertPlays(ctx, acts)
			res.Plays += n
			newPlays += n
			return err
		}},
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

	for _, step := range steps {
		if err := s.syncTable(ctx, jf, st, step.table, step.cursor, step.apply); err != nil {
			s.afterPlays(ctx, pluginIds, newPlays, &res)
			return res, s.fail(ctx, st, err)
		}
	}
	s.afterPlays(ctx, pluginIds, newPlays, &res)

	done := s.now()
	st.LastSuccessAt, st.LastError = &done, nil
	if s.lastLoggedErr != "" {
		log.Printf("[plugin-sync] sync recovered")
		s.lastLoggedErr = ""
	}
	return res, s.repos.PluginSync.SaveState(ctx, st)
}

// syncTable paginates one table from *cursor, saving the cursor after every page.
func (s *Service) syncTable(ctx context.Context, jf *jellyfin.Client, st *models.PluginSyncState,
	table jellyfin.JellysticsTable, cursor *int64, apply func(*jellyfin.JellysticsPage) error) error {
	for i := 0; i < maxPagesPerTable; i++ {
		page, err := jf.GetJellysticsPage(ctx, table, *cursor, pageLimit)
		if err != nil {
			return fmt.Errorf("%s: %w", table, err)
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

// afterPlays replaces tracker rows covered by the new plugin plays, then
// runs the same post-processing as the Playback Reporting merge.
func (s *Service) afterPlays(ctx context.Context, pluginIds []string, newPlays int, res *Result) {
	if len(pluginIds) > 0 {
		n, err := s.repos.PluginSync.DedupeTrackerRows(ctx, pluginIds)
		if err != nil {
			log.Printf("[plugin-sync] dedupe tracker rows: %v", err)
		}
		res.Deduped += n
	}
	if newPlays == 0 {
		return
	}
	if err := s.repos.Playback.BackfillTracking(ctx); err != nil {
		log.Printf("[plugin-sync] backfill tracking: %v", err)
	}
	if now := s.now(); now.Sub(s.lastRefresh) >= viewsRefreshEvery {
		s.lastRefresh = now
		if err := s.repos.Stats.RefreshViews(ctx); err != nil {
			log.Printf("[plugin-sync] view refresh: %v", err)
		}
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
