package repository

import (
	"context"
	"strings"
)

// SeedJellyfinConfig fills the Jellyfin server URL / API key of app_config
// from the JF_HOST / JF_API_KEY environment variables at startup. A value
// already stored (setup wizard, settings UI) always wins: the host is only
// set when none is stored, and the key only when none is stored and the
// stored host is the env one. It reports which values were seeded.
func SeedJellyfinConfig(ctx context.Context, cfgRepo ConfigRepository, host, apiKey string) (hostSeeded, keySeeded bool, err error) {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	apiKey = strings.TrimSpace(apiKey)
	if host == "" {
		return false, false, nil
	}
	cfg, err := cfgRepo.Get(ctx)
	if err != nil {
		return false, false, err
	}
	if cfg.JFHost == nil || strings.TrimSpace(*cfg.JFHost) == "" {
		cfg.JFHost = &host
		hostSeeded = true
	}
	if apiKey != "" && (cfg.JFApiKey == nil || *cfg.JFApiKey == "") && strings.TrimRight(*cfg.JFHost, "/") == host {
		cfg.JFApiKey = &apiKey
		keySeeded = true
	}
	if !hostSeeded && !keySeeded {
		return false, false, nil
	}
	return hostSeeded, keySeeded, cfgRepo.Save(ctx, cfg)
}
