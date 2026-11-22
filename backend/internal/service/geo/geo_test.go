package geo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/testutil"
	"gorm.io/gorm"
)

type fakeGeo struct {
	calls map[string]int
	res   map[string]error
}

func (f *fakeGeo) Locate(_ context.Context, ip string) (string, string, error) {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	f.calls[ip]++
	if err := f.res[ip]; err != nil {
		return "", "", err
	}
	return "FR", "Paris", nil
}

func seedPlay(t *testing.T, db *gorm.DB, id, endpoint string, remote *bool, startedAgoDays int) {
	t.Helper()
	err := db.Exec(`INSERT INTO jf_playback_activity ("Id","RemoteEndPoint","IsRemote","StartedAt")
		VALUES (?,?,?,?)`, id, endpoint, remote, time.Now().AddDate(0, 0, -startedAgoDays)).Error
	if err != nil {
		t.Fatal(err)
	}
}

func newSvc(t *testing.T, g Geolocator) (*Service, *gorm.DB) {
	db := testutil.NewDB(t)
	s := New(db, g)
	s.interval = 0
	return s, db
}

func TestRun_FillsCacheForPublicRecentIPsOnly(t *testing.T) {
	f := &fakeGeo{}
	s, db := newSvc(t, f)
	yes := true
	seedPlay(t, db, "a", "203.0.113.9:443", &yes, 1)
	seedPlay(t, db, "b", "203.0.113.9", &yes, 2)    // duplicate IP
	seedPlay(t, db, "c", "192.168.1.5", &yes, 1)    // private
	seedPlay(t, db, "d", "198.51.100.7", &yes, 120) // too old
	seedPlay(t, db, "e", "::ffff:10.0.0.1", nil, 1) // mapped private
	seedPlay(t, db, "f", "2001:db8::1", nil, 1)     // unknown locality, public v6

	n, err := s.Run(context.Background())
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if f.calls["203.0.113.9"] != 1 || f.calls["2001:db8::1"] != 1 || len(f.calls) != 2 {
		t.Fatalf("calls = %v", f.calls)
	}
	var country string
	db.Raw(`SELECT country FROM ip_geo_cache WHERE ip = '203.0.113.9'`).Scan(&country)
	if country != "FR" {
		t.Fatalf("country = %q", country)
	}
}

func TestRun_CachedHitsAreNotRefetchedAndMissesRetryAfter7Days(t *testing.T) {
	f := &fakeGeo{res: map[string]error{"198.51.100.7": ErrNotFound}}
	s, db := newSvc(t, f)
	yes := true
	seedPlay(t, db, "a", "203.0.113.9", &yes, 1)
	seedPlay(t, db, "b", "198.51.100.7", &yes, 1)
	if n, _ := s.Run(context.Background()); n != 2 {
		t.Fatalf("first run n=%d", n)
	}
	if n, _ := s.Run(context.Background()); n != 0 {
		t.Fatalf("second run must use cache, n=%d", n)
	}
	// 8 days later: the miss is retried, the hit is not.
	s.now = func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }
	if n, _ := s.Run(context.Background()); n != 1 {
		t.Fatalf("after 8 days n=%d", n)
	}
	if f.calls["198.51.100.7"] != 2 || f.calls["203.0.113.9"] != 1 {
		t.Fatalf("calls = %v", f.calls)
	}
	var country *string
	db.Raw(`SELECT country FROM ip_geo_cache WHERE ip = '198.51.100.7'`).Scan(&country)
	if country != nil {
		t.Fatalf("miss must be cached with NULL country, got %v", *country)
	}
}

func TestRun_TransientErrorIsNotCached(t *testing.T) {
	f := &fakeGeo{res: map[string]error{"203.0.113.9": errors.New("boom")}}
	s, db := newSvc(t, f)
	yes := true
	seedPlay(t, db, "a", "203.0.113.9", &yes, 1)
	if n, err := s.Run(context.Background()); n != 0 || err != nil {
		t.Fatalf("n=%d err=%v", n, err)
	}
	var cnt int
	db.Raw(`SELECT count(*) FROM ip_geo_cache`).Scan(&cnt)
	if cnt != 0 {
		t.Fatalf("cache rows = %d", cnt)
	}
}

func TestRun_NotConfiguredStopsQuietly(t *testing.T) {
	f := &fakeGeo{res: map[string]error{"203.0.113.9": ErrNotConfigured}}
	s, db := newSvc(t, f)
	yes := true
	seedPlay(t, db, "a", "203.0.113.9", &yes, 1)
	if _, err := s.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRun_MaxPerRun(t *testing.T) {
	f := &fakeGeo{}
	s, db := newSvc(t, f)
	s.maxPerRun = 1
	yes := true
	seedPlay(t, db, "a", "203.0.113.9", &yes, 1)
	seedPlay(t, db, "b", "203.0.113.10", &yes, 1)
	if n, _ := s.Run(context.Background()); n != 1 {
		t.Fatalf("n=%d", n)
	}
}

func TestGeoLite_Locate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "acc" || p != "key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/9.9.9.9" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"country":{"iso_code":"DE"},"city":{"names":{"en":"Berlin"}}}`))
	}))
	defer srv.Close()
	g := &GeoLite{AccountID: "acc", LicenseKey: "key", BaseURL: srv.URL}
	c, city, err := g.Locate(context.Background(), "1.1.1.1")
	if err != nil || c != "DE" || city != "Berlin" {
		t.Fatalf("%q %q %v", c, city, err)
	}
	if _, _, err := g.Locate(context.Background(), "9.9.9.9"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if _, _, err := (&GeoLite{}).Locate(context.Background(), "1.1.1.1"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v", err)
	}
}
