package auth

import (
	"net/http"

	apperr "github.com/Shazzadul-Shakib/go-auth-api/internal/errors"
)

var (
	ErrEmailTaken   = apperr.New(http.StatusConflict, "email_taken", "email already registered")
	ErrUserNotFound = apperr.New(http.StatusNotFound, "user_not_found", "user not found")
)
