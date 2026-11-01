package user

import (
	"context"

	"github.com/google/uuid"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// get logged user info service
func (s *Service) GetLoggedUserInfoService(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.repo.GetLoggedUserInfo(ctx, id)
	if err != nil {
		return nil, err
	}
	return &User{ID: user.ID, Name: user.Name, Email: user.Email}, nil
}
