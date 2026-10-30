package router

import (
	"github.com/Shazzadul-Shakib/go-auth-api/internal/modules/health"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	api := r.Group("/api/v1")

	healthHandler := health.NewHandler(health.NewService(db))
	health.RegisterRoutes(api, healthHandler)
	return r
}
