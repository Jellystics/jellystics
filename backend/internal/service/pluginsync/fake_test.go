package pluginsync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	gosync "sync"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"gorm.io/gorm"
)

// fakePlugin is an in-memory Jellystics plugin implementing the cursor API
// contract (spec §5): a global version sequence, rows served by Version >
// since, ascending, in pages of at most pageSize.
type fakePlugin struct {
	t   *testing.T
	srv *httptest.Server

	mu            gosync.Mutex
	absent        bool
	schemaVersion int
	pageSize      int
	failOn        map[string]int // path -> fail the n-th request (1-based) with 500
	calls         map[string]int
	sinces        map[string][]int64
	seq           int64
	plays         map[string]jellyfin.JellysticsPlay
	library       []jellyfin.JellysticsLibraryEvent
	userData      []jellyfin.JellysticsUserDataEvent
	logins        []jellyfin.JellysticsLogin
}

func newFakePlugin(t *testing.T) *fakePlugin {
	f := &fakePlugin{t: t, schemaVersion: 1, pageSize: 500, failOn: map[string]int{}, calls: map[string]int{},
		sinces: map[string][]int64{}, plays: map[string]jellyfin.JellysticsPlay{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/Jellystics/Info", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.absent {
			http.NotFound(w, r)
			return
		}
		if f.fail(w, r.URL.Path) {
			return
		}
		writeJSON(w, map[string]any{"pluginVersion": "1.0.0.0", "schemaVersion": f.schemaVersion, "cursor": f.seq,
			"counts": map[string]int{"plays": len(f.plays)}, "serverTimeZone": "UTC"})
	})
	page := func(path string, versions func() []int64, item func(v int64) any) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.absent {
				http.NotFound(w, r)
				return
			}
			since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
			f.sinces[path] = append(f.sinces[path], since)
			if f.fail(w, path) {
				return
			}
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit <= 0 || limit > f.pageSize {
				limit = f.pageSize
			}
			var vs []int64
			for _, v := range versions() {
				if v > since {
					vs = append(vs, v)
				}
			}
			sort.Slice(vs, func(i, j int) bool { return vs[i] < vs[j] })
			hasMore := len(vs) > limit
			if hasMore {
				vs = vs[:limit]
			}
			items := []any{}
			next := since
			for _, v := range vs {
				items = append(items, item(v))
				next = v
			}
			writeJSON(w, map[string]any{"items": items, "next": next, "hasMore": hasMore})
		})
	}
	page("/Jellystics/Plays", func() []int64 {
		var vs []int64
		for _, p := range f.plays {
			vs = append(vs, p.Version)
		}
		return vs
	}, func(v int64) any {
		for _, p := range f.plays {
			if p.Version == v {
				return p
			}
		}
		return nil
	})
	page("/Jellystics/LibraryEvents", func() []int64 {
		var vs []int64
		for _, e := range f.library {
			vs = append(vs, e.Version)
		}
		return vs
	}, func(v int64) any {
		for _, e := range f.library {
			if e.Version == v {
				return e
			}
		}
		return nil
	})
	page("/Jellystics/UserDataEvents", func() []int64 {
		var vs []int64
		for _, e := range f.userData {
			vs = append(vs, e.Version)
		}
		return vs
	}, func(v int64) any {
		for _, e := range f.userData {
			if e.Version == v {
				return e
			}
		}
		return nil
	})
	page("/Jellystics/Logins", func() []int64 {
		var vs []int64
		for _, e := range f.logins {
			vs = append(vs, e.Version)
		}
		return vs
	}, func(v int64) any {
		for _, e := range f.logins {
			if e.Version == v {
				return e
			}
		}
		return nil
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

// fail must be called with f.mu held.
func (f *fakePlugin) fail(w http.ResponseWriter, path string) bool {
	f.calls[path]++
	if n, ok := f.failOn[path]; ok && n == f.calls[path] {
		http.Error(w, "boom", http.StatusInternalServerError)
		return true
	}
	return false
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// putPlay inserts or updates a play, assigning it the next version.
func (f *fakePlugin) putPlay(p jellyfin.JellysticsPlay) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	p.Version = f.seq
	if p.Source == "" {
		p.Source = "plugin"
	}
	if p.State == "" {
		p.State = "closed"
	}
	f.plays[p.Id] = p
}

func (f *fakePlugin) set(fn func(f *fakePlugin)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f)
}

func (f *fakePlugin) nextVersion() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	return f.seq
}

// newTestService wires a Service on a fresh DB whose Jellyfin host is the fake plugin.
func newTestService(t *testing.T, f *fakePlugin) (*Service, *gorm.DB) {
	t.Helper()
	db := testutil.NewDB(t)
	if err := db.Exec(`UPDATE app_config SET "JF_HOST" = ? WHERE "ID" = 1`, f.srv.URL).Error; err != nil {
		t.Fatal(err)
	}
	return New(repository.New(db)), db
}

var t0 = time.Date(2026, 11, 20, 20, 0, 0, 0, time.UTC)

func sptr(s string) *string { return &s }

func moviePlay(id string, start time.Time, watchedMs int64) jellyfin.JellysticsPlay {
	end := jellyfin.FlexTime{Time: start.Add(time.Duration(watchedMs) * time.Millisecond)}
	return jellyfin.JellysticsPlay{
		Id: id, UserId: "user-1", UserName: sptr("alice"), ItemId: "movie-1", ItemName: sptr("Movie"),
		ItemType: sptr("Movie"), DeviceId: sptr("dev-1"), DeviceName: sptr("TV"), Client: sptr("Jellyfin Web"),
		StartedAt: jellyfin.FlexTime{Time: start}, EndedAt: &end, LastEventAt: end, WatchedMs: watchedMs,
	}
}
