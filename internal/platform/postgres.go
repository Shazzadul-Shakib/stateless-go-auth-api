package platform

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgres(ctx context.Context, url string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}
	return db, nil
}
