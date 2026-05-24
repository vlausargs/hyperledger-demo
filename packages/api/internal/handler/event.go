package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func LogEvent(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LogEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		dataJSON := "{}"
		if req.Data != nil {
			if s, err := jsonMarshal(req.Data); err == nil {
				dataJSON = s
			}
		}
		cid, _ := c.Get("correlationID")
		_, err := gw.SubmitTransaction("LogEvent",
			req.ID, req.TargetID, req.TargetType, req.EventType,
			req.Description, req.Location, req.OccurredAt, dataJSON)
		if err != nil {
			slog.Error("LogEvent failed", "id", req.ID, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to log event"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "event logged", "id": req.ID})
	}
}

func GetEvents(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetID := c.Param("targetId")
		cid, _ := c.Get("correlationID")
		result, err := evalJSON(gw, "GetEvents", targetID)
		if err != nil {
			slog.Error("GetEvents failed", "targetId", targetID, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve events"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"targetId": targetID, "events": result})
	}
}
