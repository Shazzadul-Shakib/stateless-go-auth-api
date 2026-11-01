package auth

import (
	"net/http"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/apperr"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// register user handler
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

	response.JSON(c, http.StatusCreated, res, "user registered")
}

// get user by email handler
func (h *Handler) GetUserByEmailHandler(c *gin.Context) {
	email := c.Param("email")
	user, err := h.svc.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		apperr.Respond(c, err)
		return
	}

	response.JSON(c, http.StatusOK, user, "user fetched")
}

// login handler
func (h *Handler) LoginHandler(c *gin.Context) {
	var in LoginUser
	if err := c.ShouldBindJSON(&in); err != nil {
		apperr.Respond(c, err)
		return
	}

	res, err := h.svc.Login(c.Request.Context(), in)
	if err != nil {
		apperr.Respond(c, err)
		return
	}

	response.JSON(c, http.StatusOK, res, "login successful")
}

// refresh token handler
