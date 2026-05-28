package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/service"
	"github.com/myindo/hlf-supply-chain/api/pkg/response"
)

// RecallHandler routes recall HTTP endpoints.
type RecallHandler struct {
	svc *service.RecallService
}

func NewRecallHandler(svc *service.RecallService) *RecallHandler {
	return &RecallHandler{svc: svc}
}

func (h *RecallHandler) Issue(c *gin.Context) {
	var req IssueRecallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	res, err := h.svc.Issue(c.Request.Context(), service.IssueRecallInput(req))
	if err != nil {
		slog.Error("RecallHandler.Issue failed", "id", req.ID, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.Created(c, gin.H{"message": "recall issued", "id": res.ID})
}

func (h *RecallHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("recall", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	result, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		slog.Error("RecallHandler.Get failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", result)
}

func (h *RecallHandler) RecalledProducts(c *gin.Context) {
	result, err := h.svc.RecalledProducts(c.Request.Context())
	if err != nil {
		slog.Error("RecallHandler.RecalledProducts failed", "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"products": result})
}
