package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/config"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
)

// fakeJellyfin simulates the auth-related Jellyfin endpoints.
type fakeJellyfin struct {
	mu       sync.Mutex
	tokens   map[string]bool // valid tokens
	admin    bool
	down     bool
	lastAuth string // Authorization header of the last login
}

func (f *fakeJellyfin) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.down {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		token := ""
		if i := strings.Index(r.Header.Get("Authorization"), `Token="`); i >= 0 {
			rest := r.Header.Get("Authorization")[i+7:]
			token = rest[:strings.Index(rest, `"`)]
		}
		switch r.URL.Path {
		case "/Users/AuthenticateByName":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["Pw"] != "good" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			f.lastAuth = r.Header.Get("Authorization")
			tok := "tok-" + body["Username"]
			f.tokens[tok] = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"AccessToken": tok,
				"User":        map[string]any{"Id": "u1", "Name": body["Username"], "Policy": map[string]any{"IsAdministrator": f.admin}},
			})
		case "/Users/Me":
			if !f.tokens[token] {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"Id": "u1", "Name": "alice", "Policy": map[string]any{"IsAdministrator": f.admin}})
		case "/Sessions/Logout":
			delete(f.tokens, token)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func setup(t *testing.T) (*Service, *fakeJellyfin, *time.Time) {
	t.Helper()
	db := testutil.NewDB(t)
	repos := repository.New(db)
	fj := &fakeJellyfin{tokens: map[string]bool{}, admin: true}
	srv := httptest.NewServer(fj.handler())
	t.Cleanup(srv.Close)

	ctx := context.Background()
	cfg, err := repos.Config.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	host := srv.URL
	cfg.JFHost = &host
	if err := repos.Config.Save(ctx, cfg); err != nil {
		t.Fatal(err)
	}

	clock := time.Now()
	svc := New(repos, nil, &config.Config{JWTSecret: "s"})
	svc.now = func() time.Time { return clock }
	return svc, fj, &clock
}

func TestLogin_UsesPerBrowserDeviceId(t *testing.T) {
	svc, fj, _ := setup(t)
	if _, err := svc.Login(context.Background(), "alice", "good", "abc123", "Jellystics (Firefox)"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fj.lastAuth, `DeviceId="jellystics-abc123"`) {
		t.Errorf("login header %q does not carry the browser device id", fj.lastAuth)
	}
}

func TestLogin_RejectsBadPasswordAndNonAdmin(t *testing.T) {
	svc, fj, _ := setup(t)
	if _, err := svc.Login(context.Background(), "alice", "bad", "d", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("bad password: err = %v", err)
	}
	fj.admin = false
	if _, err := svc.Login(context.Background(), "bob", "good", "d", ""); !errors.Is(err, ErrNotAdmin) {
		t.Errorf("non-admin: err = %v", err)
	}
	if fj.tokens["tok-bob"] {
		t.Error("non-admin login must revoke the Jellyfin token it obtained")
	}
}

// The session must outlive the old 24 h JWT and end only when Jellyfin revokes the token.
func TestAuthenticate_FollowsJellyfinTokenLifetime(t *testing.T) {
	svc, fj, clock := setup(t)
	ctx := context.Background()
	tok, err := svc.Login(ctx, "alice", "good", "d1", "")
	if err != nil {
		t.Fatal(err)
	}

	*clock = clock.Add(30 * 24 * time.Hour) // a month later, token still valid in Jellyfin
	if _, err := svc.Authenticate(ctx, tok); err != nil {
		t.Fatalf("session should still be valid after 30 days: %v", err)
	}

	fj.down = true // Jellyfin restarting: keep the session
	*clock = clock.Add(10 * time.Minute)
	if _, err := svc.Authenticate(ctx, tok); err != nil {
		t.Fatalf("Jellyfin unreachable must not log out: %v", err)
	}
	fj.down = false

	delete(fj.tokens, "tok-alice") // device removed in Jellyfin dashboard
	*clock = clock.Add(10 * time.Minute)
	if _, err := svc.Authenticate(ctx, tok); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("revoked token: err = %v, want ErrSessionExpired", err)
	}
}

func TestAuthenticate_DemotedUserLosesAccess(t *testing.T) {
	svc, fj, clock := setup(t)
	ctx := context.Background()
	tok, _ := svc.Login(ctx, "alice", "good", "d1", "")
	fj.admin = false
	*clock = clock.Add(revalidateEvery + time.Second)
	if _, err := svc.Authenticate(ctx, tok); err == nil {
		t.Fatal("demoted user must lose access")
	}
}

func TestLogout_RevokesJellyfinToken(t *testing.T) {
	svc, fj, _ := setup(t)
	ctx := context.Background()
	tok, _ := svc.Login(ctx, "alice", "good", "d1", "")
	claims, err := svc.Authenticate(ctx, tok)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Logout(ctx, claims); err != nil {
		t.Fatal(err)
	}
	if fj.tokens["tok-alice"] {
		t.Error("Jellyfin token not revoked on logout")
	}
	if _, err := svc.Authenticate(ctx, tok); err == nil {
		t.Error("token still accepted after logout")
	}
}

func TestAuthenticate_RejectsLegacyTokenWithoutSession(t *testing.T) {
	svc, _, _ := setup(t)
	legacy, _ := svc.sign(Claims{UserId: "u1", IsAdmin: true})
	if _, err := svc.Authenticate(context.Background(), legacy); err == nil {
		t.Error("legacy token without session id must be rejected")
	}
}

func TestPruneIdleSessions_RevokesAbandonedTokens(t *testing.T) {
	svc, fj, clock := setup(t)
	ctx := context.Background()
	tok, _ := svc.Login(ctx, "alice", "good", "d1", "")
	*clock = clock.Add(SessionMaxIdle + time.Hour)
	n, err := svc.PruneIdleSessions(ctx)
	if err != nil || n != 1 {
		t.Fatalf("pruned %d, err %v", n, err)
	}
	if fj.tokens["tok-alice"] {
		t.Error("Jellyfin token of a pruned session must be revoked")
	}
	if _, err := svc.Authenticate(ctx, tok); err == nil {
		t.Error("pruned session still accepted")
	}
}
