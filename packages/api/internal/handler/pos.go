package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/service"
	"github.com/myindo/hlf-supply-chain/api/pkg/response"
)

// POSHandler routes retailer POS endpoints (sale creation, inventory,
// pre-checkout verification).
type POSHandler struct {
	sale *service.SaleService
	pos  *service.POSService
}

func NewPOSHandler(sale *service.SaleService, pos *service.POSService) *POSHandler {
	return &POSHandler{sale: sale, pos: pos}
}

func (h *POSHandler) CreateSale(c *gin.Context) {
	var req CreateSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if vErr := validateID("sale", req.ID); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	in := service.CreateSaleInput{
		ID:          req.ID,
		CustomerID:  req.CustomerID,
		CashierID:   req.CashierID,
		CashierName: req.CashierName,
		Items:       toServiceSaleItems(req.Items),
		TaxAmount:   req.TaxAmount,
		Currency:    req.Currency,
		Notes:       req.Notes,
	}
	res, err := h.sale.Create(c.Request.Context(), in)
	if err != nil {
		slog.Error("POSHandler.CreateSale failed", "id", req.ID, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	response.Created(c, gin.H{"id": res.ID, "status": "created"})
}

func (h *POSHandler) ListSales(c *gin.Context) {
	customerID := c.Query("customerId")
	if customerID != "" {
		data, err := h.sale.ListByCustomer(c.Request.Context(), customerID)
		if err != nil {
			slog.Error("POSHandler.ListSales(byCustomer) failed", "error", err.Detail, "correlation_id", c.GetString("correlationID"))
			response.Error(c, err)
			return
		}
		c.Data(http.StatusOK, "application/json", data)
		return
	}
	ps, bm := paged(c)
	data, err := h.sale.List(c.Request.Context(), ps, bm)
	if err != nil {
		slog.Error("POSHandler.ListSales failed", "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

func (h *POSHandler) GetSale(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("sale", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	data, err := h.sale.Get(c.Request.Context(), id)
	if err != nil {
		slog.Error("POSHandler.GetSale failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

func (h *POSHandler) Inventory(c *gin.Context) {
	ownerMSP := c.DefaultQuery("ownerMsp", "Org3MSP")
	data, err := h.pos.Inventory(c.Request.Context(), ownerMSP)
	if err != nil {
		slog.Error("POSHandler.Inventory failed", "ownerMsp", ownerMSP, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

func (h *POSHandler) VerifyProduct(c *gin.Context) {
	id := c.Param("id")
	if vErr := validateID("product", id); vErr != nil {
		response.BadRequest(c, vErr.Error())
		return
	}
	data, err := h.pos.VerifyProduct(c.Request.Context(), id)
	if err != nil {
		slog.Error("POSHandler.VerifyProduct failed", "id", id, "error", err.Detail, "correlation_id", c.GetString("correlationID"))
		response.Error(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

// toServiceSaleItems converts the handler-facing item type into the
// service-layer type. Field names match, so it's a straight copy.
func toServiceSaleItems(in []SaleItemRequest) []service.SaleItemInput {
	out := make([]service.SaleItemInput, len(in))
	for i, it := range in {
		out[i] = service.SaleItemInput(it)
	}
	return out
}
