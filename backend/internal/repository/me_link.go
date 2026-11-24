package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"gorm.io/gorm"
)

// MeLinkRepository stores the « Mes statistiques » shared secret
// (app_config.me_secret, outside the AppConfig model so a config save never
// overwrites it).
type MeLinkRepository interface {
	// Secret returns the plugin shared secret, generating it on first use
	// (32 random bytes, hex). Concurrent first calls agree on one value.
	Secret(ctx context.Context) (string, error)
}

type meLinkRepo struct{ db *gorm.DB }

func (r *meLinkRepo) Secret(ctx context.Context) (string, error) {
	var s string
	if err := r.db.WithContext(ctx).Raw(`SELECT me_secret FROM app_config WHERE "ID" = 1`).Scan(&s).Error; err != nil {
		return "", err
	}
	if s != "" {
		return s, nil
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Only the first writer wins; everyone re-reads the stored value.
	if err := r.db.WithContext(ctx).Exec(`UPDATE app_config SET me_secret = ? WHERE "ID" = 1 AND me_secret = ''`, hex.EncodeToString(b)).Error; err != nil {
		return "", err
	}
	err := r.db.WithContext(ctx).Raw(`SELECT me_secret FROM app_config WHERE "ID" = 1`).Scan(&s).Error
	return s, err
}
