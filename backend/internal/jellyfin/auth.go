package jellyfin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// ErrUnauthorized is returned when Jellyfin rejects a token or credentials.
var ErrUnauthorized = errors.New("jellyfin: unauthorized")

// DeviceInfo identifies the device a Jellyfin access token is issued to.
// Jellyfin revokes any previous token for the same (user, DeviceId) pair on
// login (SessionManager.GetAuthorizationToken), so every browser must use its
// own DeviceId or logging in on one browser logs the others out.
type DeviceInfo struct {
	DeviceId   string
	DeviceName string
}

var unsafeHeaderChars = regexp.MustCompile(`[^A-Za-z0-9 ._:()\-]`)

func sanitizeHeaderValue(s string, max int) string {
	s = unsafeHeaderChars.ReplaceAllString(s, "")
	if len(s) > max {
		s = s[:max]
	}
	return strings.TrimSpace(s)
}

// AuthorizationHeaderFor builds an `Authorization: MediaBrowser ...` header for
// a specific device.
func AuthorizationHeaderFor(token string, dev DeviceInfo) string {
	id := sanitizeHeaderValue(dev.DeviceId, 64)
	if id == "" {
		id = "jellystics-server"
	}
	name := sanitizeHeaderValue(dev.DeviceName, 64)
	if name == "" {
		name = "Jellystics"
	}
	h := `MediaBrowser `
	if token != "" {
		h += fmt.Sprintf("Token=%q, ", token)
	}
	return h + fmt.Sprintf(`Client="Jellystics", Device=%q, DeviceId=%q, Version="1.0"`, name, id)
}

// AuthenticateUserOnDevice logs a user in and returns Jellyfin's access token
// bound to the given device.
func (c *Client) AuthenticateUserOnDevice(ctx context.Context, username, password string, dev DeviceInfo) (*AuthResponse, error) {
	body, _ := json.Marshal(map[string]string{"Username": username, "Pw": password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/Users/AuthenticateByName", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", AuthorizationHeaderFor("", dev))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("authenticate: status %d", resp.StatusCode)
	}
	var auth AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return nil, err
	}
	return &auth, nil
}

// GetMe returns the user owning the given access token (GET /Users/Me).
// Returns ErrUnauthorized when the token has been revoked or is invalid.
func (c *Client) GetMe(ctx context.Context, token string, dev DeviceInfo) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/Users/Me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", AuthorizationHeaderFor(token, dev))
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET /Users/Me: status %d", resp.StatusCode)
	}
	var u User
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

// Logout revokes the given access token (POST /Sessions/Logout).
func (c *Client) Logout(ctx context.Context, token string, dev DeviceInfo) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/Sessions/Logout", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", AuthorizationHeaderFor(token, dev))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusUnauthorized {
		return fmt.Errorf("logout: status %d", resp.StatusCode)
	}
	return nil
}
