package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/service"
	"github.com/myindo/hlf-supply-chain/api/pkg/response"
)

// AuthHandler issues JWTs via AuthService.
type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "username and password required")
		return
	}
	res, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}
