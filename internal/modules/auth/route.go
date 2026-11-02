package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup, h *Handler, requireAuth gin.HandlerFunc) {
	r.POST("/auth/register", h.RegisterUserHandler)
	r.GET("/auth/user/:email", h.GetUserByEmailHandler)
	r.POST("/auth/login", h.LoginHandler)
	r.POST("/auth/refresh", h.RefreshTokenHandler)
	r.POST("/auth/logout", h.LogoutHandler)
	r.POST("/auth/logout-all", requireAuth, h.LogoutAllHandler)
}
