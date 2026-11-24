package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jellystics/Jellystics/internal/middleware"
	"github.com/gin-gonic/gin"
)

const meSecret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func meRouter(requireUser bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	known := map[string]bool{"5f2a8c0e1b3d4f6a7c9e0b1d2f3a4c5e": true}
	mw := middleware.PluginMe(
		func(context.Context) (string, error) { return meSecret, nil },
		func(_ context.Context, id string) (bool, error) { return known[id], nil },
		requireUser)
	r.GET("/api/me/x", mw, func(c *gin.Context) { c.String(http.StatusOK, middleware.MeUserID(c)) })
	return r
}

func meReq(r *gin.Engine, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/me/x", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPluginMe(t *testing.T) {
	r := meRouter(true)
	user := "5F2A8C0E-1B3D-4F6A-7C9E-0B1D2F3A4C5E" // dashed / upper case: normalised
	cases := []struct {
		name    string
		headers map[string]string
		code    int
		body    string
	}{
		{"no secret", map[string]string{"X-Jellystics-User": user}, 401, ""},
		{"wrong secret", map[string]string{"X-Jellystics-Secret": meSecret[:63] + "0", "X-Jellystics-User": user}, 401, ""},
		{"session token instead of secret", map[string]string{"Authorization": "Bearer " + meSecret, "X-Jellystics-User": user}, 401, ""},
		{"no user", map[string]string{"X-Jellystics-Secret": meSecret}, 403, ""},
		{"unknown user", map[string]string{"X-Jellystics-Secret": meSecret, "X-Jellystics-User": "00000000000000000000000000000000"}, 403, ""},
		{"valid", map[string]string{"X-Jellystics-Secret": meSecret, "X-Jellystics-User": user}, 200, "5f2a8c0e1b3d4f6a7c9e0b1d2f3a4c5e"},
	}
	for _, tc := range cases {
		w := meReq(r, tc.headers)
		if w.Code != tc.code {
			t.Errorf("%s: status %d, want %d (%s)", tc.name, w.Code, tc.code, w.Body.String())
			continue
		}
		if tc.code != 200 && w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
			t.Errorf("%s: content-type %q", tc.name, w.Header().Get("Content-Type"))
		}
		if tc.body != "" && w.Body.String() != tc.body {
			t.Errorf("%s: user %q, want %q", tc.name, w.Body.String(), tc.body)
		}
	}
}

func TestPluginMeSecretOnly(t *testing.T) {
	r := meRouter(false)
	if w := meReq(r, nil); w.Code != 401 {
		t.Fatalf("no secret: %d", w.Code)
	}
	if w := meReq(r, map[string]string{"X-Jellystics-Secret": meSecret}); w.Code != 200 {
		t.Fatalf("secret only: %d", w.Code)
	}
}

func TestPluginMeEmptyStoredSecretRejects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/me/x", middleware.PluginMe(
		func(context.Context) (string, error) { return "", nil },
		func(context.Context, string) (bool, error) { return true, nil }, false),
		func(c *gin.Context) { c.Status(200) })
	if w := meReq(r, map[string]string{"X-Jellystics-Secret": ""}); w.Code != 401 {
		t.Fatalf("empty secret accepted: %d", w.Code)
	}
}

func TestPluginMeUserLookupFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/me/x", middleware.PluginMe(
		func(context.Context) (string, error) { return meSecret, nil },
		func(context.Context, string) (bool, error) { return false, context.DeadlineExceeded }, true),
		func(c *gin.Context) { c.Status(200) })
	w := meReq(r, map[string]string{"X-Jellystics-Secret": meSecret, "X-Jellystics-User": "u1"})
	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "internal error") {
		t.Fatalf("lookup failure: %d %s", w.Code, w.Body.String())
	}
}

func TestPluginMeSecretLookupFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/me/x", middleware.PluginMe(
		func(context.Context) (string, error) { return "", context.DeadlineExceeded },
		func(context.Context, string) (bool, error) { return true, nil }, true),
		func(c *gin.Context) { c.Status(200) })
	if w := meReq(r, map[string]string{"X-Jellystics-Secret": meSecret, "X-Jellystics-User": "u1"}); w.Code != http.StatusInternalServerError {
		t.Fatalf("secret failure: %d", w.Code)
	}
}
