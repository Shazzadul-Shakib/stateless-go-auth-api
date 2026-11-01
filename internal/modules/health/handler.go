package health

import (
	"net/http"

	"github.com/Shazzadul-Shakib/go-auth-api/internal/response"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) HealthCheck(c *gin.Context) {
	res := h.svc.Check(c.Request.Context())
	code, msg := http.StatusOK, "service is healthy"
	if res.Status != "ok" {
		code, msg = http.StatusServiceUnavailable, "service is unavailable"
	}
	response.JSON(c, code, res, msg)
}
