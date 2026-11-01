package middleware

import (
	"net/http"
	"strings"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/apperr"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const userIDKey = "userID"

var ErrUnauthorized = apperr.New(http.StatusUnauthorized, "unauthorized", "missing or invalid access token")

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
			apperr.Respond(c, ErrUnauthorized)
			return
		}

		claims, err := utils.ParseToken(token, utils.AccessToken)
		if err != nil {
			apperr.Respond(c, ErrUnauthorized)
			return
		}

		id, err := uuid.Parse(claims.Subject)
		if err != nil {
			apperr.Respond(c, ErrUnauthorized)
			return
		}

		c.Set(userIDKey, id)
		c.Next()
	}
}

// UserID returns the logged-in user's ID set by Auth.
func UserID(c *gin.Context) (uuid.UUID, bool) {
	id, ok := c.Get(userIDKey)
	if !ok {
		return uuid.Nil, false
	}
	uid, ok := id.(uuid.UUID)
	return uid, ok
}
