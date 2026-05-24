package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetAllShipments(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := c.Get("correlationID")
		ps, bm := paged(c)
		result, err := evalJSON(gw, "GetAllShipments", ps, bm)
		if err != nil {
			slog.Error("GetAllShipments failed", "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve shipments"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func GetShipment(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateID("shipment", id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cid, _ := c.Get("correlationID")
		result, err := evalJSON(gw, "ReadShipment", id)
		if err != nil {
			slog.Error("GetShipment failed", "id", id, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": "shipment not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve shipment"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func CreateShipment(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateShipmentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		pidJSON, err := jsonMarshal(req.ProductIDs)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid productIds"})
			return
		}
		cid, _ := c.Get("correlationID")
		_, err = gw.SubmitTransaction("CreateShipment",
			req.ID, req.Name, req.Description,
			req.ReceiverMSP, req.ReceiverName,
			req.Origin, req.Destination, pidJSON)
		if err != nil {
			slog.Error("CreateShipment failed", "id", req.ID, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "already exists") {
				c.JSON(http.StatusConflict, gin.H{"error": "shipment already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create shipment"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "shipment created", "id": req.ID})
	}
}

func DispatchShipment(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateID("shipment", id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cid, _ := c.Get("correlationID")
		_, err := gw.SubmitTransaction("DispatchShipment", id)
		if err != nil {
			slog.Error("DispatchShipment failed", "id", id, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": "shipment not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to dispatch shipment"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "shipment dispatched", "id": id})
	}
}

func GetShipmentHistory(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateID("shipment", id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cid, _ := c.Get("correlationID")
		result, err := evalJSON(gw, "GetShipmentHistory", id)
		if err != nil {
			slog.Error("GetShipmentHistory failed", "id", id, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve shipment history"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id, "history": result})
	}
}

func GetShipmentsByStatus(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Param("status")
		if status == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
			return
		}
		cid, _ := c.Get("correlationID")
		result, err := evalJSON(gw, "GetShipmentsByStatus", status)
		if err != nil {
			slog.Error("GetShipmentsByStatus failed", "status", status, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve shipments by status"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": status, "shipments": result})
	}
}

func GetCustodyChain(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateID("shipment", id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cid, _ := c.Get("correlationID")
		result, err := evalJSON(gw, "GetCustodyChain", id)
		if err != nil {
			slog.Error("GetCustodyChain failed", "id", id, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve custody chain"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"shipmentId": id, "custody": result})
	}
}

func InitiateCustodyTransfer(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateID("shipment", id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var req InitiateCustodyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cid, _ := c.Get("correlationID")
		_, err := gw.SubmitTransaction("InitiateCustodyTransfer", id, req.ToMSP, req.ToName, req.Conditions)
		if err != nil {
			slog.Error("InitiateCustodyTransfer failed", "id", id, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initiate custody transfer"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "custody transfer initiated", "shipmentId": id})
	}
}

func AcceptCustodyTransfer(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateID("shipment", id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cid, _ := c.Get("correlationID")
		_, err := gw.SubmitTransaction("AcceptCustodyTransfer", id)
		if err != nil {
			slog.Error("AcceptCustodyTransfer failed", "id", id, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to accept custody transfer"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "custody transfer accepted", "shipmentId": id})
	}
}

func RejectCustodyTransfer(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateID("shipment", id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cid, _ := c.Get("correlationID")
		_, err := gw.SubmitTransaction("RejectCustodyTransfer", id)
		if err != nil {
			slog.Error("RejectCustodyTransfer failed", "id", id, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject custody transfer"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "custody transfer rejected", "shipmentId": id})
	}
}
