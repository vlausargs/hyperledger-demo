package handler

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	hardcodedUser = "admin"
	hardcodedPass = "asdqwe123"
	tokenTTL      = 24 * time.Hour
)

// Login issues a JWT token for valid credentials.
func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server misconfigured: JWT_SECRET not set"})
			return
		}

		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "username and password required"})
			return
		}

		if req.Username != hardcodedUser || req.Password != hardcodedPass {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": req.Username,
			"exp": time.Now().Add(tokenTTL).Unix(),
			"iat": time.Now().Unix(),
		})

		signed, err := token.SignedString([]byte(secret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":      signed,
			"expires_in": int(tokenTTL.Seconds()),
		})
	}
}
