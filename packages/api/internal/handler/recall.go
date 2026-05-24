package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func IssueRecall(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req IssueRecallRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		targetJSON, err := jsonMarshal(req.TargetIDs)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid targetIds"})
			return
		}
		cid, _ := c.Get("correlationID")
		_, err = gw.SubmitTransaction("IssueRecall",
			req.ID, req.Scope, targetJSON, req.Reason,
			req.Severity, req.IssuedByName, req.InstructionsURL)
		if err != nil {
			slog.Error("IssueRecall failed", "id", req.ID, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue recall"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "recall issued", "id": req.ID})
	}
}

func ReadRecall(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateID("recall", id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cid, _ := c.Get("correlationID")
		result, err := evalJSON(gw, "ReadRecall", id)
		if err != nil {
			slog.Error("ReadRecall failed", "id", id, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": "recall not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve recall"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func GetRecalledProducts(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := c.Get("correlationID")
		result, err := evalJSON(gw, "GetRecalledProducts")
		if err != nil {
			slog.Error("GetRecalledProducts failed", "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve recalled products"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"products": result})
	}
}
