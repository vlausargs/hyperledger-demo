package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck verifies ledger connectivity.
func HealthCheck(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := gw.EvaluateTransaction("GetAllProducts", "1", "")
		if err != nil {
			slog.Error("health check failed", "error", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "unhealthy",
				"service": "Fabric Client API",
				"error":   "ledger unreachable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "Fabric Client API",
			"version": "2.0.0",
		})
	}
}
