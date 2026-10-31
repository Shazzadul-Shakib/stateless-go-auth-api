package auth

import (
	"context"
	"errors"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/utils"
	"gorm.io/gorm"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) RegisterUser(ctx context.Context, in RegisterUser) (*UserResponse, error) {
	hash, err := utils.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.Create(ctx, &User{Name: in.Name, Email: in.Email, Password: hash})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	return &UserResponse{ID: user.ID, Name: user.Name, Email: user.Email}, nil
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*UserResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return &UserResponse{ID: user.ID, Name: user.Name, Email: user.Email}, nil
}
