package service

import (
	"context"
	"encoding/json"
	"fmt"

	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// SaleItemInput is one line on a sale.
type SaleItemInput struct {
	ProductID   string  `json:"productId"`
	SKU         string  `json:"sku,omitempty"`
	ProductName string  `json:"productName,omitempty"`
	UnitPrice   float64 `json:"unitPrice"`
}

// CreateSaleInput is the typed payload for SaleService.Create.
type CreateSaleInput struct {
	ID          string
	CustomerID  string
	CashierID   string
	CashierName string
	Items       []SaleItemInput
	TaxAmount   float64
	Currency    string
	Notes       string
}

// SaleService handles point-of-sale transactions.
type SaleService struct {
	gw FabricGateway
}

func NewSaleService(gw FabricGateway) *SaleService {
	return &SaleService{gw: gw}
}

// Create records a sale, transferring product ownership to the customer.
func (s *SaleService) Create(_ context.Context, in CreateSaleInput) (*CreateResult, *apperrors.AppError) {
	itemsBytes, err := json.Marshal(in.Items)
	if err != nil {
		return nil, apperrors.NewInternal("failed to marshal items", err.Error())
	}
	currency := in.Currency
	if currency == "" {
		currency = "IDR"
	}
	taxStr := fmt.Sprintf("%f", in.TaxAmount)
	if _, err := s.gw.SubmitTransaction("CreateSale",
		in.ID, in.CustomerID, in.CashierID, in.CashierName,
		string(itemsBytes), taxStr, currency, in.Notes); err != nil {
		return nil, mapFabricError(err, "sale")
	}
	return &CreateResult{ID: in.ID}, nil
}

// Get returns a single sale by ID.
func (s *SaleService) Get(_ context.Context, id string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "sale", "ReadSale", id)
}

// List returns all sales with paging.
func (s *SaleService) List(_ context.Context, pageSize, bookmark string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "sales", "GetAllSales", pageSize, bookmark)
}

// ListByCustomer returns sales filtered by customer.
func (s *SaleService) ListByCustomer(_ context.Context, customerID string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "sales", "GetSalesByCustomer", customerID)
}
