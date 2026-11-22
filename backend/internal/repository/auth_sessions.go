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
	// Find + Limit (not First): a missing session is a normal case (logged out,
	// revoked) and must not be logged as an error by GORM.
	res := r.db.WithContext(ctx).Where("id = ?", id).Limit(1).Find(&s)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
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

func (r *authSessionRepo) ListIdle(ctx context.Context, before time.Time) ([]models.AuthSession, error) {
	var out []models.AuthSession
	return out, r.db.WithContext(ctx).Where("last_seen_at < ?", before).Find(&out).Error
}
