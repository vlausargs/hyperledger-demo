//go:build integration
// +build integration

package contracts

import (
	"encoding/json"
	"testing"

	"github.com/myindo/hlf-supply-chain/chaincode/models"
)

// ---------------------------------------------------------------------------
// Multi-contract integration tests using the shared mock stub.
//
// These tests exercise the full supply chain flow across multiple contracts
// (Product → Shipment → Custody → Sale → Recall) within a single in-memory
// state. They run with: go test -tags=integration ./contracts/...
//
// The mock stub does NOT implement CouchDB selector semantics. Tests below
// rely only on key-value GetState/PutState/GetStateByRange — which is what
// the supply chain status-transition path actually uses.
// ---------------------------------------------------------------------------

// switchOrg rebinds the mock context's client identity to a different MSP.
// This simulates a transaction submitted by a different organization.
func switchOrg(ctx *mockCtx, mspID string) {
	ctx.identity = &mockClientIdentity{mspID: mspID}
}

// sharedCtx returns a single mockCtx whose stub is reused across the test.
// switchOrg mutates only the caller identity, leaving state intact.
func sharedCtx(mspID string) *mockCtx {
	return newMockCtx(mspID)
}

// TestIntegration_FullSupplyChain exercises create → ship → custody handoff
// → delivery → sale across Org1, Org2, Org3.
func TestIntegration_FullSupplyChain(t *testing.T) {
	ctx := sharedCtx("Org1MSP")

	product := &ProductContract{}
	shipment := &ShipmentContract{}
	custody := &CustodyContract{}
	sale := &SaleContract{}

	// --- Step 1: Org1 creates two products ----------------------------------
	if err := product.CreateProduct(ctx, "P-001", "SKU-A", "Widget A", "first widget",
		"BATCH-2026-01", "Acme", "2027-01-01", `{"weight":"1.5"}`); err != nil {
		t.Fatalf("CreateProduct P-001: %v", err)
	}
	if err := product.CreateProduct(ctx, "P-002", "SKU-B", "Widget B", "second widget",
		"BATCH-2026-01", "Acme", "2027-01-01", `{"weight":"2.0"}`); err != nil {
		t.Fatalf("CreateProduct P-002: %v", err)
	}

	p1, err := product.ReadProduct(ctx, "P-001")
	if err != nil {
		t.Fatalf("ReadProduct: %v", err)
	}
	if p1.Status != models.ProductStatusActive {
		t.Errorf("expected ACTIVE after create, got %s", p1.Status)
	}
	if p1.CurrentOwnerMSP != "Org1MSP" {
		t.Errorf("expected owner Org1MSP, got %s", p1.CurrentOwnerMSP)
	}

	// --- Step 2: Org1 creates shipment to Org3 (via Org2 intermediary) ------
	pidsJSON, _ := json.Marshal([]string{"P-001", "P-002"})
	if err := shipment.CreateShipment(ctx, "SHIP-001", "Order #1", "two widgets",
		"Org3MSP", "Retailer", "Factory", "Store", string(pidsJSON)); err != nil {
		t.Fatalf("CreateShipment: %v", err)
	}
	p1, _ = product.ReadProduct(ctx, "P-001")
	if p1.Status != models.ProductStatusShipped {
		t.Errorf("expected SHIPPED after shipment create, got %s", p1.Status)
	}
	if p1.CurrentShipmentID != "SHIP-001" {
		t.Errorf("expected CurrentShipmentID SHIP-001, got %s", p1.CurrentShipmentID)
	}

	// --- Step 3: Org1 dispatches the shipment -------------------------------
	if err := shipment.DispatchShipment(ctx, "SHIP-001"); err != nil {
		t.Fatalf("DispatchShipment: %v", err)
	}
	s, err := shipment.ReadShipment(ctx, "SHIP-001")
	if err != nil {
		t.Fatalf("ReadShipment: %v", err)
	}
	if s.Status != models.ShipmentStatusInTransit {
		t.Errorf("expected IN_TRANSIT after dispatch, got %s", s.Status)
	}

	// --- Step 4: Org1 initiates custody transfer to Org2 (distributor) ------
	if err := custody.InitiateCustodyTransfer(ctx, "SHIP-001", "Org2MSP",
		"Distributor", "keep cool"); err != nil {
		t.Fatalf("InitiateCustodyTransfer to Org2: %v", err)
	}

	// --- Step 5: Org2 accepts custody (intermediate handoff) ----------------
	switchOrg(ctx, "Org2MSP")
	if err := custody.AcceptCustodyTransfer(ctx, "SHIP-001"); err != nil {
		t.Fatalf("AcceptCustodyTransfer (Org2): %v", err)
	}
	p1, _ = product.ReadProduct(ctx, "P-001")
	if p1.CurrentOwnerMSP != "Org2MSP" {
		t.Errorf("expected owner Org2MSP after handoff, got %s", p1.CurrentOwnerMSP)
	}
	if p1.Status != models.ProductStatusShipped {
		t.Errorf("expected status SHIPPED during transit, got %s", p1.Status)
	}
	s, _ = shipment.ReadShipment(ctx, "SHIP-001")
	if s.Status != models.ShipmentStatusInTransit {
		t.Errorf("expected shipment still IN_TRANSIT after intermediate handoff, got %s", s.Status)
	}

	// --- Step 6: Org2 forwards to Org3 (final destination) ------------------
	if err := custody.InitiateCustodyTransfer(ctx, "SHIP-001", "Org3MSP",
		"Retailer", "deliver to store"); err != nil {
		t.Fatalf("InitiateCustodyTransfer to Org3: %v", err)
	}

	// --- Step 7: Org3 accepts → shipment DELIVERED, products DELIVERED -----
	switchOrg(ctx, "Org3MSP")
	if err := custody.AcceptCustodyTransfer(ctx, "SHIP-001"); err != nil {
		t.Fatalf("AcceptCustodyTransfer (Org3): %v", err)
	}
	s, _ = shipment.ReadShipment(ctx, "SHIP-001")
	if s.Status != models.ShipmentStatusDelivered {
		t.Errorf("expected DELIVERED after final accept, got %s", s.Status)
	}
	p1, _ = product.ReadProduct(ctx, "P-001")
	if p1.Status != models.ProductStatusDelivered {
		t.Errorf("expected product DELIVERED, got %s", p1.Status)
	}
	if p1.CurrentOwnerMSP != "Org3MSP" {
		t.Errorf("expected owner Org3MSP, got %s", p1.CurrentOwnerMSP)
	}
	if p1.CurrentShipmentID != "" {
		t.Errorf("expected CurrentShipmentID cleared after delivery, got %s", p1.CurrentShipmentID)
	}

	// --- Step 8: Org3 sells P-001 at POS ------------------------------------
	itemsJSON, _ := json.Marshal([]map[string]interface{}{
		{"productId": "P-001", "unitPrice": 1500.0},
	})
	if err := sale.CreateSale(ctx, "SALE-001", "CUST-100", "CASH-1", "Bob",
		string(itemsJSON), "150.0", "IDR", "morning sale"); err != nil {
		t.Fatalf("CreateSale: %v", err)
	}
	p1, _ = product.ReadProduct(ctx, "P-001")
	if p1.Status != models.ProductStatusSold {
		t.Errorf("expected SOLD after POS, got %s", p1.Status)
	}
	saleRecord, err := sale.ReadSale(ctx, "SALE-001")
	if err != nil {
		t.Fatalf("ReadSale: %v", err)
	}
	if saleRecord.TotalAmount != 1650.0 {
		t.Errorf("expected total 1650.0, got %v", saleRecord.TotalAmount)
	}
}

