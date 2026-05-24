package contracts

import (
	"encoding/json"
	"testing"

	"github.com/myindo/hlf-supply-chain/chaincode/models"
)

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestProductContract_ProductExists_False(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &ProductContract{}

	exists, err := contract.ProductExists(ctx, "prod-001")
	if err != nil {
		t.Fatalf("ProductExists returned unexpected error: %v", err)
	}
	if exists {
		t.Error("expected ProductExists to return false for missing product, got true")
	}
}

func TestProductContract_ProductExists_True(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &ProductContract{}

	// Manually insert a product into the mock state
	p := models.Product{
		DocType: "PRODUCT",
		ID:      "prod-001",
		SKU:     "SKU-1",
		Name:    "Test Product",
		Status:  models.ProductStatusActive,
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("failed to marshal product: %v", err)
	}
	if err := ctx.GetStub().PutState(models.PrefixProduct+"prod-001", data); err != nil {
		t.Fatalf("PutState failed: %v", err)
	}

	exists, err := contract.ProductExists(ctx, "prod-001")
	if err != nil {
		t.Fatalf("ProductExists returned unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected ProductExists to return true after inserting product, got false")
	}
}

func TestProductContract_ReadProduct_NotFound(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &ProductContract{}

	_, err := contract.ReadProduct(ctx, "nonexistent-product")
	if err == nil {
		t.Fatal("expected error when reading nonexistent product, got nil")
	}
}

func TestProductContract_GetAllProducts_Empty(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &ProductContract{}

	result, err := contract.GetAllProducts(ctx, 10, "")
	if err != nil {
		t.Fatalf("GetAllProducts returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result, got nil")
	}
	if len(result.Products) != 0 {
		t.Errorf("expected 0 products, got %d", len(result.Products))
	}
	if result.Count != 0 {
		t.Errorf("expected count 0, got %d", result.Count)
	}
}

func TestProductContract_GetAllProducts_WithProducts(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &ProductContract{}

	// Insert 2 products into mock state
	products := []models.Product{
		{
			DocType: "PRODUCT", ID: "prod-001", SKU: "SKU-A",
			Name: "Widget Alpha", Status: models.ProductStatusActive,
		},
		{
			DocType: "PRODUCT", ID: "prod-002", SKU: "SKU-B",
			Name: "Widget Beta", Status: models.ProductStatusActive,
		},
	}
	for _, p := range products {
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("failed to marshal product: %v", err)
		}
		if err := ctx.GetStub().PutState(models.PrefixProduct+p.ID, data); err != nil {
			t.Fatalf("PutState failed: %v", err)
		}
	}

	result, err := contract.GetAllProducts(ctx, 10, "")
	if err != nil {
		t.Fatalf("GetAllProducts returned unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result, got nil")
	}
	if len(result.Products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(result.Products))
	}
	if result.Count != 2 {
		t.Errorf("expected count 2, got %d", result.Count)
	}

	// Verify products are returned in key order (prod-001 before prod-002)
	if result.Products[0].ID != "prod-001" {
		t.Errorf("expected first product ID prod-001, got %s", result.Products[0].ID)
	}
	if result.Products[1].ID != "prod-002" {
		t.Errorf("expected second product ID prod-002, got %s", result.Products[1].ID)
	}
}
