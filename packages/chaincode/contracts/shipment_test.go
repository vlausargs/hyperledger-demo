package contracts

import (
	"encoding/json"
	"testing"

	"github.com/myindo/hlf-supply-chain/chaincode/models"
)

func TestShipmentContract_ShipmentExists_False(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &ShipmentContract{}

	exists, err := contract.ShipmentExists(ctx, "ship-001")
	if err != nil {
		t.Fatalf("ShipmentExists returned unexpected error: %v", err)
	}
	if exists {
		t.Error("expected ShipmentExists to return false for missing shipment, got true")
	}
}

func TestShipmentContract_ShipmentExists_True(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &ShipmentContract{}

	s := models.Shipment{
		DocType:    "SHIPMENT",
		ID:         "ship-001",
		Name:       "Test Shipment",
		ProductIDs: []string{"prod-001"},
		SenderMSP:  "Org1MSP",
		Status:     models.ShipmentStatusDraft,
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("failed to marshal shipment: %v", err)
	}
	if err := ctx.GetStub().PutState(models.PrefixShipment+"ship-001", data); err != nil {
		t.Fatalf("PutState failed: %v", err)
	}

	exists, err := contract.ShipmentExists(ctx, "ship-001")
	if err != nil {
		t.Fatalf("ShipmentExists returned unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected ShipmentExists to return true after inserting shipment, got false")
	}
}

func TestShipmentContract_ReadShipment_NotFound(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &ShipmentContract{}

	_, err := contract.ReadShipment(ctx, "nonexistent-shipment")
	if err == nil {
		t.Fatal("expected error when reading nonexistent shipment, got nil")
	}
}

func TestShipmentContract_ReadShipment_Found(t *testing.T) {
	ctx := newMockCtx("Org1MSP")
	contract := &ShipmentContract{}

	s := models.Shipment{
		DocType:      "SHIPMENT",
		ID:           "ship-001",
		Name:         "Test Shipment",
		Description:  "A test shipment",
		ProductIDs:   []string{"prod-001", "prod-002"},
		SenderMSP:    "Org1MSP",
		SenderName:   "Manufacturer",
		ReceiverMSP:  "Org2MSP",
		ReceiverName: "Distributor",
		Status:       models.ShipmentStatusDraft,
		Origin:       "Jakarta",
		Destination:  "Surabaya",
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("failed to marshal shipment: %v", err)
	}
	if err := ctx.GetStub().PutState(models.PrefixShipment+"ship-001", data); err != nil {
		t.Fatalf("PutState failed: %v", err)
	}

	result, err := contract.ReadShipment(ctx, "ship-001")
	if err != nil {
		t.Fatalf("ReadShipment returned unexpected error: %v", err)
	}
	if result.ID != "ship-001" {
		t.Errorf("expected shipment ID ship-001, got %s", result.ID)
	}
	if result.Name != "Test Shipment" {
		t.Errorf("expected shipment name 'Test Shipment', got %s", result.Name)
	}
	if result.Status != models.ShipmentStatusDraft {
		t.Errorf("expected status DRAFT, got %s", result.Status)
	}
	if len(result.ProductIDs) != 2 {
		t.Errorf("expected 2 product IDs, got %d", len(result.ProductIDs))
	}
	if result.SenderMSP != "Org1MSP" {
		t.Errorf("expected sender MSP Org1MSP, got %s", result.SenderMSP)
	}
	if result.ReceiverMSP != "Org2MSP" {
		t.Errorf("expected receiver MSP Org2MSP, got %s", result.ReceiverMSP)
	}
}