// TestIntegration_RecallProductScope verifies recall transitions an active
// product to RECALLED status and stamps RecallID.
func TestIntegration_RecallProductScope(t *testing.T) {
	ctx := sharedCtx("Org1MSP")
	product := &ProductContract{}
	recall := &RecallContract{}

	if err := product.CreateProduct(ctx, "P-100", "SKU-X", "Faulty",
		"recall me", "BATCH-BAD", "Acme", "", ""); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	targets, _ := json.Marshal([]string{"P-100"})
	if err := recall.IssueRecall(ctx, "REC-1", models.RecallScopeProduct,
		string(targets), "contamination", "HIGH", "QA Team", "https://recall.example/REC-1"); err != nil {
		t.Fatalf("IssueRecall: %v", err)
	}

	p, _ := product.ReadProduct(ctx, "P-100")
	if p.Status != models.ProductStatusRecalled {
		t.Errorf("expected RECALLED, got %s", p.Status)
	}
	if p.RecallID != "REC-1" {
		t.Errorf("expected RecallID REC-1, got %s", p.RecallID)
	}

	r, err := recall.ReadRecall(ctx, "REC-1")
	if err != nil {
		t.Fatalf("ReadRecall: %v", err)
	}
	if r.AffectedCount != 1 {
		t.Errorf("expected affected=1, got %d", r.AffectedCount)
	}
	if r.Scope != models.RecallScopeProduct {
		t.Errorf("expected scope PRODUCT, got %s", r.Scope)
	}
}

