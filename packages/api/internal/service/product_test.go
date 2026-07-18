package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestProductService_Create_HappyPath(t *testing.T) {
	var gotFn string
	var gotArgs []string
	gw := &mockGateway{
		submitFn: func(fn string, args ...string) ([]byte, error) {
			gotFn = fn
			gotArgs = args
			return nil, nil
		},
	}
	svc := NewProductService(gw)
	res, err := svc.Create(context.Background(), CreateProductInput{
		ID:               "p1",
		SKU:              "SKU1",
		Name:             "Widget",
		BatchID:          "B1",
		ManufacturerName: "ACME",
		Metadata:         map[string]string{"k": "v"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "p1" {
		t.Errorf("want id=p1, got %s", res.ID)
	}
	if gotFn != "ProductContract:CreateProduct" {
		t.Errorf("want fn=ProductContract:CreateProduct, got %s", gotFn)
	}
	if len(gotArgs) != 8 || gotArgs[0] != "p1" {
		t.Errorf("unexpected args: %#v", gotArgs)
	}
	if gotArgs[7] != `{"k":"v"}` {
		t.Errorf("want metadata JSON, got %q", gotArgs[7])
	}
}

func TestProductService_Create_NilMetadataDefaultsToEmptyObject(t *testing.T) {
	var gotArgs []string
	gw := &mockGateway{
		submitFn: func(fn string, args ...string) ([]byte, error) {
			gotArgs = args
			return nil, nil
		},
	}
	svc := NewProductService(gw)
	_, err := svc.Create(context.Background(), CreateProductInput{ID: "p2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotArgs[7] != "{}" {
		t.Errorf("want '{}' fallback, got %q", gotArgs[7])
	}
}

func TestProductService_Create_ConflictMapping(t *testing.T) {
	gw := &mockGateway{
		submitFn: func(fn string, args ...string) ([]byte, error) {
			return nil, errors.New("the asset p1 already exists")
		},
	}
	svc := NewProductService(gw)
	_, err := svc.Create(context.Background(), CreateProductInput{ID: "p1"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Code != http.StatusConflict {
		t.Errorf("want 409, got %d", err.Code)
	}
}

func TestProductService_Get_NotFoundMapping(t *testing.T) {
	gw := &mockGateway{
		evaluateFn: func(fn string, args ...string) ([]byte, error) {
			return nil, errors.New("the asset p1 does not exist")
		},
	}
	svc := NewProductService(gw)
	_, err := svc.Get(context.Background(), "p1")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", err.Code)
	}
}

func TestProductService_Get_HappyPath(t *testing.T) {
	gw := &mockGateway{
		evaluateFn: func(fn string, args ...string) ([]byte, error) {
			return []byte(`{"id":"p1"}`), nil
		},
	}
	svc := NewProductService(gw)
	out, err := svc.Get(context.Background(), "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != `{"id":"p1"}` {
		t.Errorf("unexpected output: %s", string(out))
	}
}

func TestProductService_Get_EmptyResultBecomesNull(t *testing.T) {
	gw := &mockGateway{
		evaluateFn: func(fn string, args ...string) ([]byte, error) { return nil, nil },
	}
	svc := NewProductService(gw)
	out, err := svc.Get(context.Background(), "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "null" {
		t.Errorf("want 'null', got %s", string(out))
	}
}

func TestProductService_Update_ValidationMapping(t *testing.T) {
	gw := &mockGateway{
		submitFn: func(fn string, args ...string) ([]byte, error) {
			return nil, errors.New("invalid expiry date format")
		},
	}
	svc := NewProductService(gw)
	_, err := svc.Update(context.Background(), "p1", UpdateProductInput{ExpiryDate: "bad"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", err.Code)
	}
}
