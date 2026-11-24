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

// « Mes statistiques » link: on every successful sync, Jellystics gives the
// plugin its URL (as seen from Jellyfin) and the /api/me shared secret
// (POST /Jellystics/Link), when the plugin is not linked or either changed
// since the last successful push. A link failure never fails the sync.

// URL sources reported by the status.
const (
	MeURLSourceEnv  = "env"  // JELLYSTICS_PUBLIC_URL
	MeURLSourceAuto = "auto" // local address of the connection to Jellyfin
)

// meURLDetectTimeout bounds the auto-detection dial.
const meURLDetectTimeout = 3 * time.Second

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
	if u := strings.TrimRight(strings.TrimSpace(env), "/"); u != "" {
		return MeURL{URL: u, Source: MeURLSourceEnv}, nil
	}
	u, err := AutoMeURL(ctx, jf, port)
	return MeURL{URL: u, Source: MeURLSourceAuto}, err
}

func secretHash(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}

func (s *Service) resolveMeURL(ctx context.Context, jf *jellyfin.Client) (MeURL, error) {
	return ResolveMeURL(ctx, s.PublicURL, jf, s.Port)
}

// link pushes the url / secret to the plugin when needed and records the
// outcome in st (saved by the caller). Errors are logged once, never returned.
func (s *Service) link(ctx context.Context, jf *jellyfin.Client, st *models.PluginSyncState, info *jellyfin.JellysticsInfo) {
	if info.MeLinked == nil { // plugin without the feature
		st.MeLinked, st.MeReachable = false, nil
		return
	}
	st.MeLinked, st.MeReachable = *info.MeLinked, info.MeReachable
	if s.meUnsupported != "" && s.meUnsupported == info.PluginVersion {
		return
	}
	s.meUnsupported = ""

	failed := func(err error) {
		msg := err.Error()
		st.MeLinkError = &msg
		if msg != s.lastLinkErr {
			log.Printf("[plugin-sync] « Mes statistiques » link: %s", msg)
			s.lastLinkErr = msg
		}
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
	if *info.MeLinked && st.MeLinkedURL != nil && *st.MeLinkedURL == u.URL &&
		st.MeLinkedSecretHash != nil && *st.MeLinkedSecretHash == hash {
		return // up to date
	}
	res, err := jf.PostJellysticsLink(ctx, u.URL, secret)
	if errors.Is(err, jellyfin.ErrJellysticsLinkUnsupported) {
		// Older plugin: silently unsupported until its version changes.
		s.meUnsupported = info.PluginVersion
		st.MeLinked, st.MeReachable, st.MeLinkError = false, nil, nil
		return
	}
	if err != nil {
		failed(err)
		return
	}
	now := s.now()
	url := u.URL
	st.MeLinked, st.MeReachable = true, res.Reachable
	st.MeLinkedURL, st.MeLinkedSecretHash, st.MeLinkedAt, st.MeLinkError = &url, &hash, &now, nil
	s.lastLinkErr = ""
	log.Printf("[plugin-sync] « Mes statistiques » linked (%s, %s)", url, u.Source)
}
