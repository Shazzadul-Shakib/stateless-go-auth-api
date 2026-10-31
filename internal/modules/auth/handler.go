package auth

import (
	"net/http"

	apperr "github.com/Shazzadul-Shakib/go-auth-api/internal/errors"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterUserHandler(c *gin.Context) {
	var in RegisterUser
	if err := c.ShouldBindJSON(&in); err != nil {
		apperr.Respond(c, err)
		return
	}

	res, err := h.svc.RegisterUser(c.Request.Context(), in)
	if err != nil {
		apperr.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, res)
}

func (h *Handler) GetUserByEmailHandler(c *gin.Context) {
	email := c.Param("email")
	user, err := h.svc.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		apperr.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}
