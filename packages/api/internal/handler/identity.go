package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/service"
	"github.com/myindo/hlf-supply-chain/api/pkg/response"
)

// IdentityHandler routes fabric CA endpoints (register, enroll, list, ...).
type IdentityHandler struct {
	svc *service.IdentityService
}

func NewIdentityHandler(svc *service.IdentityService) *IdentityHandler {
	return &IdentityHandler{svc: svc}
}

func (h *IdentityHandler) List(c *gin.Context) {
	ids, err := h.svc.List(c.Request.Context())
	if err != nil {
		slog.Error("IdentityHandler.List failed", "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{
		"identities": ids,
		"count":      len(ids),
		"caName":     h.svc.CAName(),
	})
}

func (h *IdentityHandler) Get(c *gin.Context) {
	name := c.Param("id")
	id, err := h.svc.Get(c.Request.Context(), name)
	if err != nil {
		slog.Error("IdentityHandler.Get failed", "name", name, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, id)
}

func (h *IdentityHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	res, err := h.svc.Register(c.Request.Context(), service.RegisterIdentityInput(req))
	if err != nil {
		slog.Error("IdentityHandler.Register failed", "name", req.Name, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.Created(c, res)
}

func (h *IdentityHandler) Enroll(c *gin.Context) {
	var req EnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	in := service.EnrollIdentityInput{Name: req.Name, Secret: req.Secret, Force: c.Query("force") == "true"}
	res, err := h.svc.Enroll(c.Request.Context(), in)
	if err != nil {
		slog.Error("IdentityHandler.Enroll failed", "name", req.Name, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}

func (h *IdentityHandler) Delete(c *gin.Context) {
	name := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), name); err != nil {
		slog.Error("IdentityHandler.Delete failed", "name", name, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "identity removed", "name": name})
}
