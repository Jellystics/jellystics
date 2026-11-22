package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service"
	tasksvc "github.com/Jellystics/Jellystics/internal/service/task"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// TasksApiHandler.List / Run
// ---------------------------------------------------------------------------

func tasksHandler(t *testing.T) (*handler.TasksApiHandler, *repository.Container, *gin.Engine) {
	t.Helper()
	db := testutil.NewDB(t)
	repos := repository.New(db)
	svcs := &service.Container{Task: tasksvc.New(repos, nil)}
	gin.SetMode(gin.TestMode)
	return handler.NewTasksApiHandler(svcs, repos, db), repos, gin.New()
}

func TestTasksList(t *testing.T) {
	h, _, r := tasksHandler(t)
	r.GET("/tasks", h.List)
	w := getOK(t, r, "/tasks")

	var tasks []struct {
		Name        string  `json:"name"`
		DisplayName string  `json:"displayName"`
		Running     bool    `json:"running"`
		LastResult  *string `json:"lastResult"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Four canonical task entries in fixed order.
	if len(tasks) != 4 {
		t.Fatalf("tasks = %d, want 4", len(tasks))
	}
	if tasks[0].Name != "Full Jellyfin Sync" {
		t.Errorf("tasks[0].name = %q, want Full Jellyfin Sync", tasks[0].Name)
	}
	// Idle service → not running, no prior run recorded.
	if tasks[0].Running {
		t.Error("expected running=false on a fresh handler")
	}
}

func TestTasksList_ReportsLastRun(t *testing.T) {
	h, repos, r := tasksHandler(t)
	r.GET("/tasks", h.List)

	// Insert a completed log entry for the full-sync task; List should surface it.
	name := "Full Jellyfin Sync"
	res := "success"
	tr := "2026-07-12T10:00:00Z"
	if err := repos.Log.Insert(t.Context(), &models.JFLogging{
		Id: "log-1", Name: &name, Result: &res, TimeRun: &tr,
	}); err != nil {
		t.Fatalf("insert log: %v", err)
	}

	w := getOK(t, r, "/tasks")
	var tasks []struct {
		Name       string  `json:"name"`
		LastResult *string `json:"lastResult"`
		LastRun    *string `json:"lastRun"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &tasks)
	var full *struct {
		Name       string  `json:"name"`
		LastResult *string `json:"lastResult"`
		LastRun    *string `json:"lastRun"`
	}
	for i := range tasks {
		if tasks[i].Name == "Full Jellyfin Sync" {
			full = &tasks[i]
		}
	}
	if full == nil {
		t.Fatalf("Full Jellyfin Sync task missing from %+v", tasks)
	}
	if full.LastResult == nil || *full.LastResult != "success" {
		t.Errorf("lastResult = %v, want success", full.LastResult)
	}
}

func TestTasksRun_UnknownTask(t *testing.T) {
	h, _, r := tasksHandler(t)
	r.POST("/run/:name", h.Run)
	w := postJSON(t, r, "/run/DoesNotExist", `{}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for unknown task", w.Code)
	}
	var got struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Error == "" {
		t.Error("expected error message for unknown task")
	}
}

// Note: the Run handler dispatches recognized task names to background
// goroutines that invoke the Sync service (nil in this lightweight harness), so
// only the synchronous unknown-task branch is exercised here to avoid standing
// up the full Jellyfin sync dependency graph.
