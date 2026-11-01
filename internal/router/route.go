package router

import (
	"github.com/Shazzadul-Shakib/go-auth-api/internal/middleware"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/modules/auth"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/modules/health"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/modules/user"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	api := r.Group("/api/v1")

	healthHandler := health.NewHandler(health.NewService(db))
	health.RegisterRoutes(api, healthHandler)

	authHandler := auth.NewHandler(auth.NewService(auth.NewRepository(db)))
	auth.RegisterRoutes(api, authHandler)

	userHandler := user.NewHandler(user.NewService(user.NewRepository(db)))
	user.RegisterRoutes(api.Group("", middleware.Auth()), userHandler)
	return r
}
