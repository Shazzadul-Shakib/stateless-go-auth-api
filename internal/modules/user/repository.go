package user

import (
	"context"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/modules/auth"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	GetLoggedUserInfo(ctx context.Context, id uuid.UUID) (*auth.User, error)
}
type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) GetLoggedUserInfo(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	var user auth.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
