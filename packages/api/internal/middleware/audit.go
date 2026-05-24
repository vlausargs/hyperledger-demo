package middleware

import (
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var auditLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

// AuditLog logs state-changing operations (non-GET/OPTIONS/HEAD) with user context.
func AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" || c.Request.Method == "HEAD" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		sub, _ := c.Get("sub")
		org, _ := c.Get("org")
		cid, _ := c.Get("correlationID")

		auditLogger.Info("audit",
			"action", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"user", sub,
			"org", org,
			"ip", c.ClientIP(),
			"correlation_id", cid,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
}
