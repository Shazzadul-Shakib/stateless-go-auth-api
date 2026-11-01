package auth

import (
	"net/http"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/apperr"
)

var (
	ErrEmailTaken         = apperr.New(http.StatusConflict, "email_taken", "email already registered")
	ErrUserNotFound       = apperr.New(http.StatusNotFound, "user_not_found", "user not found")
	ErrInvalidCredentials = apperr.New(http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	ErrInvalidToken       = apperr.New(http.StatusUnauthorized, "invalid_token", "invalid or expired token")
)
