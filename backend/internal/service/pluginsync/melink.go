package pluginsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
)

// « Mes statistiques » link: after every successful sync, Jellystics gives
// the plugin its URL (as seen from Jellyfin) and the /api/me shared secret
// (POST /Jellystics/Link) when the plugin is not linked or holds another
// url / secret. A link failure never fails the sync; failed pushes back off.

// URL sources reported by the status.
const (
	MeURLSourceEnv  = "env"  // JELLYSTICS_PUBLIC_URL
	MeURLSourceAuto = "auto" // local address of the connection to Jellyfin
)

// Plugin meError values.
const (
	MeErrorUnreachable    = "unreachable"
	MeErrorSecretMismatch = "secretMismatch"
)

const (
	// meURLDetectTimeout bounds the auto-detection dial.
	meURLDetectTimeout = 3 * time.Second
	// meURLCacheFor: the auto-detected URL is re-detected at most this often.
	meURLCacheFor = 5 * time.Minute
	// linkTimeout bounds one POST /Jellystics/Link.
	linkTimeout = 15 * time.Second
	// linkBackoffMin / linkBackoffMax: delay before retrying a failed push,
	// doubling per consecutive failure.
	linkBackoffMin = 30 * time.Second
	linkBackoffMax = 30 * time.Minute
	// mismatchRelinkEvery rate-limits re-pushes triggered only by the plugin
	// reporting secretMismatch (e.g. two Jellystics fighting over a plugin).
	mismatchRelinkEvery = 5 * time.Minute
)

// MeURL is the resolved Jellystics base URL pushed to the plugin.
type MeURL struct {
	URL    string
	Source string
}

// AutoMeURL builds http://<ip>:<port> where ip is the local address of a TCP
// connection Jellystics opens to Jellyfin (port defaults to 3000).
func AutoMeURL(ctx context.Context, jf *jellyfin.Client, port string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, meURLDetectTimeout)
	defer cancel()
	ip, err := jf.LocalAddr(ctx)
	if err != nil {
		return "", fmt.Errorf("detect Jellystics address: %w", err)
	}
	return autoURL(ip, port), nil
}

func autoURL(ip net.IP, port string) string {
	if port = strings.TrimSpace(port); port == "" {
		port = "3000"
	}
	return "http://" + net.JoinHostPort(ip.String(), port)
}

// ResolveMeURL returns the env override (trailing slashes trimmed) when
// set, else the auto-detected URL.
func ResolveMeURL(ctx context.Context, env string, jf *jellyfin.Client, port string) (MeURL, error) {
	if u := envURL(env); u != "" {
		return MeURL{URL: u, Source: MeURLSourceEnv}, nil
	}
	u, err := AutoMeURL(ctx, jf, port)
	return MeURL{URL: u, Source: MeURLSourceAuto}, err
}

func envURL(env string) string { return strings.TrimRight(strings.TrimSpace(env), "/") }

func secretHash(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}

// SecretFingerprint is the fingerprint the plugin reports for a secret: the
// first 16 lowercase hex chars of sha256 over its UTF-8 bytes.
func SecretFingerprint(secret string) string { return secretHash(secret)[:16] }

// currentMeURL is the URL last resolved by the sync loop (never dials).
func (s *Service) currentMeURL() MeURL {
	if u := envURL(s.PublicURL); u != "" {
		return MeURL{URL: u, Source: MeURLSourceEnv}
	}
	s.meMu.Lock()
	defer s.meMu.Unlock()
	return MeURL{URL: s.meURL, Source: MeURLSourceAuto}
}

// resolveMeURL returns the env URL, or the auto-detected one (cached for
// meURLCacheFor; only the sync loop calls it).
func (s *Service) resolveMeURL(ctx context.Context, jf *jellyfin.Client) (MeURL, error) {
	if u := envURL(s.PublicURL); u != "" {
		return MeURL{URL: u, Source: MeURLSourceEnv}, nil
	}
	now := s.now()
	s.meMu.Lock()
	cached, at := s.meURL, s.meURLAt
	s.meMu.Unlock()
	if cached != "" && now.Sub(at) < meURLCacheFor {
		return MeURL{URL: cached, Source: MeURLSourceAuto}, nil
	}
	u, err := AutoMeURL(ctx, jf, s.Port)
	if err != nil {
		return MeURL{Source: MeURLSourceAuto}, err
	}
	s.meMu.Lock()
	s.meURL, s.meURLAt = u, now
	s.meMu.Unlock()
	return MeURL{URL: u, Source: MeURLSourceAuto}, nil
}

