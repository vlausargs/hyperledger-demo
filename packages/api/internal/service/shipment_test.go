package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestShipmentService_Create_HappyPath(t *testing.T) {
	var gotArgs []string
	gw := &mockGateway{
		submitFn: func(fn string, args ...string) ([]byte, error) {
			gotArgs = args
			return nil, nil
		},
	}
	svc := NewShipmentService(gw)
	res, err := svc.Create(context.Background(), CreateShipmentInput{
		ID:           "s1",
		Name:         "boxA",
		ReceiverMSP:  "Org2MSP",
		ReceiverName: "Distributor",
		ProductIDs:   []string{"p1", "p2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "s1" {
		t.Errorf("want id=s1, got %s", res.ID)
	}
	if gotArgs[7] != `["p1","p2"]` {
		t.Errorf("want productIDs JSON array, got %q", gotArgs[7])
	}
}

func TestShipmentService_Create_Conflict(t *testing.T) {
	gw := &mockGateway{
		submitFn: func(fn string, args ...string) ([]byte, error) {
			return nil, errors.New("shipment s1 already exists")
		},
	}
	svc := NewShipmentService(gw)
	_, err := svc.Create(context.Background(), CreateShipmentInput{ID: "s1"})
	if err == nil || err.Code != http.StatusConflict {
		t.Fatalf("want 409, got %v", err)
	}
}

func TestShipmentService_Dispatch_NotFound(t *testing.T) {
	gw := &mockGateway{
		submitFn: func(fn string, args ...string) ([]byte, error) {
			return nil, errors.New("shipment does not exist")
		},
	}
	svc := NewShipmentService(gw)
	_, err := svc.Dispatch(context.Background(), "missing")
	if err == nil || err.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %v", err)
	}
}

func TestShipmentService_Get_HappyPath(t *testing.T) {
	gw := &mockGateway{
		evaluateFn: func(fn string, args ...string) ([]byte, error) {
			if fn != "ShipmentContract:ReadShipment" || args[0] != "s1" {
				t.Fatalf("unexpected call: %s %v", fn, args)
			}
			return []byte(`{"id":"s1"}`), nil
		},
	}
	svc := NewShipmentService(gw)
	out, err := svc.Get(context.Background(), "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != `{"id":"s1"}` {
		t.Errorf("unexpected output: %s", string(out))
	}
}
