package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jellystics/Jellystics/internal/config"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service"
	authsvc "github.com/Jellystics/Jellystics/internal/service/auth"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/Jellystics/Jellystics/internal/ws"
	"github.com/gin-gonic/gin"
)

// fullRouter builds the real router with a configured Jellyfin host pointing
// at upstream (which records every request it receives).
func fullRouter(t *testing.T, configured bool) (*gin.Engine, *[]string) {
	t.Helper()
	db := testutil.NewDB(t)
	repos := repository.New(db)
	var hits []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.Path)
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Set-Cookie", "leak=1")
		w.Write([]byte("img"))
	}))
	t.Cleanup(upstream.Close)
	if configured {
		cfg, _ := repos.Config.Get(context.Background())
		host, key := upstream.URL, "ADMINKEY"
		cfg.JFHost, cfg.JFApiKey = &host, &key
		_ = repos.Config.Save(context.Background(), cfg)
	}
	appCfg := &config.Config{JWTSecret: "s"}
	svcs := &service.Container{Auth: authsvc.New(repos, nil, appCfg)}
	gin.SetMode(gin.TestMode)
	return New(svcs, repos, ws.NewHub(), db, appCfg, nil), &hits
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestConfigSetup_OpenOnlyBeforeSetup(t *testing.T) {
	r, _ := fullRouter(t, false)
	if w := do(r, "POST", "/auth/configSetup", `{"JF_HOST":"http://jf"}`); w.Code != http.StatusOK {
		t.Fatalf("initial setup: status %d, want 200", w.Code)
	}
	if w := do(r, "POST", "/auth/configSetup", `{"JF_HOST":"http://evil"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("re-pointing host without auth: status %d, want 401", w.Code)
	}
}

func TestProxy_OnlyWhitelistedImages(t *testing.T) {
	r, hits := fullRouter(t, true)
	for _, p := range []string{
		"/proxy/Auth/Keys",
		"/proxy/System/Configuration",
		"/proxy/Users",
		"/proxy/Items/Images/Primary/?id=../../Auth/Keys",
		"/proxy/getSessions",
		"/proxy/getAdminUsers",
	} {
		if w := do(r, "GET", p, ""); w.Code == http.StatusOK {
			t.Errorf("%s: status 200, must be refused", p)
		}
	}
	if len(*hits) != 0 {
		t.Fatalf("upstream Jellyfin was called for refused paths: %v", *hits)
	}

	w := do(r, "GET", "/proxy/Items/Images/Primary/?id=0123456789abcdef0123456789abcdef&fillWidth=200&api_key=x", "")
	if w.Code != http.StatusOK || w.Body.String() != "img" {
		t.Fatalf("image proxy: status %d body %q", w.Code, w.Body.String())
	}
	if (*hits)[0] != "/Items/0123456789abcdef0123456789abcdef/Images/Primary" {
		t.Errorf("upstream path = %s", (*hits)[0])
	}
	if w.Header().Get("Set-Cookie") != "" {
		t.Error("upstream Set-Cookie must not be forwarded")
	}
}

func TestBackupDownload_RequiresAuth(t *testing.T) {
	r, _ := fullRouter(t, true)
	if w := do(r, "GET", "/backup/files/backup_2026-11-22%2003-00-00.json", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("backup download without auth: status %d, want 401", w.Code)
	}
}

func TestValidateSettings_RequiresAuthAfterSetup(t *testing.T) {
	r, _ := fullRouter(t, true)
	if w := do(r, "POST", "/proxy/validateSettings", `{"url":"http://x","apikey":"k"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("validateSettings without auth: status %d, want 401", w.Code)
	}
}
