package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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
		if err := validateID("product", id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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
		if err := validateID("product", id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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
		if err := validateID("product", id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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
		if err := validateID("product", id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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
		if err := validateID("batch", batchID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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
		if status == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
			return
		}
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
