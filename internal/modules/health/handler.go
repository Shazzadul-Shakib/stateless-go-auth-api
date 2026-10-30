package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.Check())
}
