package middleware

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const meUserKey = "meUserId"

// Headers set by the Jellystics plugin on every /api/me request.
const (
	HeaderMeSecret = "X-Jellystics-Secret"
	HeaderMeUser   = "X-Jellystics-User"
)

// PluginMe guards /api/me: the request must carry the plugin shared secret
// (constant-time compare, else 401) and, when requireUser, the Jellyfin user
// id the plugin authenticated (normalised to the N format; unknown user →
// 403). The user id is stored in the context (MeUserID). Jellystics session
// tokens are not accepted: this is a separate trust boundary.
func PluginMe(secret func(context.Context) (string, error), userExists func(context.Context, string) (bool, error), requireUser bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		want, err := secret(c.Request.Context())
		if err != nil {
			log.Printf("api me: secret: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		got := c.GetHeader(HeaderMeSecret)
		if want == "" || got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid secret"})
			return
		}
		if !requireUser {
			c.Next()
			return
		}
		id := NormalizeUserID(c.GetHeader(HeaderMeUser))
		if id == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "unknown user"})
			return
		}
		ok, err := userExists(c.Request.Context(), id)
		if err != nil {
			log.Printf("api me: user lookup: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "unknown user"})
			return
		}
		c.Set(meUserKey, id)
		c.Next()
	}
}

// NormalizeUserID returns a Jellyfin id in the N format (32 lowercase hex,
// no dashes); other non-empty values are returned trimmed, unchanged.
func NormalizeUserID(s string) string {
	s = strings.TrimSpace(s)
	t := strings.ToLower(strings.ReplaceAll(s, "-", ""))
	if len(t) != 32 {
		return s
	}
	for _, c := range t {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return s
		}
	}
	return t
}

// MeUserID returns the user id set by PluginMe ("" when absent).
func MeUserID(c *gin.Context) string {
	v, _ := c.Get(meUserKey)
	s, _ := v.(string)
	return s
}
