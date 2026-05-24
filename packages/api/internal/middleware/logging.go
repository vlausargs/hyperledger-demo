package middleware

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// CorrelationID attaches a correlation ID to every request.
// Reads X-Correlation-ID header or generates one; echoes it in the response.
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := c.GetHeader("X-Correlation-ID")
		if cid == "" {
			cid = newCorrelationID()
		}
		c.Set("correlationID", cid)
		c.Header("X-Correlation-ID", cid)
		c.Next()
	}
}

// RequestLogger logs incoming requests as structured JSON.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if q := c.Request.URL.RawQuery; q != "" {
			path = path + "?" + q
		}

		c.Next()

		cid, _ := c.Get("correlationID")
		slog.Info("request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
			"correlation_id", cid,
		)
	}
}

// newCorrelationID generates a random hex correlation ID.
func newCorrelationID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
