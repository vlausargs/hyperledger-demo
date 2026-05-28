package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestSaleService_Create_HappyPath_DefaultsCurrency(t *testing.T) {
	var gotArgs []string
	gw := &mockGateway{
		submitFn: func(fn string, args ...string) ([]byte, error) {
			gotArgs = args
			return nil, nil
		},
	}
	svc := NewSaleService(gw)
	res, err := svc.Create(context.Background(), CreateSaleInput{
		ID:          "sale1",
		CustomerID:  "c1",
		CashierID:   "u1",
		CashierName: "Alice",
		Items: []SaleItemInput{
			{ProductID: "p1", UnitPrice: 9.99},
		},
		TaxAmount: 1.5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "sale1" {
		t.Errorf("want id=sale1, got %s", res.ID)
	}
	// args: id, customerId, cashierId, cashierName, itemsJSON, tax, currency, notes
	if gotArgs[6] != "IDR" {
		t.Errorf("want currency=IDR default, got %s", gotArgs[6])
	}
	if !strings.Contains(gotArgs[4], `"productId":"p1"`) {
		t.Errorf("want items JSON with productId, got %s", gotArgs[4])
	}
}

func TestSaleService_Create_ValidationMapping(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want int
	}{
		{"not delivered", "product p1 is not DELIVERED yet", http.StatusBadRequest},
		{"not owned", "product is not owned by Org3MSP", http.StatusBadRequest},
		{"under recall", "product p1 is under recall", http.StatusBadRequest},
		{"already exists", "sale sale1 already exists", http.StatusConflict},
		{"not authorized", "client is not authorized to call CreateSale", http.StatusForbidden},
		{"unknown", "boom", http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gw := &mockGateway{
				submitFn: func(fn string, args ...string) ([]byte, error) {
					return nil, errors.New(tc.msg)
				},
			}
			svc := NewSaleService(gw)
			_, err := svc.Create(context.Background(), CreateSaleInput{ID: "sale1"})
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Code != tc.want {
				t.Errorf("want %d, got %d (msg=%s)", tc.want, err.Code, err.Message)
			}
		})
	}
}

func TestSaleService_Get_NotFound(t *testing.T) {
	gw := &mockGateway{
		evaluateFn: func(fn string, args ...string) ([]byte, error) {
			return nil, errors.New("sale sale1 does not exist")
		},
	}
	svc := NewSaleService(gw)
	_, err := svc.Get(context.Background(), "sale1")
	if err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %v", err)
	}
}