// resetLink clears what the plugin reported (plugin absent / incompatible /
// without the feature).
func resetLink(st *models.PluginSyncState) {
	st.MeLinked, st.MeReachable, st.MeError = false, nil, nil
}

// needLink reports whether the plugin must receive url / secret again. The
// plugin's own report (fingerprint, linked url) wins; older plugins that do
// not report them are compared with the last successful push.
func needLink(info *jellyfin.JellysticsInfo, st *models.PluginSyncState, url, hash string, now time.Time) bool {
	if !*info.MeLinked {
		return true
	}
	if fp := info.MeSecretFingerprint; fp != nil {
		if !strings.EqualFold(strings.TrimSpace(*fp), hash[:16]) {
			return true
		}
	} else if st.MeLinkedSecretHash == nil || *st.MeLinkedSecretHash != hash {
		return true
	}
	if lu := info.MeLinkedURL; lu != nil {
		if strings.TrimRight(strings.TrimSpace(*lu), "/") != url {
			return true
		}
	} else if st.MeLinkedURL == nil || *st.MeLinkedURL != url {
		return true
	}
	if info.MeError != nil && *info.MeError == MeErrorSecretMismatch {
		return st.MeLinkedAt == nil || now.Sub(*st.MeLinkedAt) >= mismatchRelinkEvery
	}
	return false
}

// linkBackoff is the delay after n consecutive failures (n >= 1).
func linkBackoff(n int) time.Duration {
	d := linkBackoffMin
	for i := 1; i < n && d < linkBackoffMax; i++ {
		d *= 2
	}
	return min(d, linkBackoffMax)
}

// link pushes the url / secret to the plugin when needed and records the
// outcome in st (saved by the caller). Errors are logged once, never returned.
func (s *Service) link(ctx context.Context, jf *jellyfin.Client, st *models.PluginSyncState, info *jellyfin.JellysticsInfo) {
	if info.MeLinked == nil { // plugin without the feature
		resetLink(st)
		st.MeLinkError = nil
		return
	}
	st.MeLinked, st.MeReachable, st.MeError = *info.MeLinked, info.MeReachable, info.MeError
	if s.meUnsupported != "" && s.meUnsupported == info.PluginVersion {
		return
	}
	s.meUnsupported = ""
	now := s.now()
	if s.linkFailures > 0 && now.Before(s.linkRetryAt) {
		return // backing off after a failure
	}

	failed := func(err error) {
		msg := err.Error()
		st.MeLinkError = &msg
		s.linkFailures++
		s.linkRetryAt = now.Add(linkBackoff(s.linkFailures))
		if msg != s.lastLinkErr {
			log.Printf("[plugin-sync] « Mes statistiques » link: %s (retry in %s)", msg, linkBackoff(s.linkFailures))
			s.lastLinkErr = msg
		}
	}
	succeeded := func() {
		st.MeLinkError = nil
		s.linkFailures, s.lastLinkErr = 0, ""
	}
	u, err := s.resolveMeURL(ctx, jf)
	if err != nil {
		failed(err)
		return
	}
	secret, err := s.repos.MeLink.Secret(ctx)
	if err != nil {
		failed(fmt.Errorf("secret: %w", err))
		return
	}
	hash := secretHash(secret)
	if !needLink(info, st, u.URL, hash, now) {
		succeeded() // up to date
		return
	}
	lctx, cancel := context.WithTimeout(ctx, linkTimeout)
	res, err := jf.PostJellysticsLink(lctx, u.URL, secret)
	cancel()
	if errors.Is(err, jellyfin.ErrJellysticsLinkUnsupported) {
		// Older plugin: silently unsupported until its version changes.
		s.meUnsupported = info.PluginVersion
		resetLink(st)
		succeeded()
		return
	}
	if err != nil {
		failed(err)
		return
	}
	url := u.URL
	st.MeLinked, st.MeReachable, st.MeError = true, res.Reachable, nil
	st.MeLinkedURL, st.MeLinkedSecretHash, st.MeLinkedAt = &url, &hash, &now
	succeeded()
	log.Printf("[plugin-sync] « Mes statistiques » linked (%s, %s)", url, u.Source)
}
