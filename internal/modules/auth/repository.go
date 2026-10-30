package auth

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, user *User) error
}

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}
