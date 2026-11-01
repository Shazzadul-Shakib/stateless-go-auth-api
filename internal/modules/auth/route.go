package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	r.POST("/auth/register", h.RegisterUserHandler)
	r.GET("/auth/user/:email", h.GetUserByEmailHandler)
	r.POST("/auth/login", h.LoginHandler)

}
