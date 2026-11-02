package auth

import (
	"errors"
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

	h.respondWithTokens(c, res, "login successful")
}

// refresh token handler
func (h *Handler) RefreshTokenHandler(c *gin.Context) {
	token := refreshTokenFrom(c)
	if token == "" {
		middleware.ClearAuthCookies(c)
		apperr.Respond(c, ErrInvalidToken)
		return
	}

	res, err := h.svc.RefreshToken(c.Request.Context(), RefreshTokenRequest{RefreshToken: token})
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			middleware.ClearAuthCookies(c)
		}
		apperr.Respond(c, err)
		return
	}

	h.respondWithTokens(c, res, "token refreshed")
}

// logout handler
func (h *Handler) LogoutHandler(c *gin.Context) {
	if token := refreshTokenFrom(c); token != "" {
		if err := h.svc.Logout(c.Request.Context(), RefreshTokenRequest{RefreshToken: token}); err != nil {
			apperr.Respond(c, err)
			return
		}
	}

	middleware.ClearAuthCookies(c)
	response.JSON(c, http.StatusOK, nil, "logged out")
}

// logout from all devices handler (requires a valid access token)
func (h *Handler) LogoutAllHandler(c *gin.Context) {
	id, ok := middleware.UserID(c)
	if !ok {
		apperr.Respond(c, middleware.ErrUnauthorized)
		return
	}

	if err := h.svc.LogoutAll(c.Request.Context(), id); err != nil {
		apperr.Respond(c, err)
		return
	}

	middleware.ClearAuthCookies(c)
	response.JSON(c, http.StatusOK, nil, "logged out from all devices")
}

// respondWithTokens sets the auth cookies. The tokens are only put in the body for
// non-browser clients that ask for it with "X-Token-Delivery: body", so scripts in a
// browser never see them.
func (h *Handler) respondWithTokens(c *gin.Context, res *LoginResponse, message string) {
	if err := middleware.SetAuthCookies(c, res.AccessToken, res.RefreshToken); err != nil {
		apperr.Respond(c, err)
		return
	}
	if c.GetHeader("X-Token-Delivery") == "body" {
		response.JSON(c, http.StatusOK, res, message)
		return
	}
	response.JSON(c, http.StatusOK, nil, message)
}

// refreshTokenFrom reads the refresh token from the cookie, falling back to the JSON body.
func refreshTokenFrom(c *gin.Context) string {
	if token, err := c.Cookie(middleware.RefreshCookie); err == nil && token != "" {
		return token
	}
	var in RefreshTokenRequest
	_ = c.ShouldBindJSON(&in)
	return in.RefreshToken
}
