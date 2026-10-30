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
	res := h.svc.Check(c.Request.Context())
	code := http.StatusOK
	if res.Status != "ok" {
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, res)
}
