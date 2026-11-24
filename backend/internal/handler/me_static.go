package handler

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// meIndex is the « Mes statistiques » entry of the frontend build.
const meIndex = "me.html"

// MeStatic serves GET /me/*path (public, no data inside): the me app of the
// frontend build, proxied by the plugin under /Jellystics/Me/. /me/ and
// client-side routes (no extension) → me.html; other paths are build files
// (assets/…), 404 when missing. 404 as well while me.html is not built.
func MeStatic(webFS fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimPrefix(path.Clean("/"+c.Param("path")), "/")
		if name != "" && fs.ValidPath(name) {
			if st, err := fs.Stat(webFS, name); err == nil && !st.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					c.Header("Cache-Control", "public, max-age=31536000, immutable")
				}
				http.ServeFileFS(c.Writer, c.Request, webFS, name)
				return
			}
			if path.Ext(name) != "" {
				c.String(http.StatusNotFound, "not found")
				return
			}
		}
		if _, err := fs.Stat(webFS, meIndex); err != nil {
			c.String(http.StatusNotFound, "not found")
			return
		}
		c.Header("Cache-Control", "no-cache")
		http.ServeFileFS(c.Writer, c.Request, webFS, meIndex)
	}
}
