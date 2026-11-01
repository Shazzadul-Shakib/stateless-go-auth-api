package user

import (
	"net/http"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/apperr"
)

var (
	ErrUserNotFound = apperr.New(http.StatusNotFound, "user_not_found", "user not found")
)
