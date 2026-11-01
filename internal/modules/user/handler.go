package user

import (
	"net/http"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/apperr"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/middleware"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// get logged user info handler
func (h *Handler) GetLoggedUserInfoHandler(c *gin.Context) {
	id, ok := middleware.UserID(c)
	if !ok {
		apperr.Respond(c, middleware.ErrUnauthorized)
		return
	}

	user, err := h.svc.GetLoggedUserInfoService(c.Request.Context(), id)
	if err != nil {
		apperr.Respond(c, err)
		return
	}

	response.JSON(c, http.StatusOK, user, "user fetched")
}
