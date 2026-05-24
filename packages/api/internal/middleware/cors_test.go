package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/middleware"
)

func TestCORS_SetsHeaders(t *testing.T) {
	r := gin.New()
	r.Use(middleware.CORS("http://localhost:3000"))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("expected origin http://localhost:3000, got %s", got)
	}
}

func TestCORS_HandlesPreflight(t *testing.T) {
	r := gin.New()
	r.Use(middleware.CORS("http://localhost:3000"))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("expected 204 for OPTIONS, got %d", w.Code)
	}
}

func TestCORS_AllowsCredentials(t *testing.T) {
	r := gin.New()
	r.Use(middleware.CORS("http://localhost:3000"))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials=true, got %s", got)
	}
}

func TestCORS_AllowsMethods(t *testing.T) {
	r := gin.New()
	r.Use(middleware.CORS("http://localhost:3000"))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	allowedMethods := w.Header().Get("Access-Control-Allow-Methods")
	if allowedMethods == "" {
		t.Error("expected Access-Control-Allow-Methods to be set")
	}
}