// TestIntegration_RecallShipmentCascades verifies recalling a shipment also
// recalls all child products.
func TestIntegration_RecallShipmentCascades(t *testing.T) {
	ctx := sharedCtx("Org1MSP")
	product := &ProductContract{}
	shipment := &ShipmentContract{}
	recall := &RecallContract{}

	for _, pid := range []string{"P-A", "P-B"} {
		if err := product.CreateProduct(ctx, pid, "SKU-"+pid, pid,
			"", "B-1", "Acme", "", ""); err != nil {
			t.Fatalf("CreateProduct %s: %v", pid, err)
		}
	}
	pidsJSON, _ := json.Marshal([]string{"P-A", "P-B"})
	if err := shipment.CreateShipment(ctx, "S-1", "Order", "",
		"Org3MSP", "Retailer", "F", "S", string(pidsJSON)); err != nil {
		t.Fatalf("CreateShipment: %v", err)
	}

	targets, _ := json.Marshal([]string{"S-1"})
	if err := recall.IssueRecall(ctx, "REC-2", models.RecallScopeShipment,
		string(targets), "broken pallet", "MEDIUM", "Ops", ""); err != nil {
		t.Fatalf("IssueRecall: %v", err)
	}

	s, _ := shipment.ReadShipment(ctx, "S-1")
	if s.Status != models.ShipmentStatusRecalled {
		t.Errorf("expected shipment RECALLED, got %s", s.Status)
	}
	for _, pid := range []string{"P-A", "P-B"} {
		p, _ := product.ReadProduct(ctx, pid)
		if p.Status != models.ProductStatusRecalled {
			t.Errorf("expected product %s RECALLED, got %s", pid, p.Status)
		}
		if p.RecallID != "REC-2" {
			t.Errorf("expected product %s RecallID REC-2, got %s", pid, p.RecallID)
		}
	}
}

// TestIntegration_AccessControl verifies cross-org rejections.
func TestIntegration_AccessControl(t *testing.T) {
	ctx := sharedCtx("Org1MSP")
	product := &ProductContract{}
	shipment := &ShipmentContract{}

	if err := product.CreateProduct(ctx, "P-AC", "SKU", "X", "", "B", "Acme", "", ""); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	// Org2 may not create a product (Org1MSP-only).
	switchOrg(ctx, "Org2MSP")
	err := product.CreateProduct(ctx, "P-NEW", "SKU", "Y", "", "B", "Acme", "", "")
	if err == nil {
		t.Error("expected Org2 CreateProduct to fail (Org1MSP only)")
	}

	// Org2 may not ship Org1-owned product.
	pidsJSON, _ := json.Marshal([]string{"P-AC"})
	err = shipment.CreateShipment(ctx, "S-X", "n", "", "Org3MSP", "R", "O", "D", string(pidsJSON))
	if err == nil {
		t.Error("expected Org2 CreateShipment of Org1 product to fail")
	}
}

