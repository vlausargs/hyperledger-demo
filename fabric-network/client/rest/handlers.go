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

// Asset represents a blockchain asset
type Asset struct {
	ID             string `json:"ID"`
	Color          string `json:"color"`
	Size           int    `json:"size"`
	Owner          string `json:"owner"`
	AppraisedValue int    `json:"appraisedValue"`
	OwnerMSP       string `json:"ownerMSP,omitempty"`
}

// CreateAssetRequest represents the request to create a new asset
type CreateAssetRequest struct {
	ID             string `json:"ID" binding:"required"`
	Color          string `json:"color" binding:"required"`
	Size           int    `json:"size" binding:"required"`
	Owner          string `json:"owner" binding:"required"`
	AppraisedValue int    `json:"appraisedValue" binding:"required"`
}

// UpdateAssetRequest represents the request to update an existing asset
type UpdateAssetRequest struct {
	Color          string `json:"color" binding:"required"`
	Size           int    `json:"size" binding:"required"`
	Owner          string `json:"owner" binding:"required"`
	AppraisedValue int    `json:"appraisedValue" binding:"required"`
}

// TransferAssetRequest represents the request to transfer an asset
type TransferAssetRequest struct {
	NewOwner string `json:"newOwner" binding:"required"`
}

// AssetHistory represents the history of an asset
type AssetHistory struct {
	TxId      string      `json:"txId"`
	Timestamp string      `json:"timestamp"`
	IsDelete  bool        `json:"isDelete"`
	Record    interface{} `json:"record"`
}

// FabricGateway interface for interacting with Fabric
type FabricGateway interface {
	GetContract() *client.Contract
	SubmitTransaction(function string, args ...string) ([]byte, error)
	EvaluateTransaction(function string, args ...string) ([]byte, error)
	GetChannel() string
	GetChaincode() string
	GetConnectionProfile() *fabric.ConnectionProfile
}

// HealthCheck verifies ledger connectivity and returns health status.
func HealthCheck(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := gw.EvaluateTransaction("GetAllAssets")
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
			"version": "1.0.0",
		})
	}
}

// PagedResult mirrors the chaincode PagedQueryResult
type PagedResult struct {
	Assets   []Asset `json:"assets"`
	Bookmark string  `json:"bookmark"`
}

// GetAllAssets retrieves assets from the ledger with optional pagination.
// Query params: pageSize (default 20, max 100), bookmark (cursor from previous response).
func GetAllAssets(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := c.Get("correlationID")

		pageSize := c.DefaultQuery("pageSize", "20")
		bookmark := c.DefaultQuery("bookmark", "")

		pageSizeInt, err := strconv.Atoi(pageSize)
		if err != nil || pageSizeInt <= 0 {
			pageSizeInt = 20
		}
		if pageSizeInt > 100 {
			pageSizeInt = 100
		}

		result, err := gw.EvaluateTransaction("GetAllAssetsPaged", strconv.Itoa(pageSizeInt), bookmark)
		if err != nil {
			slog.Error("GetAllAssets failed", "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve assets"})
			return
		}

		var paged PagedResult
		if len(result) == 0 {
			paged = PagedResult{Assets: []Asset{}}
		} else if err := json.Unmarshal(result, &paged); err != nil {
			slog.Error("GetAllAssets unmarshal failed", "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse assets"})
			return
		}

		if paged.Assets == nil {
			paged.Assets = []Asset{}
		}

		c.JSON(http.StatusOK, gin.H{
			"assets":   paged.Assets,
			"count":    len(paged.Assets),
			"bookmark": paged.Bookmark,
		})
	}
}

