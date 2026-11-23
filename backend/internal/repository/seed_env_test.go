package repository_test

import (
	"context"
	"testing"

	"github.com/Jellystics/Jellystics/internal/repository"
)

func TestSeedJellyfinConfig(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	repos := repository.New(db)
	get := func() (string, string) {
		t.Helper()
		cfg, err := repos.Config.Get(ctx)
		if err != nil {
			t.Fatal(err)
		}
		h, k := "", ""
		if cfg.JFHost != nil {
			h = *cfg.JFHost
		}
		if cfg.JFApiKey != nil {
			k = *cfg.JFApiKey
		}
		return h, k
	}

	// No env: nothing happens.
	if h, k, err := repository.SeedJellyfinConfig(ctx, repos.Config, "", "key"); err != nil || h || k {
		t.Fatalf("empty host: %v %v %v", h, k, err)
	}
	// Empty DB: host (trailing slash trimmed) and key are seeded.
	h, k, err := repository.SeedJellyfinConfig(ctx, repos.Config, " http://jf:8096/ ", "env-key")
	if err != nil || !h || !k {
		t.Fatalf("seed: %v %v %v", h, k, err)
	}
	if host, key := get(); host != "http://jf:8096" || key != "env-key" {
		t.Fatalf("config = %q %q", host, key)
	}

	// A config set from the UI is never overwritten.
	db.Exec(`UPDATE app_config SET "JF_HOST" = 'http://ui:8096', "JF_API_KEY" = 'ui-key' WHERE "ID" = 1`)
	if h, k, err := repository.SeedJellyfinConfig(ctx, repos.Config, "http://jf:8096", "env-key"); err != nil || h || k {
		t.Fatalf("overwrite: %v %v %v", h, k, err)
	}
	if host, key := get(); host != "http://ui:8096" || key != "ui-key" {
		t.Fatalf("config = %q %q", host, key)
	}

	// Missing key: only filled when the stored host is the env one.
	db.Exec(`UPDATE app_config SET "JF_API_KEY" = NULL WHERE "ID" = 1`)
	if h, k, _ := repository.SeedJellyfinConfig(ctx, repos.Config, "http://jf:8096", "env-key"); h || k {
		t.Fatal("key seeded for another host")
	}
	if h, k, _ := repository.SeedJellyfinConfig(ctx, repos.Config, "http://ui:8096/", "env-key"); h || !k {
		t.Fatal("key not seeded for the same host")
	}
	if host, key := get(); host != "http://ui:8096" || key != "env-key" {
		t.Fatalf("config = %q %q", host, key)
	}
}
