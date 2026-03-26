package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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
}

// HealthCheck returns the health status of the API
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "Fabric Client API",
		"version": "1.0.0",
	})
}

// GetAllAssets retrieves all assets from the ledger
func GetAllAssets(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := gw.EvaluateTransaction("GetAllAssets")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to get all assets: %v", err),
			})
			return
		}

		var assets []Asset
		// Handle empty result gracefully
		if len(result) == 0 {
			assets = []Asset{}
		} else if err := json.Unmarshal(result, &assets); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to unmarshal assets: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"assets": assets,
			"count":  len(assets),
		})
	}
}

// GetAsset retrieves a specific asset by ID
func GetAsset(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		result, err := gw.EvaluateTransaction("ReadAsset", id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": fmt.Sprintf("Failed to read asset: %v", err),
			})
			return
		}

		var asset Asset
		if err := json.Unmarshal(result, &asset); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to unmarshal asset: %v", err),
			})
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
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid request: %v", err),
			})
			return
		}

		_, err := gw.SubmitTransaction("CreateAsset",
			req.ID,
			req.Color,
			strconv.Itoa(req.Size),
			req.Owner,
			strconv.Itoa(req.AppraisedValue),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to create asset: %v", err),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Asset created successfully",
			"assetID": req.ID,
		})
	}
}

// UpdateAsset updates an existing asset on the ledger
func UpdateAsset(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req UpdateAssetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid request: %v", err),
			})
			return
		}

		_, err := gw.SubmitTransaction("UpdateAsset",
			id,
			req.Color,
			strconv.Itoa(req.Size),
			req.Owner,
			strconv.Itoa(req.AppraisedValue),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to update asset: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Asset updated successfully",
			"assetID": id,
		})
	}
}

// DeleteAsset deletes an asset from the ledger
func DeleteAsset(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		_, err := gw.SubmitTransaction("DeleteAsset", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to delete asset: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Asset deleted successfully",
			"assetID": id,
		})
	}
}

// TransferAsset transfers ownership of an asset
func TransferAsset(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req TransferAssetRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid request: %v", err),
			})
			return
		}

		_, err := gw.SubmitTransaction("TransferAsset", id, req.NewOwner)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to transfer asset: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  "Asset transferred successfully",
			"assetID":  id,
			"newOwner": req.NewOwner,
		})
	}
}

// GetAssetHistory retrieves the history of an asset
func GetAssetHistory(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		result, err := gw.EvaluateTransaction("GetAssetHistory", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to get asset history: %v", err),
			})
			return
		}

		var history []AssetHistory
		// Handle empty result gracefully
		if len(result) == 0 {
			history = []AssetHistory{}
		} else if err := json.Unmarshal(result, &history); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to unmarshal asset history: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"assetID": id,
			"history": history,
		})
	}
}

// GetAssetsByRange retrieves assets within a specified ID range
func GetAssetsByRange(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		startKey := c.DefaultQuery("startKey", "")
		endKey := c.DefaultQuery("endKey", "")

		result, err := gw.EvaluateTransaction("GetAssetByRange", startKey, endKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to get assets by range: %v", err),
			})
			return
		}

		var assets []Asset
		// Handle empty result gracefully
		if len(result) == 0 {
			assets = []Asset{}
		} else if err := json.Unmarshal(result, &assets); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to unmarshal assets: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"assets": assets,
			"count":  len(assets),
			"range": gin.H{
				"start": startKey,
				"end":   endKey,
			},
		})
	}
}

// GetChannels retrieves information about available channels
func GetChannels(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is a placeholder implementation
		// In a real implementation, you would query the network for available channels
		c.JSON(http.StatusOK, gin.H{
			"channels": []gin.H{
				{
					"channel_id": gw.GetChannel(),
					"status":     "active",
				},
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

// GetPeers retrieves information about network peers
func GetPeers(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is a placeholder implementation
		// In a real implementation, you would query the network for peer information
		c.JSON(http.StatusOK, gin.H{
			"peers": []gin.H{
				{
					"name":    "peer0.org1.example.com",
					"address": "peer0.org1.example.com:7051",
					"status":  "online",
					"org":     "Org1",
				},
				{
					"name":    "peer0.org2.example.com",
					"address": "peer0.org2.example.com:9051",
					"status":  "online",
					"org":     "Org2",
				},
			},
		})
	}
}

// GetOrganizations retrieves information about network organizations
func GetOrganizations(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is a placeholder implementation
		// In a real implementation, you would query the network for organization information
		c.JSON(http.StatusOK, gin.H{
			"organizations": []gin.H{
				{
					"name":  "Org1MSP",
					"mspid": "Org1MSP",
					"peers": []string{"peer0.org1.example.com"},
				},
				{
					"name":  "Org2MSP",
					"mspid": "Org2MSP",
					"peers": []string{"peer0.org2.example.com"},
				},
			},
		})
	}
}

// GetTransactions retrieves transaction information
func GetTransactions(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is a placeholder implementation
		// In a real implementation, you would query the ledger for transaction history
		c.JSON(http.StatusOK, gin.H{
			"transactions": []gin.H{},
			"count":        0,
		})
	}
}

// GetTransaction retrieves a specific transaction by ID
func GetTransaction(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		txID := c.Param("txId")

		// This is a placeholder implementation
		// In a real implementation, you would query the ledger for transaction details
		c.JSON(http.StatusOK, gin.H{
			"txId":      txID,
			"status":    "VALID",
			"timestamp": "2023-01-01T00:00:00Z",
		})
	}
}
