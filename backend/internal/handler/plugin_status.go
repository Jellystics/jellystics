package handler

import (
	"net/http"

	"github.com/Jellystics/Jellystics/internal/service/pluginsync"
	"github.com/gin-gonic/gin"
)

// PluginStatus serves GET /api/plugin/status: the Jellystics plugin sync
// state (detected, versions, cursors, last success / error, healthy).
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
