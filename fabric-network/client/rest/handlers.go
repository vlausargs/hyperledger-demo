package rest

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"hlf-demo/fabric-network/client/fabric"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-gateway/pkg/client"
)

// FabricGateway interface for interacting with Fabric
type FabricGateway interface {
	GetContract() *client.Contract
	SubmitTransaction(function string, args ...string) ([]byte, error)
	EvaluateTransaction(function string, args ...string) ([]byte, error)
	GetChannel() string
	GetChaincode() string
	GetConnectionProfile() *fabric.ConnectionProfile
}

// ── Health ────────────────────────────────────────────────────────────────────

// HealthCheck verifies ledger connectivity.
func HealthCheck(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := gw.EvaluateTransaction("GetAllProducts", "1", "")
		if err != nil {
			slog.Error("health check failed", "error", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "unhealthy",
				"service": "Fabric Client API",
				"error":   "ledger unreachable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "Fabric Client API",
			"version": "2.0.0",
		})
	}
}

// ── Product request types ─────────────────────────────────────────────────────

type CreateProductRequest struct {
	ID               string            `json:"id"               binding:"required"`
	SKU              string            `json:"sku"              binding:"required"`
	Name             string            `json:"name"             binding:"required"`
	Description      string            `json:"description"`
	BatchID          string            `json:"batchId"          binding:"required"`
	ManufacturerName string            `json:"manufacturerName" binding:"required"`
	ExpiryDate       string            `json:"expiryDate"`
	Metadata         map[string]string `json:"metadata"`
}

type UpdateProductRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	ExpiryDate  string            `json:"expiryDate"`
	Metadata    map[string]string `json:"metadata"`
}

// ── Shipment request types ────────────────────────────────────────────────────

type CreateShipmentRequest struct {
	ID           string   `json:"id"           binding:"required"`
	Name         string   `json:"name"         binding:"required"`
	Description  string   `json:"description"`
	ReceiverMSP  string   `json:"receiverMsp"  binding:"required"`
	ReceiverName string   `json:"receiverName" binding:"required"`
	Origin       string   `json:"origin"`
	Destination  string   `json:"destination"`
	ProductIDs   []string `json:"productIds"   binding:"required"`
}

type InitiateCustodyRequest struct {
	ToMSP      string `json:"toMsp"      binding:"required"`
	ToName     string `json:"toName"     binding:"required"`
	Conditions string `json:"conditions"`
}

// ── Event request types ───────────────────────────────────────────────────────

type LogEventRequest struct {
	ID          string            `json:"id"          binding:"required"`
	TargetID    string            `json:"targetId"    binding:"required"`
	TargetType  string            `json:"targetType"  binding:"required"`
	EventType   string            `json:"eventType"   binding:"required"`
	Description string            `json:"description"`
	Location    string            `json:"location"`
	OccurredAt  string            `json:"occurredAt"`
	Data        map[string]string `json:"data"`
}

// ── Recall request types ──────────────────────────────────────────────────────

