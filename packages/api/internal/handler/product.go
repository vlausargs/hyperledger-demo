package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/service"
	"github.com/myindo/hlf-supply-chain/api/pkg/response"
)

// ProductHandler routes HTTP requests to ProductService. Handlers are thin:
// parse + validate input, call the service, render the result/error.
type ProductHandler struct {
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func (h *ProductHandler) List(c *gin.Context) {
	ps, bm := paged(c)
	result, err := h.svc.List(c.Request.Context(), ps, bm)
	if err != nil {
		slog.Error("ProductHandler.List failed", "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", result)
}

func (h *ProductHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("product", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	result, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		slog.Error("ProductHandler.Get failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", result)
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	res, err := h.svc.Create(c.Request.Context(), service.CreateProductInput(req))
	if err != nil {
		slog.Error("ProductHandler.Create failed", "id", req.ID, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.Created(c, gin.H{"message": "product created", "id": res.ID})
}

func (h *ProductHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("product", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	res, err := h.svc.Update(c.Request.Context(), id, service.UpdateProductInput(req))
	if err != nil {
		slog.Error("ProductHandler.Update failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"message": "product updated", "id": res.ID})
}

func (h *ProductHandler) History(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("product", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	result, err := h.svc.History(c.Request.Context(), id)
	if err != nil {
		slog.Error("ProductHandler.History failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"id": id, "history": result})
}

func (h *ProductHandler) Provenance(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("product", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	result, err := h.svc.Provenance(c.Request.Context(), id)
	if err != nil {
		slog.Error("ProductHandler.Provenance failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", result)
}

func (h *ProductHandler) ByBatch(c *gin.Context) {
	batchID := c.Param("batchId")
	if vErr := validateID("batch", batchID); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	result, err := h.svc.ByBatch(c.Request.Context(), batchID)
	if err != nil {
		slog.Error("ProductHandler.ByBatch failed", "batchId", batchID, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"batchId": batchID, "products": result})
}

func (h *ProductHandler) ByStatus(c *gin.Context) {
	status := c.Param("status")
	if status == "" {
		response.BadRequest(c, "status is required")
		return
	}
	result, err := h.svc.ByStatus(c.Request.Context(), status)
	if err != nil {
		slog.Error("ProductHandler.ByStatus failed", "status", status, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"status": status, "products": result})
}
