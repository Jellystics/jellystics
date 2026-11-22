package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
)

func do(t *testing.T, r *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	return w
}

// getOK GETs target and fails the test unless the status is 200.
func getOK(t *testing.T, r *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()
	w := do(t, r, target)
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d, body %s", target, w.Code, w.Body.String())
	}
	return w
}

// TestUserHandler verifies List and Get (including 404).
func TestUserHandler(t *testing.T) {
	db := testutil.NewDB(t)
	repos := repository.New(db)
	if err := repos.User.Upsert(t.Context(), []models.JFUser{
		{Id: "u1", Name: sp("Alice")},
		{Id: "u2", Name: sp("Bob")},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	gin.SetMode(gin.TestMode)
	h := handler.NewUserHandler(repos)
	r := gin.New()
	r.GET("/users", h.List)
	r.GET("/users/:id", h.Get)

	w := do(t, r, "/users")
	var users []models.JFUser
	_ = json.Unmarshal(w.Body.Bytes(), &users)
	if len(users) != 2 {
		t.Errorf("users = %d, want 2", len(users))
	}

	one := do(t, r, "/users/u1")
	var u models.JFUser
	_ = json.Unmarshal(one.Body.Bytes(), &u)
	if u.Name == nil || *u.Name != "Alice" {
		t.Errorf("u1 = %+v, want Alice", u)
	}

	if nf := do(t, r, "/users/ghost"); nf.Code != http.StatusNotFound {
		t.Errorf("unknown user status = %d, want 404", nf.Code)
	}
}
