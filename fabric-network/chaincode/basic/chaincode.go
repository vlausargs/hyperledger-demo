/*
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SmartContract provides functions for managing a supply chain
type SmartContract struct {
	contractapi.Contract
}

// ── Status constants ──────────────────────────────────────────────────────────

const (
	ProductStatusActive    = "ACTIVE"
	ProductStatusShipped   = "SHIPPED"
	ProductStatusDelivered = "DELIVERED"
	ProductStatusRecalled  = "RECALLED"
	ProductStatusScrapped  = "SCRAPPED"
	ProductStatusSold      = "SOLD"

	ShipmentStatusDraft     = "DRAFT"
	ShipmentStatusInTransit = "IN_TRANSIT"
	ShipmentStatusDelivered = "DELIVERED"
	ShipmentStatusRecalled  = "RECALLED"
	ShipmentStatusCancelled = "CANCELLED"

	CustodyStatusPending   = "PENDING"
	CustodyStatusCompleted = "COMPLETED"
	CustodyStatusRejected  = "REJECTED"

	RecallScopeProduct  = "PRODUCT"
	RecallScopeShipment = "SHIPMENT"
	RecallScopeBatch    = "BATCH"

	RecallSeverityLow      = "LOW"
	RecallSeverityMedium   = "MEDIUM"
	RecallSeverityHigh     = "HIGH"
	RecallSeverityCritical = "CRITICAL"
)

// ── Key prefix constants ──────────────────────────────────────────────────────

const (
	prefixProduct  = "PROD~"
	prefixShipment = "SHIP~"
	prefixCustody  = "CUSTODY~"
	prefixEvent    = "EVENT~"
	prefixRecall   = "RECALL~"
	prefixSale     = "SALE~"
)

// ── Data models ───────────────────────────────────────────────────────────────

type Product struct {
	DocType           string            `json:"docType"`
	ID                string            `json:"id"`
	SKU               string            `json:"sku"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	BatchID           string            `json:"batchId"`
	ManufacturerID    string            `json:"manufacturerId"`
	ManufacturerName  string            `json:"manufacturerName"`
	ManufacturedAt    string            `json:"manufacturedAt"`
	ExpiryDate        string            `json:"expiryDate"`
	Status            string            `json:"status"`
	CurrentOwnerMSP   string            `json:"currentOwnerMSP"`
	CurrentOwner      string            `json:"currentOwner"`
	CurrentShipmentID string            `json:"currentShipmentId"`
	RecallID          string            `json:"recallId"`
	Metadata          map[string]string `json:"metadata"`
	CreatedAt         string            `json:"createdAt"`
	UpdatedAt         string            `json:"updatedAt"`
}

type Shipment struct {
	DocType      string   `json:"docType"`
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	ProductIDs   []string `json:"productIds"`
	SenderMSP    string   `json:"senderMsp"`
	SenderName   string   `json:"senderName"`
	ReceiverMSP  string   `json:"receiverMsp"`
	ReceiverName string   `json:"receiverName"`
	Status       string   `json:"status"`
	Origin       string   `json:"origin"`
	Destination  string   `json:"destination"`
	DepartureAt  string   `json:"departureAt"`
	ArrivalAt    string   `json:"arrivalAt"`
	RecallID     string   `json:"recallId"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
}

type CustodyRecord struct {
	DocType           string `json:"docType"`
	ID                string `json:"id"`
	ShipmentID        string `json:"shipmentId"`
	Sequence          int    `json:"sequence"`
	FromMSP           string `json:"fromMsp"`
	FromName          string `json:"fromName"`
	ToMSP             string `json:"toMsp"`
	ToName            string `json:"toName"`
	Status            string `json:"status"`
	TransferredAt     string `json:"transferredAt"`
	Conditions        string `json:"conditions"`
	SenderSignature   string `json:"senderSignature"`
	ReceiverSignature string `json:"receiverSignature"`
	TxID              string `json:"txId"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

type SupplyChainEvent struct {
	DocType     string            `json:"docType"`
	ID          string            `json:"id"`
	TargetID    string            `json:"targetId"`
	TargetType  string            `json:"targetType"`
	EventType   string            `json:"eventType"`
	Description string            `json:"description"`
	Location    string            `json:"location"`
	RecordedBy  string            `json:"recordedBy"`
	Data        map[string]string `json:"data"`
	OccurredAt  string            `json:"occurredAt"`
	TxID        string            `json:"txId"`
	CreatedAt   string            `json:"createdAt"`
}

type RecallNotice struct {
	DocType         string   `json:"docType"`
	ID              string   `json:"id"`
	Scope           string   `json:"scope"`
	TargetIDs       []string `json:"targetIds"`
	Reason          string   `json:"reason"`
	IssuedBy        string   `json:"issuedBy"`
	IssuedByName    string   `json:"issuedByName"`
	Severity        string   `json:"severity"`
	InstructionsURL string   `json:"instructionsUrl"`
	AffectedCount   int      `json:"affectedCount"`
	CreatedAt       string   `json:"createdAt"`
}

type SaleItem struct {
	ProductID   string  `json:"productId"`
	SKU         string  `json:"sku"`
	ProductName string  `json:"productName"`
	UnitPrice   float64 `json:"unitPrice"`
}

type SaleTransaction struct {
	DocType     string     `json:"docType"`
	ID          string     `json:"id"`
	Items       []SaleItem `json:"items"`
	CustomerID  string     `json:"customerId"`
	CashierID   string     `json:"cashierId"`
	CashierName string     `json:"cashierName"`
	SubTotal    float64    `json:"subTotal"`
	TaxAmount   float64    `json:"taxAmount"`
	TotalAmount float64    `json:"totalAmount"`
	Currency    string     `json:"currency"`
	RetailerMSP string     `json:"retailerMsp"`
	Notes       string     `json:"notes"`
	TxID        string     `json:"txId"`
	CreatedAt   string     `json:"createdAt"`
}

type InventoryItem struct {
	SKU        string   `json:"sku"`
	Name       string   `json:"name"`
	Count      int      `json:"count"`
	ProductIDs []string `json:"productIds"`
}

// ── Paginated result wrappers ─────────────────────────────────────────────────

type PagedProductResult struct {
	Products []*Product `json:"products"`
	Bookmark string     `json:"bookmark"`
	Count    int        `json:"count"`
}

type PagedShipmentResult struct {
	Shipments []*Shipment `json:"shipments"`
	Bookmark  string      `json:"bookmark"`
	Count     int         `json:"count"`
}

type PagedSaleResult struct {
	Sales    []*SaleTransaction `json:"sales"`
	Bookmark string             `json:"bookmark"`
	Count    int                `json:"count"`
}

type HistoryQueryResult struct {
	TxID      string      `json:"txId"`
	Timestamp time.Time   `json:"timestamp"`
	IsDelete  bool        `json:"isDelete"`
	Value     interface{} `json:"value"`
}

type ProvenanceResult struct {
	Product  *Product            `json:"product"`
	History  []*HistoryQueryResult `json:"history"`
	Custody  []*CustodyRecord    `json:"custody"`
	Events   []*SupplyChainEvent `json:"events"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func callerMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
	msp, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get caller MSPID: %w", err)
	}
	return msp, nil
}

func requireMSP(ctx contractapi.TransactionContextInterface, allowed ...string) (string, error) {
	caller, err := callerMSPID(ctx)
	if err != nil {
		return "", err
	}
	for _, a := range allowed {
		if caller == a {
			return caller, nil
		}
	}
	return "", fmt.Errorf("caller MSP %s is not authorized; allowed: %s", caller, strings.Join(allowed, ", "))
}

func txTimestamp(ctx contractapi.TransactionContextInterface) (string, error) {
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return "", fmt.Errorf("failed to get tx timestamp: %w", err)
	}
	return ts.AsTime().UTC().Format(time.RFC3339), nil
}

func putJSON(ctx contractapi.TransactionContextInterface, key string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", key, err)
	}
	return ctx.GetStub().PutState(key, b)
}

func getProductByID(ctx contractapi.TransactionContextInterface, id string) (*Product, error) {
	b, err := ctx.GetStub().GetState(prefixProduct + id)
	if err != nil {
		return nil, fmt.Errorf("failed to read product %s: %w", id, err)
	}
	if b == nil {
		return nil, fmt.Errorf("product %s does not exist", id)
	}
	var p Product
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal product %s: %w", id, err)
	}
	return &p, nil
}

func getShipmentByID(ctx contractapi.TransactionContextInterface, id string) (*Shipment, error) {
	b, err := ctx.GetStub().GetState(prefixShipment + id)
	if err != nil {
		return nil, fmt.Errorf("failed to read shipment %s: %w", id, err)
	}
	if b == nil {
		return nil, fmt.Errorf("shipment %s does not exist", id)
	}
	var s Shipment
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal shipment %s: %w", id, err)
	}
	return &s, nil
}

// queryProducts executes a CouchDB selector query and returns matching products.
func queryProducts(ctx contractapi.TransactionContextInterface, query string) ([]*Product, error) {
	iter, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer iter.Close()
	var products []*Product
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var p Product
		if err := json.Unmarshal(res.Value, &p); err != nil {
			return nil, err
		}
		products = append(products, &p)
	}
	if products == nil {
		products = make([]*Product, 0)
	}
	return products, nil
}

// findPendingCustody scans CUSTODY~{shipmentID}~ range and returns (count, pendingRecord, error).
func findPendingCustody(ctx contractapi.TransactionContextInterface, shipmentID string) (int, *CustodyRecord, error) {
	start := prefixCustody + shipmentID + "~"
	end := prefixCustody + shipmentID + "~\x7f"
	iter, err := ctx.GetStub().GetStateByRange(start, end)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to scan custody records: %w", err)
	}
	defer iter.Close()

	count := 0
	var pending *CustodyRecord
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return 0, nil, err
		}
		count++
		var cr CustodyRecord
		if err := json.Unmarshal(res.Value, &cr); err != nil {
			return 0, nil, err
		}
		if cr.Status == CustodyStatusPending {
			cp := cr
			pending = &cp
		}
	}
	return count, pending, nil
}

// ── InitLedger ────────────────────────────────────────────────────────────────

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	caller, err := callerMSPID(ctx)
	if err != nil {
		return err
	}
	now, err := txTimestamp(ctx)
	if err != nil {
		return err
	}

	products := []Product{
		{
			DocType: "PRODUCT", ID: "PROD-001", SKU: "SKU-A100", Name: "Widget Alpha",
			Description: "Industrial widget", BatchID: "BATCH-2026-01",
			ManufacturerID: caller, ManufacturerName: "Acme Manufacturing",
			ManufacturedAt: now, Status: ProductStatusActive,
			CurrentOwnerMSP: caller, CurrentOwner: "Acme Manufacturing",
			Metadata: map[string]string{"weight_kg": "1.5", "material": "steel"},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			DocType: "PRODUCT", ID: "PROD-002", SKU: "SKU-B200", Name: "Widget Beta",
			Description: "Consumer widget", BatchID: "BATCH-2026-01",
			ManufacturerID: caller, ManufacturerName: "Acme Manufacturing",
			ManufacturedAt: now, Status: ProductStatusActive,
			CurrentOwnerMSP: caller, CurrentOwner: "Acme Manufacturing",
			Metadata: map[string]string{"weight_kg": "0.8", "material": "plastic"},
			CreatedAt: now, UpdatedAt: now,
		},
	}
	for _, p := range products {
		if err := putJSON(ctx, prefixProduct+p.ID, p); err != nil {
			return err
		}
	}
	return nil
}

// ── Product functions ─────────────────────────────────────────────────────────

func (s *SmartContract) CreateProduct(ctx contractapi.TransactionContextInterface,
	id, sku, name, description, batchID, manufacturerName, expiryDate, metaJSON string) error {

	caller, err := requireMSP(ctx, "Org1MSP")
	if err != nil {
		return err
	}
	now, err := txTimestamp(ctx)
	if err != nil {
		return err
	}
	exists, err := s.ProductExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("product %s already exists", id)
	}

	var meta map[string]string
	if metaJSON != "" {
		if err := json.Unmarshal([]byte(metaJSON), &meta); err != nil {
			return fmt.Errorf("invalid metadata JSON: %w", err)
		}
	}

	p := Product{
		DocType: "PRODUCT", ID: id, SKU: sku, Name: name, Description: description,
		BatchID: batchID, ManufacturerID: caller, ManufacturerName: manufacturerName,
		ManufacturedAt: now, ExpiryDate: expiryDate, Status: ProductStatusActive,
		CurrentOwnerMSP: caller, CurrentOwner: manufacturerName,
		Metadata: meta, CreatedAt: now, UpdatedAt: now,
	}
	return putJSON(ctx, prefixProduct+id, p)
}

func (s *SmartContract) ReadProduct(ctx contractapi.TransactionContextInterface, id string) (*Product, error) {
	return getProductByID(ctx, id)
}

func (s *SmartContract) UpdateProduct(ctx contractapi.TransactionContextInterface,
	id, name, description, expiryDate, metaJSON string) error {

	p, err := getProductByID(ctx, id)
	if err != nil {
		return err
	}
	caller, err := callerMSPID(ctx)
	if err != nil {
		return err
	}
	if caller != p.CurrentOwnerMSP {
		return fmt.Errorf("caller MSP %s is not the current owner of product %s", caller, id)
	}
	now, err := txTimestamp(ctx)
	if err != nil {
		return err
	}

	if name != "" {
		p.Name = name
	}
	if description != "" {
		p.Description = description
	}
	if expiryDate != "" {
		p.ExpiryDate = expiryDate
	}
	if metaJSON != "" {
		var meta map[string]string
		if err := json.Unmarshal([]byte(metaJSON), &meta); err != nil {
			return fmt.Errorf("invalid metadata JSON: %w", err)
		}
		p.Metadata = meta
	}
	p.UpdatedAt = now
	return putJSON(ctx, prefixProduct+id, p)
}

func (s *SmartContract) ProductExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	b, err := ctx.GetStub().GetState(prefixProduct + id)
	if err != nil {
		return false, fmt.Errorf("failed to read product %s: %w", id, err)
	}
	return b != nil, nil
}

func (s *SmartContract) GetAllProducts(ctx contractapi.TransactionContextInterface, pageSize int, bookmark string) (*PagedProductResult, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	start := prefixProduct
	end := prefixProduct + "\x7f"
	iter, meta, err := ctx.GetStub().GetStateByRangeWithPagination(start, end, int32(pageSize), bookmark)
	if err != nil {
		return nil, fmt.Errorf("failed to get paged products: %w", err)
	}
	defer iter.Close()

	var products []*Product
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var p Product
		if err := json.Unmarshal(res.Value, &p); err != nil {
			return nil, err
		}
		products = append(products, &p)
	}
	if products == nil {
		products = make([]*Product, 0)
	}
	return &PagedProductResult{Products: products, Bookmark: meta.Bookmark, Count: len(products)}, nil
}

func (s *SmartContract) GetProductsByBatch(ctx contractapi.TransactionContextInterface, batchID string) ([]*Product, error) {
	q := fmt.Sprintf(`{"selector":{"docType":"PRODUCT","batchId":"%s"}}`, batchID)
	return queryProducts(ctx, q)
}

func (s *SmartContract) GetProductsByStatus(ctx contractapi.TransactionContextInterface, status string) ([]*Product, error) {
	q := fmt.Sprintf(`{"selector":{"docType":"PRODUCT","status":"%s"}}`, status)
	return queryProducts(ctx, q)
}

func (s *SmartContract) GetProductsByOwner(ctx contractapi.TransactionContextInterface, ownerMSP string) ([]*Product, error) {
	q := fmt.Sprintf(`{"selector":{"docType":"PRODUCT","currentOwnerMSP":"%s"}}`, ownerMSP)
	return queryProducts(ctx, q)
}

func (s *SmartContract) GetProductHistory(ctx contractapi.TransactionContextInterface, id string) ([]*HistoryQueryResult, error) {
	return collectHistoryForKey(ctx, prefixProduct+id)
}

func (s *SmartContract) GetProvenance(ctx contractapi.TransactionContextInterface, productID string) (*ProvenanceResult, error) {
	p, err := getProductByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	history, err := s.GetProductHistory(ctx, productID)
	if err != nil {
		return nil, err
	}

	custody := make([]*CustodyRecord, 0)
	if p.CurrentShipmentID != "" {
		custody, err = s.GetCustodyChain(ctx, p.CurrentShipmentID)
		if err != nil {
			return nil, err
		}
	}

	events, err := s.GetEvents(ctx, productID)
	if err != nil {
		return nil, err
	}

	if history == nil {
		history = make([]*HistoryQueryResult, 0)
	}
	if events == nil {
		events = make([]*SupplyChainEvent, 0)
	}

	return &ProvenanceResult{Product: p, History: history, Custody: custody, Events: events}, nil
}

// ── Shipment functions ────────────────────────────────────────────────────────

func (s *SmartContract) CreateShipment(ctx contractapi.TransactionContextInterface,
	id, name, description, receiverMSP, receiverName, origin, destination, productIDsJSON string) error {

	caller, err := callerMSPID(ctx)
	if err != nil {
		return err
	}
	now, err := txTimestamp(ctx)
	if err != nil {
		return err
	}

	exists, err := s.ShipmentExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("shipment %s already exists", id)
	}

	var productIDs []string
	if err := json.Unmarshal([]byte(productIDsJSON), &productIDs); err != nil {
		return fmt.Errorf("invalid productIds JSON: %w", err)
	}
	if len(productIDs) == 0 {
		return fmt.Errorf("shipment must contain at least one product")
	}

	// Validate all products exist and are owned by caller
	for _, pid := range productIDs {
		p, err := getProductByID(ctx, pid)
		if err != nil {
			return err
		}
		if p.CurrentOwnerMSP != caller {
			return fmt.Errorf("caller MSP %s does not own product %s (owner: %s)", caller, pid, p.CurrentOwnerMSP)
		}
		if p.Status != ProductStatusActive {
			return fmt.Errorf("product %s is not in ACTIVE status (current: %s)", pid, p.Status)
		}
	}

	// Update product references
	for _, pid := range productIDs {
		p, _ := getProductByID(ctx, pid)
		p.CurrentShipmentID = id
		p.Status = ProductStatusShipped
		p.UpdatedAt = now
		if err := putJSON(ctx, prefixProduct+pid, p); err != nil {
			return err
		}
	}

	ship := Shipment{
		DocType: "SHIPMENT", ID: id, Name: name, Description: description,
		ProductIDs: productIDs, SenderMSP: caller, SenderName: caller,
		ReceiverMSP: receiverMSP, ReceiverName: receiverName,
		Status: ShipmentStatusDraft, Origin: origin, Destination: destination,
		CreatedAt: now, UpdatedAt: now,
	}
	return putJSON(ctx, prefixShipment+id, ship)
}

func (s *SmartContract) ReadShipment(ctx contractapi.TransactionContextInterface, id string) (*Shipment, error) {
	return getShipmentByID(ctx, id)
}

func (s *SmartContract) DispatchShipment(ctx contractapi.TransactionContextInterface, id string) error {
	ship, err := getShipmentByID(ctx, id)
	if err != nil {
		return err
	}
	caller, err := callerMSPID(ctx)
	if err != nil {
		return err
	}
	if caller != ship.SenderMSP {
		return fmt.Errorf("only sender MSP %s may dispatch shipment %s", ship.SenderMSP, id)
	}
	if ship.Status != ShipmentStatusDraft {
		return fmt.Errorf("shipment %s is not in DRAFT status (current: %s)", id, ship.Status)
	}
	now, err := txTimestamp(ctx)
	if err != nil {
		return err
	}
	ship.Status = ShipmentStatusInTransit
	ship.DepartureAt = now
	ship.UpdatedAt = now
	return putJSON(ctx, prefixShipment+id, ship)
}

func (s *SmartContract) ShipmentExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	b, err := ctx.GetStub().GetState(prefixShipment + id)
	if err != nil {
		return false, fmt.Errorf("failed to read shipment %s: %w", id, err)
	}
	return b != nil, nil
}

func (s *SmartContract) GetAllShipments(ctx contractapi.TransactionContextInterface, pageSize int, bookmark string) (*PagedShipmentResult, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	start := prefixShipment
	end := prefixShipment + "\x7f"
	iter, meta, err := ctx.GetStub().GetStateByRangeWithPagination(start, end, int32(pageSize), bookmark)
	if err != nil {
		return nil, fmt.Errorf("failed to get paged shipments: %w", err)
	}
	defer iter.Close()

	var shipments []*Shipment
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var sh Shipment
		if err := json.Unmarshal(res.Value, &sh); err != nil {
			return nil, err
		}
		shipments = append(shipments, &sh)
	}
	if shipments == nil {
		shipments = make([]*Shipment, 0)
	}
	return &PagedShipmentResult{Shipments: shipments, Bookmark: meta.Bookmark, Count: len(shipments)}, nil
}

func (s *SmartContract) GetShipmentsByStatus(ctx contractapi.TransactionContextInterface, status string) ([]*Shipment, error) {
	q := fmt.Sprintf(`{"selector":{"docType":"SHIPMENT","status":"%s"}}`, status)
	iter, err := ctx.GetStub().GetQueryResult(q)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer iter.Close()
	var shipments []*Shipment
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var sh Shipment
		if err := json.Unmarshal(res.Value, &sh); err != nil {
			return nil, err
		}
		shipments = append(shipments, &sh)
	}
	if shipments == nil {
		shipments = make([]*Shipment, 0)
	}
	return shipments, nil
}

func (s *SmartContract) GetShipmentHistory(ctx contractapi.TransactionContextInterface, id string) ([]*HistoryQueryResult, error) {
	return collectHistoryForKey(ctx, prefixShipment+id)
}

// ── Custody transfer functions ────────────────────────────────────────────────

func (s *SmartContract) InitiateCustodyTransfer(ctx contractapi.TransactionContextInterface,
	shipmentID, toMSP, toName, conditions string) error {

	ship, err := getShipmentByID(ctx, shipmentID)
	if err != nil {
		return err
	}
	caller, err := callerMSPID(ctx)
	if err != nil {
		return err
	}
	if caller != ship.SenderMSP {
		return fmt.Errorf("only sender MSP %s may initiate custody transfer for shipment %s", ship.SenderMSP, shipmentID)
	}
	if ship.Status != ShipmentStatusInTransit {
		return fmt.Errorf("shipment %s must be IN_TRANSIT to initiate custody transfer (current: %s)", shipmentID, ship.Status)
	}

	count, pending, err := findPendingCustody(ctx, shipmentID)
	if err != nil {
		return err
	}
	if pending != nil {
		return fmt.Errorf("shipment %s already has a pending custody transfer", shipmentID)
	}

	now, err := txTimestamp(ctx)
	if err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()

	seq := count + 1
	custodyID := fmt.Sprintf("%s~%04d", shipmentID, seq)
	cr := CustodyRecord{
		DocType: "CUSTODY", ID: custodyID, ShipmentID: shipmentID,
		Sequence: seq, FromMSP: caller, FromName: caller,
		ToMSP: toMSP, ToName: toName, Status: CustodyStatusPending,
		Conditions: conditions, SenderSignature: txID,
		TxID: txID, CreatedAt: now, UpdatedAt: now,
	}
	return putJSON(ctx, prefixCustody+custodyID, cr)
}

func (s *SmartContract) AcceptCustodyTransfer(ctx contractapi.TransactionContextInterface, shipmentID string) error {
	ship, err := getShipmentByID(ctx, shipmentID)
	if err != nil {
		return err
	}
	caller, err := callerMSPID(ctx)
	if err != nil {
		return err
	}

	_, pending, err := findPendingCustody(ctx, shipmentID)
	if err != nil {
		return err
	}
	if pending == nil {
		return fmt.Errorf("no pending custody transfer for shipment %s", shipmentID)
	}
	if caller != pending.ToMSP {
		return fmt.Errorf("caller MSP %s is not the intended receiver %s for custody transfer", caller, pending.ToMSP)
	}

	now, err := txTimestamp(ctx)
	if err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()

	pending.Status = CustodyStatusCompleted
	pending.ReceiverSignature = txID
	pending.TransferredAt = now
	pending.UpdatedAt = now
	if err := putJSON(ctx, prefixCustody+pending.ID, pending); err != nil {
		return err
	}

	// Update shipment: new sender = receiver of this custody
	ship.SenderMSP = pending.ToMSP
	ship.SenderName = pending.ToName
	ship.UpdatedAt = now

	// Check if receiver is the final destination
	if pending.ToMSP == ship.ReceiverMSP {
		ship.Status = ShipmentStatusDelivered
		ship.ArrivalAt = now
		// Update all products: delivered
		for _, pid := range ship.ProductIDs {
			p, err := getProductByID(ctx, pid)
			if err != nil {
				return err
			}
			p.CurrentOwnerMSP = pending.ToMSP
			p.CurrentOwner = pending.ToName
			p.Status = ProductStatusDelivered
			p.CurrentShipmentID = ""
			p.UpdatedAt = now
			if err := putJSON(ctx, prefixProduct+pid, p); err != nil {
				return err
			}
		}
	} else {
		// Intermediate handoff — update products' owner but keep them SHIPPED
		for _, pid := range ship.ProductIDs {
			p, err := getProductByID(ctx, pid)
			if err != nil {
				return err
			}
			p.CurrentOwnerMSP = pending.ToMSP
			p.CurrentOwner = pending.ToName
			p.UpdatedAt = now
			if err := putJSON(ctx, prefixProduct+pid, p); err != nil {
				return err
			}
		}
	}

	return putJSON(ctx, prefixShipment+shipmentID, ship)
}

func (s *SmartContract) RejectCustodyTransfer(ctx contractapi.TransactionContextInterface, shipmentID string) error {
	_, err := getShipmentByID(ctx, shipmentID)
	if err != nil {
		return err
	}
	caller, err := callerMSPID(ctx)
	if err != nil {
		return err
	}

	_, pending, err := findPendingCustody(ctx, shipmentID)
	if err != nil {
		return err
	}
	if pending == nil {
		return fmt.Errorf("no pending custody transfer for shipment %s", shipmentID)
	}
	if caller != pending.ToMSP {
		return fmt.Errorf("caller MSP %s is not the intended receiver %s for custody transfer", caller, pending.ToMSP)
	}

	now, err := txTimestamp(ctx)
	if err != nil {
		return err
	}

	pending.Status = CustodyStatusRejected
	pending.UpdatedAt = now
	return putJSON(ctx, prefixCustody+pending.ID, pending)
}

func (s *SmartContract) GetCustodyChain(ctx contractapi.TransactionContextInterface, shipmentID string) ([]*CustodyRecord, error) {
	start := prefixCustody + shipmentID + "~"
	end := prefixCustody + shipmentID + "~\x7f"
	iter, err := ctx.GetStub().GetStateByRange(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get custody chain for shipment %s: %w", shipmentID, err)
	}
	defer iter.Close()

	var records []*CustodyRecord
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var cr CustodyRecord
		if err := json.Unmarshal(res.Value, &cr); err != nil {
			return nil, err
		}
		records = append(records, &cr)
	}
	if records == nil {
		records = make([]*CustodyRecord, 0)
	}
	return records, nil
}

// ── Event functions ───────────────────────────────────────────────────────────

func (s *SmartContract) LogEvent(ctx contractapi.TransactionContextInterface,
	id, targetID, targetType, eventType, description, location, occurredAt, dataJSON string) error {

	caller, err := callerMSPID(ctx)
	if err != nil {
		return err
	}
	now, err := txTimestamp(ctx)
	if err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()

	var data map[string]string
	if dataJSON != "" {
		if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
			return fmt.Errorf("invalid data JSON: %w", err)
		}
	}

	if occurredAt == "" {
		occurredAt = now
	}

	ev := SupplyChainEvent{
		DocType: "EVENT", ID: id, TargetID: targetID, TargetType: targetType,
		EventType: eventType, Description: description, Location: location,
		RecordedBy: caller, Data: data, OccurredAt: occurredAt,
		TxID: txID, CreatedAt: now,
	}
	// Key: EVENT~{targetID}~{txTimestamp}~{eventID}
	key := fmt.Sprintf("%s%s~%s~%s", prefixEvent, targetID, now, id)
	return putJSON(ctx, key, ev)
}

func (s *SmartContract) GetEvents(ctx contractapi.TransactionContextInterface, targetID string) ([]*SupplyChainEvent, error) {
	start := prefixEvent + targetID + "~"
	end := prefixEvent + targetID + "~\x7f"
	iter, err := ctx.GetStub().GetStateByRange(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for target %s: %w", targetID, err)
	}
	defer iter.Close()

	var events []*SupplyChainEvent
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var ev SupplyChainEvent
		if err := json.Unmarshal(res.Value, &ev); err != nil {
			return nil, err
		}
		events = append(events, &ev)
	}
	if events == nil {
		events = make([]*SupplyChainEvent, 0)
	}
	return events, nil
}

// ── Recall functions ──────────────────────────────────────────────────────────

func (s *SmartContract) IssueRecall(ctx contractapi.TransactionContextInterface,
	id, scope, targetIDsJSON, reason, severity, issuedByName, instructionsURL string) error {

	caller, err := callerMSPID(ctx)
	if err != nil {
		return err
	}
	now, err := txTimestamp(ctx)
	if err != nil {
		return err
	}

	var targetIDs []string
	if err := json.Unmarshal([]byte(targetIDsJSON), &targetIDs); err != nil {
		return fmt.Errorf("invalid targetIds JSON: %w", err)
	}
	if len(targetIDs) == 0 {
		return fmt.Errorf("recall must target at least one item")
	}

	affectedCount := 0

	switch scope {
	case RecallScopeProduct:
		for _, pid := range targetIDs {
			p, err := getProductByID(ctx, pid)
			if err != nil {
				return err
			}
			p.Status = ProductStatusRecalled
			p.RecallID = id
			p.UpdatedAt = now
			if err := putJSON(ctx, prefixProduct+pid, p); err != nil {
				return err
			}
			affectedCount++
		}
	case RecallScopeShipment:
		for _, sid := range targetIDs {
			ship, err := getShipmentByID(ctx, sid)
			if err != nil {
				return err
			}
			ship.Status = ShipmentStatusRecalled
			ship.RecallID = id
			ship.UpdatedAt = now
			if err := putJSON(ctx, prefixShipment+sid, ship); err != nil {
				return err
			}
			// Also recall all products in shipment
			for _, pid := range ship.ProductIDs {
				p, err := getProductByID(ctx, pid)
				if err != nil {
					return err
				}
				p.Status = ProductStatusRecalled
				p.RecallID = id
				p.UpdatedAt = now
				if err := putJSON(ctx, prefixProduct+pid, p); err != nil {
					return err
				}
				affectedCount++
			}
		}
	case RecallScopeBatch:
		// targetIDs are batch IDs — find all matching products
		for _, batchID := range targetIDs {
			products, err := s.GetProductsByBatch(ctx, batchID)
			if err != nil {
				return err
			}
			for _, p := range products {
				p.Status = ProductStatusRecalled
				p.RecallID = id
				p.UpdatedAt = now
				if err := putJSON(ctx, prefixProduct+p.ID, p); err != nil {
					return err
				}
				affectedCount++
			}
		}
	default:
		return fmt.Errorf("invalid recall scope %s; must be PRODUCT, SHIPMENT, or BATCH", scope)
	}

	recall := RecallNotice{
		DocType: "RECALL", ID: id, Scope: scope, TargetIDs: targetIDs,
		Reason: reason, IssuedBy: caller, IssuedByName: issuedByName,
		Severity: severity, InstructionsURL: instructionsURL,
		AffectedCount: affectedCount, CreatedAt: now,
	}
	return putJSON(ctx, prefixRecall+id, recall)
}

func (s *SmartContract) ReadRecall(ctx contractapi.TransactionContextInterface, id string) (*RecallNotice, error) {
	b, err := ctx.GetStub().GetState(prefixRecall + id)
	if err != nil {
		return nil, fmt.Errorf("failed to read recall %s: %w", id, err)
	}
	if b == nil {
		return nil, fmt.Errorf("recall %s does not exist", id)
	}
	var r RecallNotice
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("failed to unmarshal recall %s: %w", id, err)
	}
	return &r, nil
}

func (s *SmartContract) GetRecalledProducts(ctx contractapi.TransactionContextInterface) ([]*Product, error) {
	return s.GetProductsByStatus(ctx, ProductStatusRecalled)
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func collectHistoryForKey(ctx contractapi.TransactionContextInterface, key string) ([]*HistoryQueryResult, error) {
	iter, err := ctx.GetStub().GetHistoryForKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get history for key %s: %w", key, err)
	}
	defer iter.Close()

	results := make([]*HistoryQueryResult, 0)
	for iter.HasNext() {
		response, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var ts time.Time
		if response.Timestamp != nil {
			ts = response.Timestamp.AsTime()
		}
		var val interface{}
		if err := json.Unmarshal(response.Value, &val); err != nil {
			val = string(response.Value)
		}
		results = append(results, &HistoryQueryResult{
			TxID:      response.TxId,
			Timestamp: ts,
			IsDelete:  response.IsDelete,
			Value:     val,
		})
	}
	return results, nil
}

// ── POS functions ─────────────────────────────────────────────────────────────

func (s *SmartContract) CreateSale(ctx contractapi.TransactionContextInterface,
	id, customerID, cashierID, cashierName, itemsJSON, taxAmountStr, currency, notes string) error {

	caller, err := requireMSP(ctx, "Org3MSP")
	if err != nil {
		return err
	}

	existing, err := ctx.GetStub().GetState(prefixSale + id)
	if err != nil {
		return fmt.Errorf("failed to check sale %s: %w", id, err)
	}
	if existing != nil {
		return fmt.Errorf("sale %s already exists", id)
	}

	var items []SaleItem
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return fmt.Errorf("failed to unmarshal items: %w", err)
	}
	if len(items) == 0 {
		return fmt.Errorf("sale must have at least one item")
	}

	taxAmount, err := strconv.ParseFloat(taxAmountStr, 64)
	if err != nil {
		return fmt.Errorf("invalid taxAmount %q: %w", taxAmountStr, err)
	}

	now, err := txTimestamp(ctx)
	if err != nil {
		return err
	}

	var subTotal float64
	for i, item := range items {
		p, err := getProductByID(ctx, item.ProductID)
		if err != nil {
			return err
		}
		if p.Status != ProductStatusDelivered {
			return fmt.Errorf("product %s is not DELIVERED (status: %s)", item.ProductID, p.Status)
		}
		if p.CurrentOwnerMSP != "Org3MSP" {
			return fmt.Errorf("product %s is not owned by Org3MSP", item.ProductID)
		}
		if p.RecallID != "" {
			return fmt.Errorf("product %s is under recall %s", item.ProductID, p.RecallID)
		}

		p.Status = ProductStatusSold
		p.UpdatedAt = now
		if err := putJSON(ctx, prefixProduct+item.ProductID, p); err != nil {
			return err
		}

		items[i].SKU = p.SKU
		items[i].ProductName = p.Name
		subTotal += item.UnitPrice
	}

	if currency == "" {
		currency = "IDR"
	}

	sale := SaleTransaction{
		DocType:     "SALE",
		ID:          id,
		Items:       items,
		CustomerID:  customerID,
		CashierID:   cashierID,
		CashierName: cashierName,
		SubTotal:    subTotal,
		TaxAmount:   taxAmount,
		TotalAmount: subTotal + taxAmount,
		Currency:    currency,
		RetailerMSP: caller,
		Notes:       notes,
		TxID:        ctx.GetStub().GetTxID(),
		CreatedAt:   now,
	}
	return putJSON(ctx, prefixSale+id, sale)
}

func (s *SmartContract) ReadSale(ctx contractapi.TransactionContextInterface, id string) (*SaleTransaction, error) {
	b, err := ctx.GetStub().GetState(prefixSale + id)
	if err != nil {
		return nil, fmt.Errorf("failed to read sale %s: %w", id, err)
	}
	if b == nil {
		return nil, fmt.Errorf("sale %s does not exist", id)
	}
	var sale SaleTransaction
	if err := json.Unmarshal(b, &sale); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sale %s: %w", id, err)
	}
	return &sale, nil
}

func (s *SmartContract) GetAllSales(ctx contractapi.TransactionContextInterface, pageSizeStr, bookmark string) (*PagedSaleResult, error) {
	pageSize, err := strconv.ParseInt(pageSizeStr, 10, 32)
	if err != nil || pageSize <= 0 {
		pageSize = 20
	}

	iter, meta, err := ctx.GetStub().GetStateByRangeWithPagination(
		prefixSale, prefixSale+"\x7f", int32(pageSize), bookmark)
	if err != nil {
		return nil, fmt.Errorf("failed to query sales: %w", err)
	}
	defer iter.Close()

	var sales []*SaleTransaction
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var sale SaleTransaction
		if err := json.Unmarshal(res.Value, &sale); err != nil {
			return nil, err
		}
		sales = append(sales, &sale)
	}
	if sales == nil {
		sales = make([]*SaleTransaction, 0)
	}
	return &PagedSaleResult{Sales: sales, Bookmark: meta.Bookmark, Count: len(sales)}, nil
}

func (s *SmartContract) GetSalesByCustomer(ctx contractapi.TransactionContextInterface, customerID string) ([]*SaleTransaction, error) {
	query := fmt.Sprintf(`{"selector":{"docType":"SALE","customerId":%q},"use_index":["indexSaleByCustomerDoc","indexSaleByCustomer"]}`, customerID)
	iter, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer iter.Close()

	var sales []*SaleTransaction
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var sale SaleTransaction
		if err := json.Unmarshal(res.Value, &sale); err != nil {
			return nil, err
		}
		sales = append(sales, &sale)
	}
	if sales == nil {
		sales = make([]*SaleTransaction, 0)
	}
	return sales, nil
}

func (s *SmartContract) GetInventory(ctx contractapi.TransactionContextInterface, ownerMSP string) ([]*InventoryItem, error) {
	if ownerMSP == "" {
		ownerMSP = "Org3MSP"
	}
	query := fmt.Sprintf(`{"selector":{"docType":"PRODUCT","status":"DELIVERED","currentOwnerMSP":%q},"use_index":["indexByStatusAndOwnerDoc","indexByStatusAndOwner"]}`, ownerMSP)
	products, err := queryProducts(ctx, query)
	if err != nil {
		return nil, err
	}

	bySKU := make(map[string]*InventoryItem)
	var order []string
	for _, p := range products {
		if _, seen := bySKU[p.SKU]; !seen {
			bySKU[p.SKU] = &InventoryItem{SKU: p.SKU, Name: p.Name, ProductIDs: []string{}}
			order = append(order, p.SKU)
		}
		item := bySKU[p.SKU]
		item.Count++
		item.ProductIDs = append(item.ProductIDs, p.ID)
	}

	result := make([]*InventoryItem, 0, len(order))
	for _, sku := range order {
		result = append(result, bySKU[sku])
	}
	return result, nil
}

func main() {
	cc, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		log.Panicf("Error creating supply chain chaincode: %v", err)
	}
	if err := cc.Start(); err != nil {
		log.Panicf("Error starting supply chain chaincode: %v", err)
	}
}
