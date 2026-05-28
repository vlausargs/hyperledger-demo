package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// Error writes an AppError as a JSON error response using its embedded
// HTTP status code. Detail is intentionally not exposed to the client; it
// should be logged by the caller before invoking this helper.
func Error(c *gin.Context, e *apperrors.AppError) {
	if e == nil {
		return
	}
	c.JSON(e.Code, gin.H{"error": e.Message})
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, gin.H{"error": msg})
}

func Conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, gin.H{"error": msg})
}

func InternalError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
}

func Unavailable(c *gin.Context, msg string) {
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": msg})
}
