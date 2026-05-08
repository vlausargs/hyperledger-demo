package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ── POS request types ─────────────────────────────────────────────────────────

type SaleItemRequest struct {
	ProductID   string  `json:"productId"   binding:"required"`
	SKU         string  `json:"sku"`
	ProductName string  `json:"productName"`
	UnitPrice   float64 `json:"unitPrice"   binding:"required"`
}

type CreateSaleRequest struct {
	ID          string            `json:"id"          binding:"required"`
	CustomerID  string            `json:"customerId"  binding:"required"`
	CashierID   string            `json:"cashierId"   binding:"required"`
	CashierName string            `json:"cashierName" binding:"required"`
	Items       []SaleItemRequest `json:"items"       binding:"required,min=1"`
	TaxAmount   float64           `json:"taxAmount"`
	Currency    string            `json:"currency"`
	Notes       string            `json:"notes"`
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func CreateSale(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateSaleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validateSaleID(req.ID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		itemsBytes, err := json.Marshal(req.Items)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal items"})
			return
		}

		taxAmountStr := fmt.Sprintf("%f", req.TaxAmount)
		if req.Currency == "" {
			req.Currency = "IDR"
		}

		result, err := gw.SubmitTransaction("CreateSale",
			req.ID, req.CustomerID, req.CashierID, req.CashierName,
			string(itemsBytes), taxAmountStr, req.Currency, req.Notes)
		if err != nil {
			status := http.StatusInternalServerError
			msg := err.Error()
			if strings.Contains(msg, "already exists") {
				status = http.StatusConflict
			} else if strings.Contains(msg, "does not exist") || strings.Contains(msg, "not DELIVERED") ||
				strings.Contains(msg, "not owned by") || strings.Contains(msg, "under recall") {
				status = http.StatusBadRequest
			} else if strings.Contains(msg, "not authorized") {
				status = http.StatusForbidden
			}
			c.JSON(status, gin.H{"error": msg})
			return
		}

		_ = result
		c.JSON(http.StatusCreated, gin.H{"id": req.ID, "status": "created"})
	}
}

func GetAllSales(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID := c.Query("customerId")
		if customerID != "" {
			data, err := evalJSON(gw, "GetSalesByCustomer", customerID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.Data(http.StatusOK, "application/json", data)
			return
		}

		ps, bm := paged(c)
		data, err := evalJSON(gw, "GetAllSales", ps, bm)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/json", data)
	}
}

func GetSale(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateSaleID(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		data, err := evalJSON(gw, "ReadSale", id)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("sale %s not found", id)})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/json", data)
	}
}

func GetInventory(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		ownerMSP := c.DefaultQuery("ownerMsp", "Org3MSP")
		data, err := evalJSON(gw, "GetInventory", ownerMSP)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/json", data)
	}
}

func VerifyProduct(gw FabricGateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := validateProductID(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		data, err := evalJSON(gw, "GetProvenance", id)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("product %s not found", id)})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/json", data)
	}
}
