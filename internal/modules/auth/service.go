package auth

import (
	"context"
	"errors"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	repo     Repository
	sessions *SessionStore
}

func NewService(repo Repository, sessions *SessionStore) *Service {
	return &Service{repo: repo, sessions: sessions}
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

	userID := user.ID.String()
	sessionID, refreshToken, err := s.sessions.Create(ctx, userID)
	if err != nil {
		return nil, err
	}

	accessToken, err := utils.GenerateAccessToken(userID, sessionID)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// refreshToken service: rotates the refresh token and issues a new access token
func (s *Service) RefreshToken(ctx context.Context, in RefreshTokenRequest) (*LoginResponse, error) {
	rec, refreshToken, err := s.sessions.Rotate(ctx, in.RefreshToken)
	if err != nil {
		return nil, err
	}

	accessToken, err := utils.GenerateAccessToken(rec.UserID, rec.SessionID)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

// logout service: ends the session owning the refresh token
func (s *Service) Logout(ctx context.Context, in RefreshTokenRequest) error {
	return s.sessions.Revoke(ctx, in.RefreshToken)
}

// logout-all service: ends every session of the user
func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.sessions.RevokeAll(ctx, userID.String())
}