type IssueRecallRequest struct {
	ID              string   `json:"id"              binding:"required"`
	Scope           string   `json:"scope"           binding:"required"`
	TargetIDs       []string `json:"targetIds"       binding:"required"`
	Reason          string   `json:"reason"          binding:"required"`
	Severity        string   `json:"severity"        binding:"required"`
	IssuedByName    string   `json:"issuedByName"    binding:"required"`
	InstructionsURL string   `json:"instructionsUrl"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func evalJSON(gw FabricGateway, fn string, args ...string) (json.RawMessage, error) {
	b, err := gw.EvaluateTransaction(fn, args...)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return json.RawMessage("null"), nil
	}
	return b, nil
}

func paged(c *gin.Context) (string, string) {
	ps := c.DefaultQuery("pageSize", "20")
	bm := c.DefaultQuery("bookmark", "")
	n, err := strconv.Atoi(ps)
	if err != nil || n <= 0 {
		n = 20
	}
	if n > 100 {
		n = 100
	}
	return strconv.Itoa(n), bm
}

func jsonMarshal(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal failed: %w", err)
	}
	return string(b), nil
}

// ── Product handlers ──────────────────────────────────────────────────────────

func GetAllProducts(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := c.Get("correlationID")
		ps, bm := paged(c)
		result, err := evalJSON(gw, "GetAllProducts", ps, bm)
		if err != nil {
			slog.Error("GetAllProducts failed", "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve products"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func GetProduct(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		cid, _ := c.Get("correlationID")
		result, err := evalJSON(gw, "ReadProduct", id)
		if err != nil {
			slog.Error("GetProduct failed", "id", id, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve product"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func CreateProduct(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateProductRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		metaJSON := "{}"
		if req.Metadata != nil {
			if s, err := jsonMarshal(req.Metadata); err == nil {
				metaJSON = s
			}
		}
		cid, _ := c.Get("correlationID")
		_, err := gw.SubmitTransaction("CreateProduct",
			req.ID, req.SKU, req.Name, req.Description,
			req.BatchID, req.ManufacturerName, req.ExpiryDate, metaJSON)
		if err != nil {
			slog.Error("CreateProduct failed", "id", req.ID, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "already exists") {
				c.JSON(http.StatusConflict, gin.H{"error": "product already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "product created", "id": req.ID})
	}
}

func UpdateProduct(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req UpdateProductRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		metaJSON := ""
		if req.Metadata != nil {
			if s, err := jsonMarshal(req.Metadata); err == nil {
				metaJSON = s
			}
		}
		cid, _ := c.Get("correlationID")
		_, err := gw.SubmitTransaction("UpdateProduct", id, req.Name, req.Description, req.ExpiryDate, metaJSON)
		if err != nil {
			slog.Error("UpdateProduct failed", "id", id, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update product"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "product updated", "id": id})
	}
}

func GetProductHistory(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		cid, _ := c.Get("correlationID")
		result, err := evalJSON(gw, "GetProductHistory", id)
		if err != nil {
			slog.Error("GetProductHistory failed", "id", id, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve product history"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id, "history": result})
	}
}

func GetProductProvenance(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		cid, _ := c.Get("correlationID")
		result, err := evalJSON(gw, "GetProvenance", id)
		if err != nil {
			slog.Error("GetProductProvenance failed", "id", id, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve provenance"})
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func GetProductsByBatch(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		batchID := c.Param("batchId")
		cid, _ := c.Get("correlationID")
		result, err := evalJSON(gw, "GetProductsByBatch", batchID)
		if err != nil {
			slog.Error("GetProductsByBatch failed", "batchId", batchID, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve products by batch"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"batchId": batchID, "products": result})
	}
}

func GetProductsByStatus(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Param("status")
		cid, _ := c.Get("correlationID")
		result, err := evalJSON(gw, "GetProductsByStatus", status)
		if err != nil {
			slog.Error("GetProductsByStatus failed", "status", status, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve products by status"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": status, "products": result})
	}
}

// ── Shipment handlers ─────────────────────────────────────────────────────────

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

// ── Event handlers ────────────────────────────────────────────────────────────

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

// ── Recall handlers ───────────────────────────────────────────────────────────

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

// ── Infrastructure handlers (unchanged) ───────────────────────────────────────

func GetChannels(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"channels": []gin.H{
				{"channel_id": gw.GetChannel(), "status": "active"},
			},
		})
	}
}

func GetChannelInfo(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelID := c.Param("channelId")
		if channelID != gw.GetChannel() {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"channel_id": channelID,
			"status":     "active",
			"chaincode":  gw.GetChaincode(),
		})
	}
}

func GetChaincodes(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"chaincodes": []gin.H{
				{
					"name":    gw.GetChaincode(),
					"version": "3.0",
					"channel": gw.GetChannel(),
					"status":  "active",
				},
			},
		})
	}
}

func GetChaincodeInfo(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		chaincodeID := c.Param("chaincodeId")
		if chaincodeID != gw.GetChaincode() {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chaincode not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"name":     chaincodeID,
			"version":  "3.0",
			"channel":  gw.GetChannel(),
			"status":   "active",
			"language": "golang",
			"domain":   "supply-chain",
		})
	}
}

func GetPeers(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		cp := gw.GetConnectionProfile()
		if cp == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection profile not loaded"})
			return
		}
		var peers []gin.H
		for name, peer := range cp.Peers {
			peers = append(peers, gin.H{"name": name, "address": peer.URL})
		}
		c.JSON(http.StatusOK, gin.H{"peers": peers})
	}
}

func GetOrganizations(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		cp := gw.GetConnectionProfile()
		if cp == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection profile not loaded"})
			return
		}
		var orgs []gin.H
		for name, org := range cp.Organizations {
			orgs = append(orgs, gin.H{"name": name, "mspid": org.MSPID, "peers": org.Peers})
		}
		c.JSON(http.StatusOK, gin.H{"organizations": orgs})
	}
}

func GetTransactions(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "transaction history query not supported by Fabric Gateway SDK"})
	}
}

func GetTransaction(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "transaction lookup not supported by Fabric Gateway SDK"})
	}
}
