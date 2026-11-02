package router

import (
	"github.com/Shazzadul-Shakib/go-auth-api/internal/middleware"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/modules/auth"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/modules/health"
	"github.com/Shazzadul-Shakib/go-auth-api/internal/modules/user"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func New(db *gorm.DB, rdb *redis.Client) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORS(), middleware.CSRF())
	api := r.Group("/api/v1")

	healthHandler := health.NewHandler(health.NewService(db))
	health.RegisterRoutes(api, healthHandler)

	authHandler := auth.NewHandler(auth.NewService(auth.NewRepository(db), auth.NewSessionStore(rdb)))
	auth.RegisterRoutes(api, authHandler, middleware.Auth())

	userHandler := user.NewHandler(user.NewService(user.NewRepository(db)))
	user.RegisterRoutes(api.Group("", middleware.Auth()), userHandler)
	return r
}
