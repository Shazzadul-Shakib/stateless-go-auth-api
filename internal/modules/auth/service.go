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

// RegisterUser service
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

// GetUserByEmail service
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*UserResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return &UserResponse{ID: user.ID, Name: user.Name, Email: user.Email}, nil
}

// login service
func (s *Service) Login(ctx context.Context, in LoginUser) (*LoginResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}

	isValid := utils.CheckPasswordHash(in.Password, user.Password)

	if !isValid {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := utils.GenerateAccessToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	return &LoginResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}
