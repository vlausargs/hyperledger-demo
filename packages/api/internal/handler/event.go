package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/service"
	"github.com/myindo/hlf-supply-chain/api/pkg/response"
)

// EventHandler routes event-log HTTP endpoints.
type EventHandler struct {
	svc *service.EventService
}

func NewEventHandler(svc *service.EventService) *EventHandler {
	return &EventHandler{svc: svc}
}

func (h *EventHandler) Log(c *gin.Context) {
	var req LogEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	res, err := h.svc.Log(c.Request.Context(), service.LogEventInput(req))
	if err != nil {
		slog.Error("EventHandler.Log failed", "id", req.ID, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.Created(c, gin.H{"message": "event logged", "id": res.ID})
}

func (h *EventHandler) List(c *gin.Context) {
	targetID := c.Param("targetId")
	if vErr := validateID("target", targetID); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	result, err := h.svc.List(c.Request.Context(), targetID)
	if err != nil {
		slog.Error("EventHandler.List failed", "targetId", targetID, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"targetId": targetID, "events": result})
}
