package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/config"
	"github.com/Jellystics/Jellystics/internal/middleware"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service/auth"
	"github.com/Jellystics/Jellystics/internal/statsql"
	"github.com/gin-gonic/gin"
)

type FrontendAuthHandler struct {
	repos *repository.Container
	cfg   *config.Config
	auth  *auth.Service
}

func NewFrontendAuthHandler(repos *repository.Container, cfg *config.Config, authSvc *auth.Service) *FrontendAuthHandler {
	return &FrontendAuthHandler{repos: repos, cfg: cfg, auth: authSvc}
}

// GET /auth/isConfigured
func (h *FrontendAuthHandler) IsConfigured(c *gin.Context) {
	tz, _ := statsql.Timezone()
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
		c.JSON(http.StatusOK, gin.H{"state": 0, "timezone": tz})
		return
	}
	c.JSON(http.StatusOK, gin.H{"state": 1, "jellyfinUrl": *cfg.JFHost, "timezone": tz})
}

// POST /auth/configSetup — protected by middleware.SetupOrAdmin.
func (h *FrontendAuthHandler) ConfigSetup(c *gin.Context) {
	var body struct {
		JFHost   string `json:"JF_HOST"`
		JFApiKey string `json:"JF_API_KEY"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errorMessage": err.Error()})
		return
	}
	if body.JFHost == "" {
		c.JSON(http.StatusBadRequest, gin.H{"errorMessage": "JF_HOST is required"})
		return
	}

	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errorMessage": "Failed to load config"})
		return
	}

	host := strings.TrimRight(body.JFHost, "/")
	cfg.JFHost = &host
	if body.JFApiKey != "" {
		cfg.JFApiKey = &body.JFApiKey
	}

	if err := h.repos.Config.Save(c.Request.Context(), cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"errorMessage": "Failed to save config"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /auth/check-server
// Before setup it pings the URL given in the body; once configured it only
// ever pings the configured Jellyfin host (no arbitrary outbound requests).
func (h *FrontendAuthHandler) CheckServer(c *gin.Context) {
	var body struct {
		URL string `json:"url"`
	}
	_ = c.ShouldBindJSON(&body)

	target := body.URL
	if cfg, err := h.repos.Config.Get(c.Request.Context()); err == nil && cfg.JFHost != nil && *cfg.JFHost != "" {
		target = *cfg.JFHost
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}

	reqURL := strings.TrimRight(target, "/") + "/System/Info/Public"
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(reqURL)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cannot reach Jellyfin server"})
		return
	}
	defer resp.Body.Close()

	var info struct {
		ServerName string `json:"ServerName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil || info.ServerName == "" {
		info.ServerName = target
	}
	c.JSON(http.StatusOK, gin.H{"serverName": info.ServerName, "url": target})
}

// POST /auth/jellyfin-login
// Body: {username, password, deviceId?, deviceName?}. deviceId should be a
// random id persisted by the browser so each browser gets its own Jellyfin
// token (Jellyfin revokes the previous token of the same user+device).
func (h *FrontendAuthHandler) JellyfinLogin(c *gin.Context) {
	var body struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		DeviceId   string `json:"deviceId"`
		DeviceName string `json:"deviceName"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	token, err := h.auth.Login(c.Request.Context(), body.Username, body.Password, body.DeviceId, body.DeviceName)
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
	case errors.Is(err, auth.ErrNotAdmin):
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
	case errors.Is(err, auth.ErrNotConfigured):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Jellyfin not configured"})
	case err != nil:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Jellyfin unreachable"})
	default:
		c.JSON(http.StatusOK, gin.H{"token": token})
	}
}

// POST /auth/logout — revokes the Jellyfin token and deletes the session.
func (h *FrontendAuthHandler) Logout(c *gin.Context) {
	_ = h.auth.Logout(c.Request.Context(), middleware.GetClaims(c))
	c.Status(http.StatusNoContent)
}

// GET /auth/me — returns the current user (also used to check the session).
func (h *FrontendAuthHandler) Me(c *gin.Context) {
	cl := middleware.GetClaims(c)
	c.JSON(http.StatusOK, gin.H{"userId": cl.UserId, "userName": cl.UserName, "isAdmin": cl.IsAdmin})
}
