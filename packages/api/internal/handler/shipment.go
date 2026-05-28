package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/service"
	"github.com/myindo/hlf-supply-chain/api/pkg/response"
)

// ShipmentHandler routes shipment + custody HTTP endpoints.
type ShipmentHandler struct {
	svc     *service.ShipmentService
	custody *service.CustodyService
}

func NewShipmentHandler(svc *service.ShipmentService, custody *service.CustodyService) *ShipmentHandler {
	return &ShipmentHandler{svc: svc, custody: custody}
}

func (h *ShipmentHandler) List(c *gin.Context) {
	ps, bm := paged(c)
	result, err := h.svc.List(c.Request.Context(), ps, bm)
	if err != nil {
		slog.Error("ShipmentHandler.List failed", "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", result)
}

func (h *ShipmentHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("shipment", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	result, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		slog.Error("ShipmentHandler.Get failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", result)
}

func (h *ShipmentHandler) Create(c *gin.Context) {
	var req CreateShipmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	res, err := h.svc.Create(c.Request.Context(), service.CreateShipmentInput(req))
	if err != nil {
		slog.Error("ShipmentHandler.Create failed", "id", req.ID, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.Created(c, gin.H{"message": "shipment created", "id": res.ID})
}

func (h *ShipmentHandler) Dispatch(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("shipment", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	res, err := h.svc.Dispatch(c.Request.Context(), id)
	if err != nil {
		slog.Error("ShipmentHandler.Dispatch failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "shipment dispatched", "id": res.ID})
}

func (h *ShipmentHandler) History(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("shipment", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	result, err := h.svc.History(c.Request.Context(), id)
	if err != nil {
		slog.Error("ShipmentHandler.History failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"id": id, "history": result})
}

func (h *ShipmentHandler) ByStatus(c *gin.Context) {
	status := c.Param("status")
	if status == "" {
		response.BadRequest(c, "status is required")
		return
	}
	result, err := h.svc.ByStatus(c.Request.Context(), status)
	if err != nil {
		slog.Error("ShipmentHandler.ByStatus failed", "status", status, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"status": status, "shipments": result})
}

func (h *ShipmentHandler) CustodyChain(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("shipment", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	result, err := h.custody.Chain(c.Request.Context(), id)
	if err != nil {
		slog.Error("ShipmentHandler.CustodyChain failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"shipmentId": id, "custody": result})
}

func (h *ShipmentHandler) InitiateCustody(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("shipment", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	var req InitiateCustodyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	in := service.InitiateCustodyInput{ShipmentID: id, ToMSP: req.ToMSP, ToName: req.ToName, Conditions: req.Conditions}
	if err := h.custody.Initiate(c.Request.Context(), in); err != nil {
		slog.Error("ShipmentHandler.InitiateCustody failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "custody transfer initiated", "shipmentId": id})
}

func (h *ShipmentHandler) AcceptCustody(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("shipment", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	if err := h.custody.Accept(c.Request.Context(), id); err != nil {
		slog.Error("ShipmentHandler.AcceptCustody failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "custody transfer accepted", "shipmentId": id})
}

func (h *ShipmentHandler) RejectCustody(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("shipment", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	if err := h.custody.Reject(c.Request.Context(), id); err != nil {
		slog.Error("ShipmentHandler.RejectCustody failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "custody transfer rejected", "shipmentId": id})
}
