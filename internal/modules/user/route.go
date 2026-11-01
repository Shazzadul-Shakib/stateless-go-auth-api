package user

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	r.GET("/users/me", h.GetLoggedUserInfoHandler)
}
