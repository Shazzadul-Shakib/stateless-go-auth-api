package router

import (
	"github.com/Shazzadul-Shakib/go-auth-api/internal/modules/health"
	"github.com/gin-gonic/gin"
)

func New() *gin.Engine {
	r := gin.Default()
	api:= r.Group("/api/v1")

	healthHandler := health.NewHandler(health.NewService())
	health.RegisterRoutes(api, healthHandler)
	return r
}