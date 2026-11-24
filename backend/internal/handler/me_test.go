package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/router"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// meEnv is a seeded database (seedV2: u1 Alice on Web/Chrome, u2 Bob on
// Android/Pixel) with IPs on every play, and the /api/me routes.
type meEnv struct {
	db     *gorm.DB
	r      *gin.Engine
	secret string
}

func newMeEnv(t *testing.T) *meEnv {
	t.Helper()
	db := v2Seeded(t)
	db.Exec(`UPDATE jf_playback_activity SET "RemoteEndPoint" = CASE "UserId" WHEN 'u1' THEN '198.51.100.11' ELSE '203.0.113.22' END`)
	repos := repository.New(db)
	secret, err := repos.MeLink.Secret(context.Background())
	if err != nil || len(secret) != 64 {
		t.Fatalf("secret %q: %v", secret, err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	router.RegisterMe(r, repos, db)
	return &meEnv{db: db, r: r, secret: secret}
}

func (e *meEnv) get(t *testing.T, target, user string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("X-Jellystics-Secret", e.secret)
	if user != "" {
		req.Header.Set("X-Jellystics-User", user)
	}
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

// Bob's fingerprints: none may appear in Alice's responses.
var bobMarkers = []string{"Bob", "u2", "Pixel", "Android", "t1", "t2", "Blue Album", "203.0.113.22"}

// Never on /api/me, whoever asks.
var ipMarkers = []string{"198.51.100.11", "203.0.113.22", "RemoteEndPoint", "remoteEndPoint", "activeUsers", "topUsers", "\"user\""}

func TestMeEndpointsIsolation(t *testing.T) {
	e := newMeEnv(t)
	cases := []struct {
		path string
		mine []string // Alice's data that must be present
	}{
		{"/api/me/profile", []string{`"id":"u1"`, `"name":"Alice"`, "Chrome", "Sci-Fi"}},
		{"/api/me/overview?compare=1", []string{`"watchSeconds":340`, `"views":4`, `"playedToEnd"`}},
		{"/api/me/top", []string{"m1", "s1", "Inception"}},
		{"/api/me/history", []string{`"total":4`, "Inception", "Chrome"}},
		{"/api/me/types", []string{`"movie":{"watchSeconds":100`, `"series":{"watchSeconds":240`, `"music":{"watchSeconds":0`}},
		{"/api/me/habits", []string{`"weekHour"`, `"hours"`, `"weekdays"`}},
		{"/api/me/devices", []string{"Chrome", "Web", `"clients"`}},
		{"/api/me/scatter", []string{`"total":4`, "Inception"}},
	}
	for _, tc := range cases {
		for _, forged := range []string{"userId=u2", "userId=u2&users=u2&user=u2&UserId=u2&id=u2"} {
			sep := "?"
			if strings.Contains(tc.path, "?") {
				sep = "&"
			}
			target := tc.path + sep + forged
			w := e.get(t, target, "u1")
			if w.Code != http.StatusOK {
				t.Errorf("%s: status %d %s", target, w.Code, w.Body.String())
				continue
			}
			body := w.Body.String()
			if !json.Valid(w.Body.Bytes()) {
				t.Errorf("%s: invalid JSON", target)
			}
			for _, m := range append(append([]string{}, bobMarkers...), ipMarkers...) {
				if strings.Contains(body, m) {
					t.Errorf("%s: leaked %q in %s", target, m, body)
				}
			}
			for _, m := range tc.mine {
				if !strings.Contains(body, m) {
					t.Errorf("%s: missing own data %q in %s", target, m, body)
				}
			}
		}
	}

	// Bob sees his own data, not Alice's.
	w := e.get(t, "/api/me/history?userId=u1", "u2")
	if b := w.Body.String(); w.Code != 200 || strings.Contains(b, "Chrome") || !strings.Contains(b, "Pixel") || strings.Contains(b, "Alice") {
		t.Errorf("bob history: %d %s", w.Code, b)
	}
}

func TestMeEndpointsAuth(t *testing.T) {
	e := newMeEnv(t)
	for _, p := range []string{"profile", "overview", "top", "history", "types", "habits", "devices", "scatter"} {
		target := "/api/me/" + p
		// Unknown / missing user.
		if w := e.get(t, target, "ffffffffffffffffffffffffffffffff"); w.Code != http.StatusForbidden {
			t.Errorf("%s unknown user: %d", target, w.Code)
		}
		if w := e.get(t, target, ""); w.Code != http.StatusForbidden {
			t.Errorf("%s no user: %d", target, w.Code)
		}
		// No / wrong secret, or a Jellystics session token instead.
		for _, hdr := range []map[string]string{
			{"X-Jellystics-User": "u1"},
			{"X-Jellystics-User": "u1", "X-Jellystics-Secret": "nope"},
			{"X-Jellystics-User": "u1", "Authorization": "Bearer " + e.secret},
		} {
			req := httptest.NewRequest(http.MethodGet, target, nil)
			for k, v := range hdr {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			e.r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s %v: %d", target, hdr, w.Code)
			}
		}
	}
}

func TestMeHealth(t *testing.T) {
	e := newMeEnv(t)
	w := e.get(t, "/api/me/health", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"ok":true`) || strings.Contains(w.Body.String(), e.secret) {
		t.Fatalf("health: %d %s", w.Code, w.Body.String())
	}
	req := httptest.NewRequest(http.MethodGet, "/api/me/health", nil)
	rec := httptest.NewRecorder()
	e.r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("health without secret: %d", rec.Code)
	}
}

func TestMeHistoryPagination(t *testing.T) {
	e := newMeEnv(t)
	var got struct {
		Rows           []map[string]any
		Total          int
		Page, PageSize int
	}
	w := e.get(t, "/api/me/history?page=2&pageSize=3&userId=u2", "u1")
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Total != 4 || got.Page != 2 || got.PageSize != 3 || len(got.Rows) != 1 {
		t.Fatalf("page 2: %+v", got)
	}
}

// Same figures as the admin user page (R5).
func TestMeMatchesAdminUserPage(t *testing.T) {
	e := newMeEnv(t)
	var admin, me struct {
		Totals      struct{ WatchSeconds, Views, Clicks int }
		PlayedToEnd struct{ Plays, Finished, Measured int }
	}
	v2JSON(t, e.db, "/api/v2/users/u1", &admin)
	if err := json.Unmarshal(e.get(t, "/api/me/overview", "u1").Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	if admin != me {
		t.Fatalf("admin %+v != me %+v", admin, me)
	}
}

func TestMeStatic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	get := func(fsys fstest.MapFS, target string) *httptest.ResponseRecorder {
		r := gin.New()
		r.GET("/me/*path", handler.MeStatic(fsys))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
		return w
	}
	built := fstest.MapFS{
		"me.html":           {Data: []byte("<html>me</html>")},
		"index.html":        {Data: []byte("<html>admin</html>")},
		"assets/me-abc.js":  {Data: []byte("console.log(1)")},
		"assets/me-abc.css": {Data: []byte("body{}")},
	}
	if w := get(built, "/me/"); w.Code != 200 || w.Body.String() != "<html>me</html>" || w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("/me/: %d %q", w.Code, w.Body.String())
	}
	if w := get(built, "/me/history"); w.Code != 200 || w.Body.String() != "<html>me</html>" {
		t.Errorf("client route: %d %q", w.Code, w.Body.String())
	}
	if w := get(built, "/me/assets/me-abc.js"); w.Code != 200 || w.Body.String() != "console.log(1)" ||
		!strings.Contains(w.Header().Get("Content-Type"), "javascript") || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("asset: %d %q %v", w.Code, w.Body.String(), w.Header())
	}
	if w := get(built, "/me/assets/missing.js"); w.Code != 404 {
		t.Errorf("missing asset: %d", w.Code)
	}
	if w := get(built, "/me"); w.Code != http.StatusMovedPermanently && w.Code != 200 {
		t.Errorf("/me: %d", w.Code)
	}
	// Not built yet: 404, no panic.
	if w := get(fstest.MapFS{"index.html": {Data: []byte("x")}}, "/me/"); w.Code != 404 {
		t.Errorf("no me.html: %d", w.Code)
	}
}
