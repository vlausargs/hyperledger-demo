package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/service"
	"github.com/myindo/hlf-supply-chain/api/pkg/response"
)

// HealthHandler exposes /health backed by HealthService.
type HealthHandler struct {
	svc *service.HealthService
}

func NewHealthHandler(svc *service.HealthService) *HealthHandler {
	return &HealthHandler{svc: svc}
}

func (h *HealthHandler) Check(c *gin.Context) {
	if err := h.svc.Check(c.Request.Context()); err != nil {
		slog.Error("health check failed", "error", err.Detail)
		response.Unavailable(c, "ledger unreachable")
		return
	}
	response.OK(c, gin.H{
		"status":  "healthy",
		"service": "Fabric Client API",
		"version": "2.0.0",
	})
}
