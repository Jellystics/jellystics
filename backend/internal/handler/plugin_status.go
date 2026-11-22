package handler

import (
	"errors"
	"net/http"

	"github.com/Jellystics/Jellystics/internal/service/pluginsync"
	"github.com/gin-gonic/gin"
)

// PluginStatus serves GET /api/plugin/status: the Jellystics plugin sync
// state (detected, versions, cursors, last success / error, healthy, plays
// synced).
func PluginStatus(svc *pluginsync.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		st, err := svc.Status(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, st)
	}
}

// PluginPushHistory serves POST /api/plugin/push-history: starts (or
// resumes) the background push of the Jellystics history to the plugin.
// 202 with the push status; 409 when a push is already running or the
// plugin is absent / incompatible / Jellyfin not configured.
func PluginPushHistory(svc *pluginsync.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		st, err := svc.StartPush(c.Request.Context())
		switch {
		case errors.Is(err, pluginsync.ErrPushRunning), errors.Is(err, pluginsync.ErrPluginAbsent),
			errors.Is(err, pluginsync.ErrPluginIncompatible), errors.Is(err, pluginsync.ErrNotConfigured):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case err != nil:
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusAccepted, st)
		}
	}
}

// PluginPushStatus serves GET /api/plugin/push-history/status.
func PluginPushStatus(svc *pluginsync.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		st, err := svc.PushStatus(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, st)
	}
}

// PluginResync serves POST /api/plugin/resync: cursors back to 0 and a sync
// run in the background (202).
func PluginResync(svc *pluginsync.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.Resync(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"message": "resync started"})
	}
}
