package handler_test

import (
	"encoding/json"
	"net/http"
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
	if body["detected"] != true || body["healthy"] != true || body["pluginVersion"] != "1.0.0.0" || body["cursorPlays"] != float64(42) {
		t.Fatalf("body = %v", body)
	}
}
