package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/myindo/hlf-supply-chain/api/internal/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

func TestJWT_NoHeader(t *testing.T) {
	r := gin.New()
	r.Use(middleware.JWT("test-secret"))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWT_InvalidToken(t *testing.T) {
	r := gin.New()
	r.Use(middleware.JWT("test-secret"))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWT_ValidToken(t *testing.T) {
	secret := "test-secret-key-for-testing"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "admin",
		"org":  "Org1MSP",
		"role": "admin",
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(secret))

	r := gin.New()
	r.Use(middleware.JWT(secret))
	r.GET("/test", func(c *gin.Context) {
		sub, _ := c.Get("sub")
		org, _ := c.Get("org")
		c.JSON(200, gin.H{"sub": sub, "org": org})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestJWT_ExpiredToken(t *testing.T) {
	secret := "test-secret-key-for-testing"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "admin",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte(secret))

	r := gin.New()
	r.Use(middleware.JWT(secret))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestJWT_MissingSecret(t *testing.T) {
	r := gin.New()
	r.Use(middleware.JWT(""))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestJWT_MalformedAuthHeader(t *testing.T) {
	r := gin.New()
	r.Use(middleware.JWT("test-secret"))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "NotBearer token")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
