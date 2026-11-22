package handler

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
)

type ProxyApiHandler struct {
	repos      *repository.Container
	httpClient *http.Client
}

func NewProxyApiHandler(repos *repository.Container) *ProxyApiHandler {
	return &ProxyApiHandler{
		repos:      repos,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

var (
	proxyImagePath = regexp.MustCompile(`^/(Items|Users)/Images/(Primary|Backdrop|Thumb|Logo|Banner|Art)/?$`)
	proxyItemID    = regexp.MustCompile(`^[0-9a-fA-F]{32}$|^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	proxyImageArgs = map[string]bool{
		"fillWidth": true, "fillHeight": true, "maxWidth": true, "maxHeight": true,
		"width": true, "height": true, "quality": true, "tag": true, "format": true,
	}
	proxyRespHeaders = []string{"Content-Type", "Content-Length", "Cache-Control", "ETag", "Last-Modified", "Expires"}
)

// Proxy returns the GET /proxy/*path handler.
//
// Only Jellyfin *image* endpoints are proxied, and only for a well-formed item
// id: the request is made with the admin API key, so forwarding arbitrary paths
// would hand full Jellyfin admin access to anyone. The JSON helpers
// (getSessions, getAdminUsers, getRecentlyAdded) are gated by authMW.
func (h *ProxyApiHandler) Proxy(authMW gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawPath := c.Param("path")

		switch rawPath {
		case "/getSessions", "/getAdminUsers", "/getRecentlyAdded":
			authMW(c)
			if c.IsAborted() {
				return
			}
			switch rawPath {
			case "/getSessions":
				h.GetSessions(c)
			case "/getAdminUsers":
				h.GetAdminUsers(c)
			default:
				h.GetRecentlyAdded(c)
			}
			return
		}

		m := proxyImagePath.FindStringSubmatch(rawPath)
		id := c.Query("id")
		if m == nil || !proxyItemID.MatchString(id) {
			c.Status(http.StatusNotFound)
			return
		}
		jfPath := "/" + m[1] + "/" + id + "/Images/" + m[2]

		cfg, err := h.repos.Config.Get(c.Request.Context())
		if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
			c.Status(http.StatusServiceUnavailable)
			return
		}

		q := url.Values{}
		for k, v := range c.Request.URL.Query() {
			if proxyImageArgs[k] && len(v) > 0 {
				q.Set(k, v[0])
			}
		}
		jfURL := strings.TrimRight(*cfg.JFHost, "/") + jfPath
		if len(q) > 0 {
			jfURL += "?" + q.Encode()
		}

		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, jfURL, nil)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if cfg.JFApiKey != nil && *cfg.JFApiKey != "" {
			req.Header.Set("Authorization", jellyfin.AuthorizationHeader(*cfg.JFApiKey))
		}

		resp, err := h.httpClient.Do(req)
		if err != nil {
			c.Status(http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for _, k := range proxyRespHeaders {
			if v := resp.Header.Get(k); v != "" {
				c.Header(k, v)
			}
		}
		c.Status(resp.StatusCode)
		_, _ = io.Copy(c.Writer, resp.Body)
	}
}

// GET /proxy/getSessions — returns live Jellyfin sessions.
func (h *ProxyApiHandler) GetSessions(c *gin.Context) {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "jellyfin not configured"})
		return
	}
	apiKey := ""
	if cfg.JFApiKey != nil {
		apiKey = *cfg.JFApiKey
	}
	jf := jellyfin.NewClient(*cfg.JFHost, apiKey)
	sessions, err := jf.GetSessions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sessions)
}

// GET /proxy/getAdminUsers — returns Jellyfin users with IsAdministrator = true.
func (h *ProxyApiHandler) GetAdminUsers(c *gin.Context) {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "jellyfin not configured"})
		return
	}
	apiKey := ""
	if cfg.JFApiKey != nil {
		apiKey = *cfg.JFApiKey
	}
	jf := jellyfin.NewClient(*cfg.JFHost, apiKey)
	users, err := jf.GetUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	admins := make([]jellyfin.User, 0)
	for _, u := range users {
		if u.Policy.IsAdministrator {
			admins = append(admins, u)
		}
	}
	c.JSON(http.StatusOK, admins)
}

// GET /proxy/getRecentlyAdded?libraryid=<id> — returns recently added items for a library.
func (h *ProxyApiHandler) GetRecentlyAdded(c *gin.Context) {
	libraryId := c.Query("libraryid")

	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "jellyfin not configured"})
		return
	}
	apiKey := ""
	if cfg.JFApiKey != nil {
		apiKey = *cfg.JFApiKey
	}
	jf := jellyfin.NewClient(*cfg.JFHost, apiKey)

	// Find the first admin user to use for the /Users/{id}/Items/Latest endpoint.
	users, err := jf.GetUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	var adminId string
	for _, u := range users {
		if u.Policy.IsAdministrator {
			adminId = u.Id
			break
		}
	}
	if adminId == "" && len(users) > 0 {
		adminId = users[0].Id
	}
	if adminId == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no users found"})
		return
	}

	items, err := jf.GetRecentlyAdded(c.Request.Context(), adminId, libraryId, 20, jellyfin.StandardFields)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

// POST /proxy/validateSettings — validates a Jellyfin URL + API key.
// Body: {"url": "...", "apikey": "..."}
func (h *ProxyApiHandler) ValidateSettings(c *gin.Context) {
	var body struct {
		URL    string `json:"url"`
		APIKey string `json:"apikey"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.URL == "" || body.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url and apikey are required"})
		return
	}

	// Normalise the URL (strip trailing slash, add scheme if missing).
	rawURL := body.URL
	rawURL = strings.TrimSuffix(rawURL, "/")
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}

	info, _, err := jellyfin.GetSystemInfo(c.Request.Context(), rawURL, body.APIKey)
	if err != nil {
		statusCode := http.StatusBadGateway
		c.JSON(http.StatusOK, gin.H{
			"isValid":      false,
			"status":       statusCode,
			"errorMessage": err.Error(),
		})
		return
	}

	serverName, _ := info["ServerName"].(string)
	serverId, _ := info["Id"].(string)
	c.JSON(http.StatusOK, gin.H{
		"isValid":    true,
		"serverName": serverName,
		"serverId":   serverId,
	})
}
