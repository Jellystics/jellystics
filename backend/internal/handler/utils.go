package handler

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/Jellystics/Jellystics/internal/service/geo"
	"github.com/gin-gonic/gin"
)

// UtilsHandler handles miscellaneous utility endpoints.
type UtilsHandler struct{}

func NewUtilsHandler() *UtilsHandler { return &UtilsHandler{} }

// isPublicIPv4 returns true if the string is a valid IPv4 address that is not
// in a private/loopback range (10.x, 172.16-31.x, 192.168.x, 127.x).
func isPublicIPv4(s string) bool {
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	ip = ip.To4()
	if ip == nil {
		return false
	}
	private := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8"}
	for _, cidr := range private {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return false
		}
	}
	return true
}

// POST /utils/geolocateIp
func (h *UtilsHandler) GeolocateIP(c *gin.Context) {
	gl := geo.NewGeoLiteFromEnv()
	if !gl.Configured() {
		c.JSON(http.StatusNotImplemented, "GeoLite information missing!")
		return
	}

	var body struct {
		IPAddress string `json:"ipAddress"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || !isPublicIPv4(body.IPAddress) {
		c.JSON(http.StatusBadRequest, "Invalid IP address sent!")
		return
	}

	status, data, err := gl.Raw(c.Request.Context(), body.IPAddress)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		c.Data(status, "application/json", data)
		return
	}
	c.JSON(status, result)
}