// TestIntegration_IllegalStatusTransitions verifies that ValidateProductStatusTransition
// and ValidateShipmentStatusTransition guards reject illegal state changes (spec 2.4).
func TestIntegration_IllegalStatusTransitions(t *testing.T) {
	ctx := sharedCtx("Org1MSP")
	product := &ProductContract{}
	recall := &RecallContract{}

	// Required-field validation: empty manufacturerName must be rejected.
	if err := product.CreateProduct(ctx, "P-REQ", "SKU", "Name", "", "B-1", "", "", ""); err == nil {
		t.Error("expected CreateProduct with empty manufacturerName to fail (ValidateRequired)")
	}

	// Create a normal product and move it through ACTIVE → SOLD by direct state insertion,
	// then attempt to recall it. SOLD has no outgoing transitions in productTransitions,
	// so ValidateProductStatusTransition must reject the recall.
	if err := product.CreateProduct(ctx, "P-SOLD", "SKU", "Sold Widget", "",
		"B-1", "Acme", "", ""); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	p, _ := product.ReadProduct(ctx, "P-SOLD")
	p.Status = models.ProductStatusSold
	raw, _ := json.Marshal(p)
	if err := ctx.GetStub().PutState(models.PrefixProduct+"P-SOLD", raw); err != nil {
		t.Fatalf("PutState: %v", err)
	}

	targets, _ := json.Marshal([]string{"P-SOLD"})
	err := recall.IssueRecall(ctx, "REC-ILLEGAL", models.RecallScopeProduct,
		string(targets), "test", "HIGH", "QA", "")
	if err == nil {
		t.Error("expected IssueRecall on SOLD product to fail (illegal SOLD->RECALLED transition)")
	}
}

// TestIntegration_RejectCustodyKeepsState verifies a rejected custody does
// NOT change shipment status away from IN_TRANSIT.
func TestIntegration_RejectCustodyKeepsState(t *testing.T) {
	ctx := sharedCtx("Org1MSP")
	product := &ProductContract{}
	shipment := &ShipmentContract{}
	custody := &CustodyContract{}

	if err := product.CreateProduct(ctx, "P-R", "SKU", "X", "", "B", "Acme", "", ""); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	pidsJSON, _ := json.Marshal([]string{"P-R"})
	if err := shipment.CreateShipment(ctx, "S-R", "n", "", "Org3MSP", "R", "O", "D", string(pidsJSON)); err != nil {
		t.Fatalf("CreateShipment: %v", err)
	}
	if err := shipment.DispatchShipment(ctx, "S-R"); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if err := custody.InitiateCustodyTransfer(ctx, "S-R", "Org2MSP", "Dist", ""); err != nil {
		t.Fatalf("Initiate: %v", err)
	}

	switchOrg(ctx, "Org2MSP")
	if err := custody.RejectCustodyTransfer(ctx, "S-R"); err != nil {
		t.Fatalf("Reject: %v", err)
	}

	s, _ := shipment.ReadShipment(ctx, "S-R")
	if s.Status != models.ShipmentStatusInTransit {
		t.Errorf("expected shipment still IN_TRANSIT after reject, got %s", s.Status)
	}

	// Org1 can now reroute by initiating another transfer (sender still Org1
	// because shipment.SenderMSP only updates on accept).
	switchOrg(ctx, "Org1MSP")
	if err := custody.InitiateCustodyTransfer(ctx, "S-R", "Org3MSP", "Retailer", ""); err != nil {
		t.Fatalf("Initiate-2: %v", err)
	}
}
