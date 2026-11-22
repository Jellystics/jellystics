// Package geo resolves the remote IPs of recorded plays to a country and city
// (MaxMind GeoLite web service) and caches the answers in ip_geo_cache.
package geo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	// ErrNotConfigured means no GeoLite credentials are set.
	ErrNotConfigured = errors.New("geolite credentials missing")
	// ErrNotFound means the geolocator has no answer for this IP (cached as a miss).
	ErrNotFound = errors.New("ip not found")
)

const (
	// LookbackDays is how far back plays are scanned for remote IPs.
	LookbackDays = 90
	// MissRetry is how long a failed lookup is remembered before retrying.
	MissRetry = 7 * 24 * time.Hour
	// HitRefresh is how long a successful lookup is trusted.
	HitRefresh = 90 * 24 * time.Hour
)

// Geolocator resolves an IP to an ISO country code and a city name.
type Geolocator interface {
	Locate(ctx context.Context, ip string) (country, city string, err error)
}

// GeoLite calls the MaxMind GeoLite web service.
type GeoLite struct {
	AccountID, LicenseKey string
	Client                *http.Client
	BaseURL               string // default https://geolite.info/geoip/v2.1/city
}

// NewGeoLiteFromEnv reads JS_GEOLITE_ACCOUNT_ID / JS_GEOLITE_LICENSE_KEY.
func NewGeoLiteFromEnv() *GeoLite {
	return &GeoLite{
		AccountID:  os.Getenv("JS_GEOLITE_ACCOUNT_ID"),
		LicenseKey: os.Getenv("JS_GEOLITE_LICENSE_KEY"),
	}
}

func (g *GeoLite) Configured() bool { return g.AccountID != "" && g.LicenseKey != "" }

// Raw performs the lookup and returns the HTTP status and body untouched.
func (g *GeoLite) Raw(ctx context.Context, ip string) (int, []byte, error) {
	if !g.Configured() {
		return 0, nil, ErrNotConfigured
	}
	base := g.BaseURL
	if base == "" {
		base = "https://geolite.info/geoip/v2.1/city"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s", base, ip), nil)
	if err != nil {
		return 0, nil, err
	}
	req.SetBasicAuth(g.AccountID, g.LicenseKey)
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, data, err
}

// Locate implements Geolocator.
func (g *GeoLite) Locate(ctx context.Context, ip string) (string, string, error) {
	status, body, err := g.Raw(ctx, ip)
	if err != nil {
		return "", "", err
	}
	if status == http.StatusNotFound {
		return "", "", ErrNotFound
	}
	if status != http.StatusOK {
		return "", "", fmt.Errorf("geolite status %d", status)
	}
	var r struct {
		Country struct {
			IsoCode string `json:"iso_code"`
		} `json:"country"`
		City struct {
			Names map[string]string `json:"names"`
		} `json:"city"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return "", "", err
	}
	if r.Country.IsoCode == "" {
		return "", "", ErrNotFound
	}
	return r.Country.IsoCode, r.City.Names["en"], nil
}

// Service fills ip_geo_cache.
type Service struct {
	db        *gorm.DB
	loc       Geolocator
	interval  time.Duration // pause between external calls (rate limit)
	maxPerRun int
	now       func() time.Time
}

func New(db *gorm.DB, loc Geolocator) *Service {
	return &Service{db: db, loc: loc, interval: 300 * time.Millisecond, maxPerRun: 500, now: time.Now}
}

// publicIP normalises a RemoteEndPoint ("ip", "ip:port", "[v6]:port",
// IPv4-mapped IPv6) and returns false for non-public addresses.
func publicIP(s string) (string, bool) {
	s = strings.TrimSpace(s)
	var addr netip.Addr
	if ap, err := netip.ParseAddrPort(s); err == nil {
		addr = ap.Addr()
	} else if a, err := netip.ParseAddr(strings.Trim(s, "[]")); err == nil {
		addr = a
	} else {
		return "", false
	}
	addr = addr.Unmap().WithZone("")
	if addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() || addr.IsUnspecified() || addr.IsMulticast() {
		return "", false
	}
	return addr.String(), true
}

// Run geolocates the not-yet-cached remote IPs of the last 90 days.
// Returns the number of IPs looked up.
func (s *Service) Run(ctx context.Context) (int, error) {
	var raw []string
	since := s.now().AddDate(0, 0, -LookbackDays)
	if err := s.db.WithContext(ctx).Raw(
		`SELECT DISTINCT "RemoteEndPoint" FROM jf_playback_activity
		 WHERE "RemoteEndPoint" IS NOT NULL AND "RemoteEndPoint" <> ''
		   AND COALESCE("IsRemote", true) AND "StartedAt" >= ?`, since).Scan(&raw).Error; err != nil {
		return 0, err
	}
	seen := map[string]bool{}
	var ips []string
	for _, r := range raw {
		if ip, ok := publicIP(r); ok && !seen[ip] {
			seen[ip] = true
			ips = append(ips, ip)
		}
	}
	if len(ips) == 0 {
		return 0, nil
	}

	type row struct {
		IP        string
		Country   *string
		UpdatedAt time.Time
	}
	var cached []row
	if err := s.db.WithContext(ctx).Raw(`SELECT ip, country, updated_at FROM ip_geo_cache WHERE ip IN ?`, ips).
		Scan(&cached).Error; err != nil {
		return 0, err
	}
	fresh := map[string]bool{}
	now := s.now()
	for _, c := range cached {
		ttl := MissRetry
		if c.Country != nil {
			ttl = HitRefresh
		}
		if now.Sub(c.UpdatedAt) < ttl {
			fresh[c.IP] = true
		}
	}

	done := 0
	for _, ip := range ips {
		if fresh[ip] {
			continue
		}
		if done >= s.maxPerRun {
			break
		}
		if done > 0 && s.interval > 0 {
			select {
			case <-ctx.Done():
				return done, ctx.Err()
			case <-time.After(s.interval):
			}
		}
		country, city, err := s.loc.Locate(ctx, ip)
		switch {
		case errors.Is(err, ErrNotConfigured):
			return done, nil
		case errors.Is(err, ErrNotFound):
			country, city = "", "" // remembered as a miss
		case err != nil:
			log.Printf("[geo] lookup %s failed: %v", ip, err)
			continue // transient: not cached, retried next run
		}
		done++
		if err := s.store(ctx, ip, country, city, now); err != nil {
			return done, err
		}
	}
	return done, nil
}

func (s *Service) store(ctx context.Context, ip, country, city string, at time.Time) error {
	var c, ci *string
	if country != "" {
		c = &country
	}
	if city != "" {
		ci = &city
	}
	return s.db.WithContext(ctx).Exec(
		`INSERT INTO ip_geo_cache (ip, country, city, updated_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT (ip) DO UPDATE SET country = EXCLUDED.country, city = EXCLUDED.city, updated_at = EXCLUDED.updated_at`,
		ip, c, ci, at).Error
}
