package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/middleware"
)

func TestRateLimit_AllowsNormalTraffic(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RateLimit(10, 10))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
	}
}

func TestRateLimit_BlocksExcessTraffic(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RateLimit(1, 1))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	// First request should pass
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Errorf("first request: expected 200, got %d", w1.Code)
	}

	// Rapid second request should be rate limited
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != 429 {
		t.Errorf("second request: expected 429, got %d", w2.Code)
	}
}