// GetAsset retrieves a specific asset by ID
func GetAsset(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateAssetID(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cid, _ := c.Get("correlationID")
		result, err := gw.EvaluateTransaction("ReadAsset", id)
		if err != nil {
			slog.Error("GetAsset failed", "id", id, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve asset"})
			return
		}

		var asset Asset
		if err := json.Unmarshal(result, &asset); err != nil {
			slog.Error("GetAsset unmarshal failed", "id", id, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse asset"})
			return
		}

		c.JSON(http.StatusOK, asset)
	}
}

// CreateAsset creates a new asset on the ledger
func CreateAsset(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateAssetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
			return
		}
		if err := validateAssetID(req.ID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cid, _ := c.Get("correlationID")
		_, err := gw.SubmitTransaction("CreateAsset",
			req.ID,
			req.Color,
			strconv.Itoa(req.Size),
			req.Owner,
			strconv.Itoa(req.AppraisedValue),
		)
		if err != nil {
			slog.Error("CreateAsset failed", "id", req.ID, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "already exists") {
				c.JSON(http.StatusConflict, gin.H{"error": "asset already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create asset"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "asset created", "assetID": req.ID})
	}
}

// UpdateAsset updates an existing asset on the ledger
func UpdateAsset(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateAssetID(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var req UpdateAssetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
			return
		}

		cid, _ := c.Get("correlationID")
		_, err := gw.SubmitTransaction("UpdateAsset",
			id,
			req.Color,
			strconv.Itoa(req.Size),
			req.Owner,
			strconv.Itoa(req.AppraisedValue),
		)
		if err != nil {
			slog.Error("UpdateAsset failed", "id", id, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update asset"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "asset updated", "assetID": id})
	}
}

// DeleteAsset deletes an asset from the ledger
func DeleteAsset(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateAssetID(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cid, _ := c.Get("correlationID")
		_, err := gw.SubmitTransaction("DeleteAsset", id)
		if err != nil {
			slog.Error("DeleteAsset failed", "id", id, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete asset"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "asset deleted", "assetID": id})
	}
}

// TransferAsset transfers ownership of an asset
func TransferAsset(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateAssetID(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var req TransferAssetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
			return
		}

		cid, _ := c.Get("correlationID")
		_, err := gw.SubmitTransaction("TransferAsset", id, req.NewOwner)
		if err != nil {
			slog.Error("TransferAsset failed", "id", id, "error", err, "correlation_id", cid)
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": "asset not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to transfer asset"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "asset transferred", "assetID": id, "newOwner": req.NewOwner})
	}
}

// GetAssetHistory retrieves the history of an asset
func GetAssetHistory(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateAssetID(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cid, _ := c.Get("correlationID")
		result, err := gw.EvaluateTransaction("GetAssetHistory", id)
		if err != nil {
			slog.Error("GetAssetHistory failed", "id", id, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve asset history"})
			return
		}

		var history []AssetHistory
		if len(result) == 0 {
			history = []AssetHistory{}
		} else if err := json.Unmarshal(result, &history); err != nil {
			slog.Error("GetAssetHistory unmarshal failed", "id", id, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse asset history"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"assetID": id, "history": history})
	}
}

// GetAssetsByRange retrieves assets within a specified ID range
func GetAssetsByRange(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		startKey := c.DefaultQuery("startKey", "")
		endKey := c.DefaultQuery("endKey", "")

		if len(startKey) > 256 || len(endKey) > 256 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "startKey and endKey max 256 chars"})
			return
		}

		cid, _ := c.Get("correlationID")
		result, err := gw.EvaluateTransaction("GetAssetByRange", startKey, endKey)
		if err != nil {
			slog.Error("GetAssetsByRange failed", "start", startKey, "end", endKey, "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve assets by range"})
			return
		}

		var assets []Asset
		if len(result) == 0 {
			assets = []Asset{}
		} else if err := json.Unmarshal(result, &assets); err != nil {
			slog.Error("GetAssetsByRange unmarshal failed", "error", err, "correlation_id", cid)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse assets"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"assets": assets,
			"count":  len(assets),
			"range":  gin.H{"start": startKey, "end": endKey},
		})
	}
}

// GetChannels returns the channel this client is connected to.
func GetChannels(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"channels": []gin.H{
				{"channel_id": gw.GetChannel(), "status": "active"},
			},
		})
	}
}

// GetChannelInfo retrieves information about a specific channel
func GetChannelInfo(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelID := c.Param("channelId")

		if channelID != gw.GetChannel() {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Channel not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"channel_id": channelID,
			"status":     "active",
			"chaincode":  gw.GetChaincode(),
		})
	}
}

// GetChaincodes retrieves information about deployed chaincodes
func GetChaincodes(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"chaincodes": []gin.H{
				{
					"name":    gw.GetChaincode(),
					"version": "1.0",
					"channel": gw.GetChannel(),
					"status":  "active",
				},
			},
		})
	}
}

// GetChaincodeInfo retrieves information about a specific chaincode
func GetChaincodeInfo(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		chaincodeID := c.Param("chaincodeId")

		if chaincodeID != gw.GetChaincode() {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Chaincode not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"name":     chaincodeID,
			"version":  "1.0",
			"channel":  gw.GetChannel(),
			"status":   "active",
			"language": "golang",
		})
	}
}

// GetPeers retrieves peer information from the connection profile
func GetPeers(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		cp := gw.GetConnectionProfile()
		if cp == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection profile not loaded"})
			return
		}
		var peers []gin.H
		for name, peer := range cp.Peers {
			peers = append(peers, gin.H{
				"name":    name,
				"address": peer.URL,
			})
		}
		c.JSON(http.StatusOK, gin.H{"peers": peers})
	}
}

// GetOrganizations retrieves organization information from the connection profile
func GetOrganizations(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		cp := gw.GetConnectionProfile()
		if cp == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "connection profile not loaded"})
			return
		}
		var orgs []gin.H
		for name, org := range cp.Organizations {
			orgs = append(orgs, gin.H{
				"name":  name,
				"mspid": org.MSPID,
				"peers": org.Peers,
			})
		}
		c.JSON(http.StatusOK, gin.H{"organizations": orgs})
	}
}

// GetTransactions is not implemented — Fabric Gateway SDK does not expose raw TX query.
func GetTransactions(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "transaction history query not supported by Fabric Gateway SDK"})
	}
}

// GetTransaction is not implemented — Fabric Gateway SDK does not expose raw TX lookup.
func GetTransaction(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "transaction lookup not supported by Fabric Gateway SDK"})
	}
}
