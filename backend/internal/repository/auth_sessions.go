package repository

import (
	"context"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"gorm.io/gorm"
)

type authSessionRepo struct{ db *gorm.DB }

func (r *authSessionRepo) Create(ctx context.Context, s *models.AuthSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *authSessionRepo) Get(ctx context.Context, id string) (*models.AuthSession, error) {
	var s models.AuthSession
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// Touch updates last_seen_at, and last_validated_at when validated is true.
func (r *authSessionRepo) Touch(ctx context.Context, id string, validated bool) error {
	now := time.Now()
	updates := map[string]any{"last_seen_at": now}
	if validated {
		updates["last_validated_at"] = now
	}
	return r.db.WithContext(ctx).Model(&models.AuthSession{}).Where("id = ?", id).Updates(updates).Error
}

func (r *authSessionRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.AuthSession{}).Error
}
