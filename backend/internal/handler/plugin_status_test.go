package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service/pluginsync"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestPluginStatus(t *testing.T) {
	db := testutil.NewDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/plugin/status", handler.PluginStatus(pluginsync.New(repository.New(db))))

	db.Exec(`UPDATE plugin_sync_state SET detected = true, plugin_version = '1.0.0.0', schema_version = 1,
		cursor_plays = 42, last_success_at = now()`)
	w := do(t, r, "/api/plugin/status")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["detected"] != true || body["healthy"] != true || body["pluginVersion"] != "1.0.0.0" || body["cursorPlays"] != float64(42) ||
		body["playsSynced"] != float64(0) || body["supportedSchemaVersion"] != float64(1) {
		t.Fatalf("body = %v", body)
	}
	if _, leaked := body["PushCursor"]; leaked {
		t.Fatalf("push columns must not be serialised: %v", body)
	}
}

func TestPluginPushAndResyncEndpoints(t *testing.T) {
	db := testutil.NewDB(t)
	gin.SetMode(gin.TestMode)
	svc := pluginsync.New(repository.New(db))
	r := gin.New()
	r.POST("/api/plugin/push-history", handler.PluginPushHistory(svc))
	r.GET("/api/plugin/push-history/status", handler.PluginPushStatus(svc))
	r.POST("/api/plugin/resync", handler.PluginResync(svc))
	post := func(target string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, target, nil))
		return w
	}

	// No Jellyfin configured: the push is refused.
	db.Exec(`UPDATE app_config SET "JF_HOST" = NULL`)
	if w := post("/api/plugin/push-history"); w.Code != http.StatusConflict {
		t.Fatalf("push = %d %s", w.Code, w.Body.String())
	}
	w := do(t, r, "/api/plugin/push-history/status")
	var st map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &st)
	if w.Code != http.StatusOK || st["state"] != "idle" || st["sent"] != float64(0) || st["resumable"] != false {
		t.Fatalf("push status = %d %v", w.Code, st)
	}
	if w := post("/api/plugin/resync"); w.Code != http.StatusAccepted {
		t.Fatalf("resync = %d %s", w.Code, w.Body.String())
	}
	svc.WaitResync()
}
