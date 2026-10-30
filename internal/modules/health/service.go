package health

import (
	"context"

	"gorm.io/gorm"
)

type Status struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Check(ctx context.Context) Status {
	sqlDB, err := s.db.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		return Status{Status: "degraded", Database: "down"}
	}
	return Status{Status: "ok", Database: "up"}
}
