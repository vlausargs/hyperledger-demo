package service

import (
	"context"
	"encoding/json"
	"fmt"

	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// CreateProductInput is the typed payload for ProductService.Create.
type CreateProductInput struct {
	ID               string
	SKU              string
	Name             string
	Description      string
	BatchID          string
	ManufacturerName string
	ExpiryDate       string
	Metadata         map[string]string
}

// UpdateProductInput is the typed payload for ProductService.Update.
type UpdateProductInput struct {
	Name        string
	Description string
	ExpiryDate  string
	Metadata    map[string]string
}

// CreateResult is the small ack returned by submit-style writes.
type CreateResult struct {
	ID string `json:"id"`
}

// ProductService handles product-domain operations on the ledger.
type ProductService struct {
	gw FabricGateway
}

func NewProductService(gw FabricGateway) *ProductService {
	return &ProductService{gw: gw}
}

// Create writes a new product to the ledger. Returns ErrConflict if the
// product ID already exists.
func (s *ProductService) Create(_ context.Context, in CreateProductInput) (*CreateResult, *apperrors.AppError) {
	metaJSON, err := marshalMap(in.Metadata, "{}")
	if err != nil {
		return nil, apperrors.NewBadRequest("invalid metadata: " + err.Error())
	}
	if _, err := s.gw.SubmitTransaction("CreateProduct",
		in.ID, in.SKU, in.Name, in.Description,
		in.BatchID, in.ManufacturerName, in.ExpiryDate, metaJSON); err != nil {
		return nil, mapFabricError(err, "product")
	}
	return &CreateResult{ID: in.ID}, nil
}

// Update edits an existing product. Returns ErrNotFound if the ID is unknown.
func (s *ProductService) Update(_ context.Context, id string, in UpdateProductInput) (*CreateResult, *apperrors.AppError) {
	metaJSON, err := marshalMap(in.Metadata, "")
	if err != nil {
		return nil, apperrors.NewBadRequest("invalid metadata: " + err.Error())
	}
	if _, err := s.gw.SubmitTransaction("UpdateProduct", id, in.Name, in.Description, in.ExpiryDate, metaJSON); err != nil {
		return nil, mapFabricError(err, "product")
	}
	return &CreateResult{ID: id}, nil
}

// Get returns the raw product JSON from the ledger.
func (s *ProductService) Get(_ context.Context, id string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "product", "ReadProduct", id)
}

// List returns all products with paging.
func (s *ProductService) List(_ context.Context, pageSize, bookmark string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "products", "GetAllProducts", pageSize, bookmark)
}

// History returns the immutable change history for a product.
func (s *ProductService) History(_ context.Context, id string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "product history", "GetProductHistory", id)
}

// Provenance returns the upstream supply-chain trace.
func (s *ProductService) Provenance(_ context.Context, id string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "product", "GetProvenance", id)
}

// ByBatch lists products that belong to a batch.
func (s *ProductService) ByBatch(_ context.Context, batchID string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "products", "GetProductsByBatch", batchID)
}

// ByStatus lists products in a given lifecycle status.
func (s *ProductService) ByStatus(_ context.Context, status string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "products", "GetProductsByStatus", status)
}

// ── shared helpers (kept in this file so each domain file is self-contained
//    when read in isolation, but only declared once) ───────────────────────

// marshalMap encodes a string map to JSON, returning fallback when nil.
func marshalMap(m map[string]string, fallback string) (string, error) {
	if m == nil {
		return fallback, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("marshal failed: %w", err)
	}
	return string(b), nil
}

// evalRaw runs an Evaluate query and returns the raw JSON, mapping errors.
// An empty result becomes JSON "null" so handlers can always render it.
func evalRaw(gw FabricGateway, resource, fn string, args ...string) (json.RawMessage, *apperrors.AppError) {
	b, err := gw.EvaluateTransaction(fn, args...)
	if err != nil {
		return nil, mapFabricError(err, resource)
	}
	if len(b) == 0 {
		return json.RawMessage("null"), nil
	}
	return b, nil
}
