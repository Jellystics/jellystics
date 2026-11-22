package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Jellystics/Jellystics/internal/config"
	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// revalidateEvery is how often a session's Jellyfin token is re-checked
// against Jellyfin (GET /Users/Me). A token revoked in Jellyfin (logout,
// device removed in the dashboard, user disabled or demoted) therefore ends
// the Jellystics session within this delay.
const revalidateEvery = 5 * time.Minute

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotAdmin           = errors.New("admin access required")
	ErrNotConfigured      = errors.New("jellyfin host not configured")
	ErrSessionExpired     = errors.New("session expired")
)

type Service struct {
	repos *repository.Container
	jf    *jellyfin.Client
	cfg   *config.Config
	now   func() time.Time
}

func New(repos *repository.Container, jf *jellyfin.Client, cfg *config.Config) *Service {
	return &Service{repos: repos, jf: jf, cfg: cfg, now: time.Now}
}

// Claims are carried by the Jellystics JWT. SessionId points to an
// auth_sessions row holding the Jellyfin access token; the JWT itself has no
// expiry — the session lives exactly as long as the Jellyfin token.
type Claims struct {
	UserId    string `json:"userId"`
	UserName  string `json:"userName"`
	IsAdmin   bool   `json:"isAdmin"`
	SessionId string `json:"sid,omitempty"`
	jwt.RegisteredClaims
}

func (s *Service) jfClient(ctx context.Context) (*jellyfin.Client, error) {
	cfg, err := s.repos.Config.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if cfg.JFHost == nil || *cfg.JFHost == "" {
		return nil, ErrNotConfigured
	}
	return jellyfin.NewClient(*cfg.JFHost, ""), nil
}

// Login authenticates against Jellyfin on behalf of a browser (deviceId is a
// random id generated and persisted by that browser), stores the Jellyfin
// access token in a server-side session and returns a signed JWT.
func (s *Service) Login(ctx context.Context, username, password, deviceId, deviceName string) (string, error) {
	jf, err := s.jfClient(ctx)
	if err != nil {
		return "", err
	}
	if deviceId == "" {
		deviceId = uuid.NewString()
	}
	dev := jellyfin.DeviceInfo{DeviceId: "jellystics-" + deviceId, DeviceName: deviceName}
	if dev.DeviceName == "" {
		dev.DeviceName = "Jellystics"
	}

	res, err := jf.AuthenticateUserOnDevice(ctx, username, password, dev)
	if err != nil {
		if errors.Is(err, jellyfin.ErrUnauthorized) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("jellyfin auth: %w", err)
	}
	if !res.User.Policy.IsAdministrator {
		_ = jf.Logout(ctx, res.AccessToken, dev)
		return "", ErrNotAdmin
	}

	now := s.now()
	sess := &models.AuthSession{
		Id:              uuid.NewString(),
		UserId:          res.User.Id,
		UserName:        res.User.Name,
		JFToken:         res.AccessToken,
		DeviceId:        dev.DeviceId,
		DeviceName:      dev.DeviceName,
		CreatedAt:       now,
		LastValidatedAt: now,
		LastSeenAt:      now,
	}
	if err := s.repos.AuthSession.Create(ctx, sess); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	claims := Claims{
		UserId:           res.User.Id,
		UserName:         res.User.Name,
		IsAdmin:          true,
		SessionId:        sess.Id,
		RegisteredClaims: jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(now)},
	}
	return s.sign(claims)
}

func (s *Service) sign(claims Claims) (string, error) {
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// Logout revokes the Jellyfin token and deletes the session.
func (s *Service) Logout(ctx context.Context, claims *Claims) error {
	if claims == nil || claims.SessionId == "" || s.repos == nil || s.repos.AuthSession == nil {
		return nil
	}
	sess, err := s.repos.AuthSession.Get(ctx, claims.SessionId)
	if err != nil {
		return nil // already gone
	}
	if jf, err := s.jfClient(ctx); err == nil {
		dev := jellyfin.DeviceInfo{DeviceId: sess.DeviceId, DeviceName: sess.DeviceName}
		if err := jf.Logout(ctx, sess.JFToken, dev); err != nil {
			log.Printf("[auth] jellyfin logout failed (session removed anyway): %v", err)
		}
	}
	return s.repos.AuthSession.Delete(ctx, sess.Id)
}

// ValidateToken parses and verifies a JWT signature, returning its claims.
// It does not check the backing session; use Authenticate for requests.
func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// Authenticate validates a JWT and its backing session. The session's
// Jellyfin token is re-checked against Jellyfin every revalidateEvery; if
// Jellyfin revoked it (or the user is no longer admin) the session is deleted.
// If Jellyfin is merely unreachable the session is kept, so a Jellyfin restart
// does not log everybody out.
func (s *Service) Authenticate(ctx context.Context, tokenStr string) (*Claims, error) {
	claims, err := s.ValidateToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if s.repos == nil || s.repos.AuthSession == nil {
		return claims, nil // no session store wired (unit tests)
	}
	if claims.SessionId == "" {
		return nil, ErrSessionExpired // legacy token from before Jellyfin-backed sessions
	}
	sess, err := s.repos.AuthSession.Get(ctx, claims.SessionId)
	if err != nil {
		return nil, ErrSessionExpired
	}

	if s.now().Sub(sess.LastValidatedAt) < revalidateEvery {
		if s.now().Sub(sess.LastSeenAt) > time.Minute {
			_ = s.repos.AuthSession.Touch(ctx, sess.Id, false)
		}
		return claims, nil
	}

	jf, err := s.jfClient(ctx)
	if err != nil {
		return claims, nil
	}
	dev := jellyfin.DeviceInfo{DeviceId: sess.DeviceId, DeviceName: sess.DeviceName}
	me, err := jf.GetMe(ctx, sess.JFToken, dev)
	switch {
	case errors.Is(err, jellyfin.ErrUnauthorized):
		_ = s.repos.AuthSession.Delete(ctx, sess.Id)
		return nil, ErrSessionExpired
	case err != nil:
		log.Printf("[auth] cannot reach Jellyfin to revalidate session: %v", err)
		return claims, nil
	case !me.Policy.IsAdministrator || me.Id != sess.UserId:
		_ = s.repos.AuthSession.Delete(ctx, sess.Id)
		return nil, ErrNotAdmin
	}
	_ = s.repos.AuthSession.Touch(ctx, sess.Id, true)
	claims.UserName = me.Name
	return claims, nil
}

// GetConfig returns public auth configuration.
func (s *Service) GetConfig(ctx context.Context) (map[string]any, error) {
	cfg, err := s.repos.Config.Get(ctx)
	if err != nil {
		return nil, err
	}
	hasHost := cfg.JFHost != nil && *cfg.JFHost != ""
	return map[string]any{
		"requireLogin": cfg.RequireLogin,
		"hasHost":      hasHost,
	}, nil
}

// IsConfigured reports whether the initial setup (Jellyfin host) is done.
func (s *Service) IsConfigured(ctx context.Context) bool {
	if s.repos == nil {
		return true
	}
	cfg, err := s.repos.Config.Get(ctx)
	return err == nil && cfg.JFHost != nil && *cfg.JFHost != ""
}
