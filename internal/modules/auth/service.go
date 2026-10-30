package auth

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrEmailTaken = errors.New("email already registered")

type Service struct{ repo Repository }

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) RegisterUser(ctx context.Context, in RegisterUser) (*UserResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{Name: in.Name, Email: in.Email, Password: string(hash)}
	if err := s.repo.Create(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	return &UserResponse{ID: user.ID, Name: user.Name, Email: user.Email}, nil
}
