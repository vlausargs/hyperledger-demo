# Phase 1: Codebase Restructure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the monolithic chaincode (1,313 lines) and API handlers (673+ lines) into domain-focused modules with clean architecture layers.

**Architecture:** Extract models, ledger operations, and validation into shared packages. Split single SmartContract struct into per-domain contracts. Add service layer between API handlers and Fabric Gateway. Add typed config module with startup validation.

**Tech Stack:** Go 1.23 (chaincode), Go 1.25 (API), Gin v1.12, Fabric Contract API v2, Fabric Gateway v1.11

**Spec:** `docs/superpowers/specs/2026-05-24-production-readiness-design.md`

---

## File Map

### Chaincode (new files from split of `chaincode.go`)

| File | Responsibility |
|------|---------------|
| `packages/chaincode/models/product.go` | Product, PagedProductResult structs + status constants |
| `packages/chaincode/models/shipment.go` | Shipment, PagedShipmentResult structs + status constants |
| `packages/chaincode/models/custody.go` | CustodyRecord struct + status constants |
| `packages/chaincode/models/event.go` | SupplyChainEvent struct |
| `packages/chaincode/models/recall.go` | RecallNotice struct + scope/severity constants |
| `packages/chaincode/models/sale.go` | SaleItem, SaleTransaction, InventoryItem, PagedSaleResult structs |
| `packages/chaincode/models/common.go` | HistoryQueryResult, ProvenanceResult, key prefix constants |
| `packages/chaincode/ledger/store.go` | PutJSON, GetByID, QueryBySelector, RangeQuery, History helpers |
| `packages/chaincode/validation/validate.go` | ID validation, required fields, status transition checks |
| `packages/chaincode/contracts/product.go` | ProductContract — all product chaincode functions |
| `packages/chaincode/contracts/shipment.go` | ShipmentContract — all shipment chaincode functions |
| `packages/chaincode/contracts/custody.go` | CustodyContract — custody transfer functions |
| `packages/chaincode/contracts/event.go` | EventContract — event logging functions |
| `packages/chaincode/contracts/recall.go` | RecallContract — recall management functions |
| `packages/chaincode/contracts/sale.go` | SaleContract — POS sale + inventory functions |
| `packages/chaincode/contracts/init.go` | InitContract — InitLedger seed function |
| `packages/chaincode/contracts/helpers.go` | callerMSPID, requireMSP, txTimestamp shared helpers |
| `packages/chaincode/main.go` | Registers all contracts, starts chaincode |

### API (new files from split of handlers/main)

| File | Responsibility |
|------|---------------|
| `packages/api/internal/config/config.go` | Typed config struct, env parsing, validation |
| `packages/api/pkg/response/response.go` | Standard JSON response helpers (OK, Created, Error, etc.) |
| `packages/api/internal/handler/product.go` | Product HTTP handlers |
| `packages/api/internal/handler/shipment.go` | Shipment + custody HTTP handlers |
| `packages/api/internal/handler/event.go` | Event HTTP handlers |
| `packages/api/internal/handler/recall.go` | Recall HTTP handlers |
| `packages/api/internal/handler/pos.go` | POS (sale, inventory, verify) HTTP handlers |
| `packages/api/internal/handler/network.go` | Channel, chaincode, transaction, peer, org handlers |
| `packages/api/internal/handler/auth.go` | Login handler |
| `packages/api/internal/handler/health.go` | Health check handler |
| `packages/api/internal/handler/types.go` | Request/response structs shared across handlers |
| `packages/api/internal/middleware/auth.go` | JWT middleware (moved from current) |
| `packages/api/internal/middleware/cors.go` | CORS middleware (extracted from main.go) |
| `packages/api/internal/middleware/logging.go` | Request logger + correlation ID (extracted from main.go) |
| `packages/api/internal/router/router.go` | All route registration in one place |
| `packages/api/cmd/server/main.go` | Entrypoint — config, gateway, router, server lifecycle |

### Files to keep (moved)

| Current | New Location |
|---------|-------------|
| `fabric-network/client/fabric/connector.go` | `packages/api/internal/fabric/connector.go` |
| `fabric-network/client/fabric/ca.go` | `packages/api/internal/fabric/ca.go` |
| `fabric-network/chaincode/basic/META-INF/` | `packages/chaincode/META-INF/` |

---

## Task 1: Create Directory Structure and Go Modules

**Files:**
- Create: `packages/chaincode/go.mod`
- Create: `packages/api/go.mod`
- Create: All directories listed in file map

- [ ] **Step 1: Create chaincode directory structure**

```bash
mkdir -p packages/chaincode/{models,ledger,validation,contracts}
mkdir -p packages/chaincode/META-INF/statedb/couchdb/indexes
```

- [ ] **Step 2: Create API directory structure**

```bash
mkdir -p packages/api/cmd/server
mkdir -p packages/api/internal/{config,handler,middleware,router,fabric}
mkdir -p packages/api/pkg/response
```

- [ ] **Step 3: Initialize chaincode go.mod**

Create `packages/chaincode/go.mod`:

```go
module github.com/myindo/hlf-supply-chain/chaincode

go 1.23

require github.com/hyperledger/fabric-contract-api-go/v2 v2.0.0
```

Then run:

```bash
cd packages/chaincode && go mod tidy
```

- [ ] **Step 4: Initialize API go.mod**

Create `packages/api/go.mod`:

```go
module github.com/myindo/hlf-supply-chain/api

go 1.25.10

require (
	github.com/gin-gonic/gin v1.12.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/hyperledger/fabric-ca v1.5.19
	github.com/hyperledger/fabric-gateway v1.11.0
	github.com/hyperledger/fabric-protos-go-apiv2 v0.3.7
	google.golang.org/grpc v1.81.0
	gopkg.in/yaml.v3 v3.0.1
)
```

Then run:

```bash
cd packages/api && go mod tidy
```

- [ ] **Step 5: Copy CouchDB indexes**

```bash
cp fabric-network/chaincode/basic/META-INF/statedb/couchdb/indexes/*.json packages/chaincode/META-INF/statedb/couchdb/indexes/
```

- [ ] **Step 6: Commit**

```bash
git add packages/
git commit -m "chore: scaffold packages directory structure for monorepo"
```

---

## Task 2: Extract Chaincode Models

Extract all struct definitions and constants from `fabric-network/chaincode/basic/chaincode.go` into separate model files.

**Files:**
- Create: `packages/chaincode/models/product.go`
- Create: `packages/chaincode/models/shipment.go`
- Create: `packages/chaincode/models/custody.go`
- Create: `packages/chaincode/models/event.go`
- Create: `packages/chaincode/models/recall.go`
- Create: `packages/chaincode/models/sale.go`
- Create: `packages/chaincode/models/common.go`

- [ ] **Step 1: Create `packages/chaincode/models/common.go`**

```go
package models

import "time"

const (
	PrefixProduct  = "PROD~"
	PrefixShipment = "SHIP~"
	PrefixCustody  = "CUSTODY~"
	PrefixEvent    = "EVENT~"
	PrefixRecall   = "RECALL~"
	PrefixSale     = "SALE~"
)

type HistoryQueryResult struct {
	TxID      string      `json:"txId"`
	Timestamp time.Time   `json:"timestamp"`
	IsDelete  bool        `json:"isDelete"`
	Value     interface{} `json:"value"`
}

type ProvenanceResult struct {
	Product *Product             `json:"product"`
	History []*HistoryQueryResult `json:"history"`
	Custody []*CustodyRecord     `json:"custody"`
	Events  []*SupplyChainEvent  `json:"events"`
}
```

- [ ] **Step 2: Create `packages/chaincode/models/product.go`**

```go
package models

const (
	ProductStatusActive    = "ACTIVE"
	ProductStatusShipped   = "SHIPPED"
	ProductStatusDelivered = "DELIVERED"
	ProductStatusRecalled  = "RECALLED"
	ProductStatusScrapped  = "SCRAPPED"
	ProductStatusSold      = "SOLD"
)

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

type PagedProductResult struct {
	Products []*Product `json:"products"`
	Bookmark string     `json:"bookmark"`
	Count    int        `json:"count"`
}
```

- [ ] **Step 3: Create `packages/chaincode/models/shipment.go`**

```go
package models

const (
	ShipmentStatusDraft     = "DRAFT"
	ShipmentStatusInTransit = "IN_TRANSIT"
	ShipmentStatusDelivered = "DELIVERED"
	ShipmentStatusRecalled  = "RECALLED"
	ShipmentStatusCancelled = "CANCELLED"
)

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

type PagedShipmentResult struct {
	Shipments []*Shipment `json:"shipments"`
	Bookmark  string      `json:"bookmark"`
	Count     int         `json:"count"`
}
```

- [ ] **Step 4: Create `packages/chaincode/models/custody.go`**

```go
package models

const (
	CustodyStatusPending   = "PENDING"
	CustodyStatusCompleted = "COMPLETED"
	CustodyStatusRejected  = "REJECTED"
)

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
```

- [ ] **Step 5: Create `packages/chaincode/models/event.go`**

```go
package models

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
```

- [ ] **Step 6: Create `packages/chaincode/models/recall.go`**

```go
package models

const (
	RecallScopeProduct  = "PRODUCT"
	RecallScopeShipment = "SHIPMENT"
	RecallScopeBatch    = "BATCH"

	RecallSeverityLow      = "LOW"
	RecallSeverityMedium   = "MEDIUM"
	RecallSeverityHigh     = "HIGH"
	RecallSeverityCritical = "CRITICAL"
)

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
```

- [ ] **Step 7: Create `packages/chaincode/models/sale.go`**

```go
package models

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

type PagedSaleResult struct {
	Sales    []*SaleTransaction `json:"sales"`
	Bookmark string             `json:"bookmark"`
	Count    int                `json:"count"`
}
```

- [ ] **Step 8: Verify models compile**

```bash
cd packages/chaincode && go build ./models/...
```

Expected: clean build, no errors.

- [ ] **Step 9: Commit**

```bash
git add packages/chaincode/models/
git commit -m "refactor: extract chaincode models into separate domain files"
```

---

## Task 3: Create Ledger Abstraction Layer

Extract all state read/write/query operations from chaincode.go into a reusable ledger package.

**Files:**
- Create: `packages/chaincode/ledger/store.go`
- Test: `packages/chaincode/ledger/store_test.go`

- [ ] **Step 1: Write test for PutJSON and GetByID**

Create `packages/chaincode/ledger/store_test.go`:

```go
package ledger_test

import (
	"encoding/json"
	"testing"

	"github.com/myindo/hlf-supply-chain/chaincode/ledger"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type testItem struct {
	DocType string `json:"docType"`
	ID      string `json:"id"`
	Name    string `json:"name"`
}

func newMockCtx() contractapi.TransactionContextInterface {
	stub := shimtest.NewMockStub("test", nil)
	stub.MockTransactionStart("tx1")
	ctx := new(contractapi.TransactionContext)
	ctx.SetStub(stub)
	return ctx
}

func TestPutJSONAndGetByID(t *testing.T) {
	ctx := newMockCtx()
	store := ledger.NewStore()

	item := testItem{DocType: "TEST", ID: "item1", Name: "Test Item"}
	err := store.PutJSON(ctx, "TEST~item1", item)
	if err != nil {
		t.Fatalf("PutJSON failed: %v", err)
	}

	var result testItem
	err = store.GetByID(ctx, "TEST~item1", &result)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if result.Name != "Test Item" {
		t.Fatalf("expected 'Test Item', got '%s'", result.Name)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	ctx := newMockCtx()
	store := ledger.NewStore()

	var result testItem
	err := store.GetByID(ctx, "TEST~nonexistent", &result)
	if err == nil {
		t.Fatal("expected error for nonexistent key")
	}
}

func TestExists(t *testing.T) {
	ctx := newMockCtx()
	store := ledger.NewStore()

	exists, err := store.Exists(ctx, "TEST~nope")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Fatal("expected false for nonexistent key")
	}

	item := testItem{DocType: "TEST", ID: "item1", Name: "exists"}
	_ = store.PutJSON(ctx, "TEST~item1", item)

	exists, err = store.Exists(ctx, "TEST~item1")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Fatal("expected true for existing key")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd packages/chaincode && go test ./ledger/... -v
```

Expected: FAIL — `ledger` package does not exist yet.

- [ ] **Step 3: Implement ledger store**

Create `packages/chaincode/ledger/store.go`:

```go
package ledger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type Store struct{}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) PutJSON(ctx contractapi.TransactionContextInterface, key string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", key, err)
	}
	return ctx.GetStub().PutState(key, b)
}

func (s *Store) GetByID(ctx contractapi.TransactionContextInterface, key string, result interface{}) error {
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", key, err)
	}
	if b == nil {
		return fmt.Errorf("%s does not exist", key)
	}
	return json.Unmarshal(b, result)
}

func (s *Store) Exists(ctx contractapi.TransactionContextInterface, key string) (bool, error) {
	b, err := ctx.GetStub().GetState(key)
	if err != nil {
		return false, fmt.Errorf("failed to read %s: %w", key, err)
	}
	return b != nil, nil
}

func (s *Store) GetByRange(ctx contractapi.TransactionContextInterface, startKey, endKey string, unmarshal func([]byte) error) error {
	iter, err := ctx.GetStub().GetStateByRange(startKey, endKey)
	if err != nil {
		return fmt.Errorf("range query failed: %w", err)
	}
	defer iter.Close()
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return err
		}
		if err := unmarshal(res.Value); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) QueryBySelector(ctx contractapi.TransactionContextInterface, query string, unmarshal func([]byte) error) error {
	iter, err := ctx.GetStub().GetQueryResult(query)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer iter.Close()
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return err
		}
		if err := unmarshal(res.Value); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetPaginated(ctx contractapi.TransactionContextInterface, startKey, endKey string, pageSize int32, bookmark string, unmarshal func([]byte) error) (string, error) {
	iter, meta, err := ctx.GetStub().GetStateByRangeWithPagination(startKey, endKey, pageSize, bookmark)
	if err != nil {
		return "", fmt.Errorf("paginated query failed: %w", err)
	}
	defer iter.Close()
	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return "", err
		}
		if err := unmarshal(res.Value); err != nil {
			return "", err
		}
	}
	return meta.Bookmark, nil
}

func (s *Store) GetHistory(ctx contractapi.TransactionContextInterface, key string) ([]HistoryEntry, error) {
	iter, err := ctx.GetStub().GetHistoryForKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get history for %s: %w", key, err)
	}
	defer iter.Close()

	var results []HistoryEntry
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
		results = append(results, HistoryEntry{
			TxID:      response.TxId,
			Timestamp: ts,
			IsDelete:  response.IsDelete,
			Value:     val,
		})
	}
	return results, nil
}

func CallerMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
	msp, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get caller MSPID: %w", err)
	}
	return msp, nil
}

func RequireMSP(ctx contractapi.TransactionContextInterface, allowed ...string) (string, error) {
	caller, err := CallerMSPID(ctx)
	if err != nil {
		return "", err
	}
	for _, a := range allowed {
		if caller == a {
			return caller, nil
		}
	}
	return "", fmt.Errorf("caller MSP %s is not authorized; allowed: %v", caller, allowed)
}

func TxTimestamp(ctx contractapi.TransactionContextInterface) (string, error) {
	ts, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return "", fmt.Errorf("failed to get tx timestamp: %w", err)
	}
	return ts.AsTime().UTC().Format(time.RFC3339), nil
}

type HistoryEntry struct {
	TxID      string      `json:"txId"`
	Timestamp time.Time   `json:"timestamp"`
	IsDelete  bool        `json:"isDelete"`
	Value     interface{} `json:"value"`
}
```

- [ ] **Step 4: Run tests**

```bash
cd packages/chaincode && go test ./ledger/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add packages/chaincode/ledger/
git commit -m "feat: add chaincode ledger abstraction layer"
```

---

## Task 4: Create Chaincode Validation Layer

**Files:**
- Create: `packages/chaincode/validation/validate.go`
- Test: `packages/chaincode/validation/validate_test.go`

- [ ] **Step 1: Write validation tests**

Create `packages/chaincode/validation/validate_test.go`:

```go
package validation_test

import (
	"testing"

	"github.com/myindo/hlf-supply-chain/chaincode/validation"
)

func TestValidateID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"PROD-001", true},
		{"abc_123", true},
		{"valid-id-with-tilde~001", true},
		{"", false},
		{"a", true},
		{string(make([]byte, 65)), false},
		{"invalid id with spaces", false},
		{"inject'; DROP TABLE--", false},
	}
	for _, tt := range tests {
		err := validation.ValidateID(tt.id)
		if tt.valid && err != nil {
			t.Errorf("ValidateID(%q) should be valid, got error: %v", tt.id, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ValidateID(%q) should be invalid, got nil", tt.id)
		}
	}
}

func TestValidateRequired(t *testing.T) {
	err := validation.ValidateRequired("name", "")
	if err == nil {
		t.Error("expected error for empty required field")
	}

	err = validation.ValidateRequired("name", "value")
	if err != nil {
		t.Errorf("expected nil for non-empty field, got: %v", err)
	}
}

func TestValidateStatusTransition(t *testing.T) {
	tests := []struct {
		from, to string
		valid    bool
	}{
		{"ACTIVE", "SHIPPED", true},
		{"SHIPPED", "DELIVERED", true},
		{"DELIVERED", "SOLD", true},
		{"ACTIVE", "RECALLED", true},
		{"SHIPPED", "ACTIVE", false},
		{"SOLD", "ACTIVE", false},
		{"DELIVERED", "SHIPPED", false},
	}
	for _, tt := range tests {
		err := validation.ValidateProductStatusTransition(tt.from, tt.to)
		if tt.valid && err != nil {
			t.Errorf("%s->%s should be valid, got: %v", tt.from, tt.to, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("%s->%s should be invalid", tt.from, tt.to)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd packages/chaincode && go test ./validation/... -v
```

Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement validation**

Create `packages/chaincode/validation/validate.go`:

```go
package validation

import (
	"fmt"
	"regexp"
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-~]{1,64}$`)

func ValidateID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid ID %q: must be 1-64 alphanumeric/underscore/hyphen/tilde characters", id)
	}
	return nil
}

func ValidateRequired(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

var productTransitions = map[string]map[string]bool{
	"ACTIVE":    {"SHIPPED": true, "RECALLED": true, "SCRAPPED": true},
	"SHIPPED":   {"DELIVERED": true, "RECALLED": true},
	"DELIVERED": {"SOLD": true, "RECALLED": true, "ACTIVE": true},
	"RECALLED":  {"SCRAPPED": true},
}

func ValidateProductStatusTransition(from, to string) error {
	allowed, exists := productTransitions[from]
	if !exists {
		return fmt.Errorf("unknown product status %q", from)
	}
	if !allowed[to] {
		return fmt.Errorf("invalid status transition: %s -> %s", from, to)
	}
	return nil
}

var shipmentTransitions = map[string]map[string]bool{
	"DRAFT":      {"IN_TRANSIT": true, "CANCELLED": true},
	"IN_TRANSIT": {"DELIVERED": true, "RECALLED": true},
}

func ValidateShipmentStatusTransition(from, to string) error {
	allowed, exists := shipmentTransitions[from]
	if !exists {
		return fmt.Errorf("unknown shipment status %q", from)
	}
	if !allowed[to] {
		return fmt.Errorf("invalid shipment status transition: %s -> %s", from, to)
	}
	return nil
}

func ValidateSelectorValue(value string) error {
	for _, ch := range value {
		if ch == '"' || ch == '\\' || ch == '{' || ch == '}' {
			return fmt.Errorf("invalid character %q in query parameter", ch)
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd packages/chaincode && go test ./validation/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add packages/chaincode/validation/
git commit -m "feat: add chaincode input validation with status transition checks"
```

---

## Task 5: Split Chaincode Contracts — Helpers and Product

**Files:**
- Create: `packages/chaincode/contracts/helpers.go`
- Create: `packages/chaincode/contracts/product.go`
- Test: `packages/chaincode/contracts/product_test.go`

- [ ] **Step 1: Create shared contract helpers**

Create `packages/chaincode/contracts/helpers.go`:

```go
package contracts

import (
	"github.com/myindo/hlf-supply-chain/chaincode/ledger"
)

var store = ledger.NewStore()
```

This file provides the shared `store` instance used by all contract files. Each contract imports `validation` directly where needed.

- [ ] **Step 2: Write product contract test**

Create `packages/chaincode/contracts/product_test.go`:

```go
package contracts_test

import (
	"encoding/json"
	"testing"

	"github.com/myindo/hlf-supply-chain/chaincode/contracts"
	"github.com/myindo/hlf-supply-chain/chaincode/models"
	"github.com/hyperledger/fabric-chaincode-go/v2/shim/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

func newTestCtx(mspID string) contractapi.TransactionContextInterface {
	stub := shimtest.NewMockStub("test", nil)
	stub.MockTransactionStart("tx1")
	ctx := new(contractapi.TransactionContext)
	ctx.SetStub(stub)
	// Note: MockStub doesn't support GetClientIdentity with MSPID out of the box.
	// For unit tests that check MSP, use integration tests with real peer.
	// These tests focus on data logic, not access control.
	return ctx
}

func TestProductContract_ReadProduct_NotFound(t *testing.T) {
	pc := new(contracts.ProductContract)
	ctx := newTestCtx("Org1MSP")
	_, err := pc.ReadProduct(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent product")
	}
}

func TestProductContract_ProductExists(t *testing.T) {
	pc := new(contracts.ProductContract)
	ctx := newTestCtx("Org1MSP")

	exists, err := pc.ProductExists(ctx, "PROD-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("product should not exist")
	}

	// Manually insert a product via stub
	p := models.Product{DocType: "PRODUCT", ID: "PROD-001", Name: "Test"}
	b, _ := json.Marshal(p)
	ctx.GetStub().(*shimtest.MockStub).State[models.PrefixProduct+"PROD-001"] = b

	exists, err = pc.ProductExists(ctx, "PROD-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("product should exist")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
cd packages/chaincode && go test ./contracts/... -v
```

Expected: FAIL — `contracts.ProductContract` does not exist.

- [ ] **Step 4: Implement ProductContract**

Create `packages/chaincode/contracts/product.go`:

```go
package contracts

import (
	"encoding/json"
	"fmt"

	"github.com/myindo/hlf-supply-chain/chaincode/ledger"
	"github.com/myindo/hlf-supply-chain/chaincode/models"
	"github.com/myindo/hlf-supply-chain/chaincode/validation"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type ProductContract struct {
	contractapi.Contract
}

func (c *ProductContract) CreateProduct(ctx contractapi.TransactionContextInterface,
	id, sku, name, description, batchID, manufacturerName, expiryDate, metaJSON string) error {

	if err := validation.ValidateID(id); err != nil {
		return err
	}
	caller, err := ledger.RequireMSP(ctx, "Org1MSP")
	if err != nil {
		return err
	}
	now, err := ledger.TxTimestamp(ctx)
	if err != nil {
		return err
	}
	exists, err := c.ProductExists(ctx, id)
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

	p := models.Product{
		DocType: "PRODUCT", ID: id, SKU: sku, Name: name, Description: description,
		BatchID: batchID, ManufacturerID: caller, ManufacturerName: manufacturerName,
		ManufacturedAt: now, ExpiryDate: expiryDate, Status: models.ProductStatusActive,
		CurrentOwnerMSP: caller, CurrentOwner: manufacturerName,
		Metadata: meta, CreatedAt: now, UpdatedAt: now,
	}
	return store.PutJSON(ctx, models.PrefixProduct+id, p)
}

func (c *ProductContract) ReadProduct(ctx contractapi.TransactionContextInterface, id string) (*models.Product, error) {
	var p models.Product
	if err := store.GetByID(ctx, models.PrefixProduct+id, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *ProductContract) UpdateProduct(ctx contractapi.TransactionContextInterface,
	id, name, description, expiryDate, metaJSON string) error {

	p, err := c.ReadProduct(ctx, id)
	if err != nil {
		return err
	}
	caller, err := ledger.CallerMSPID(ctx)
	if err != nil {
		return err
	}
	if caller != p.CurrentOwnerMSP {
		return fmt.Errorf("caller MSP %s is not the current owner of product %s", caller, id)
	}
	now, err := ledger.TxTimestamp(ctx)
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
	return store.PutJSON(ctx, models.PrefixProduct+id, p)
}

func (c *ProductContract) ProductExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	return store.Exists(ctx, models.PrefixProduct+id)
}

func (c *ProductContract) GetAllProducts(ctx contractapi.TransactionContextInterface, pageSize int, bookmark string) (*models.PagedProductResult, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var products []*models.Product
	bm, err := store.GetPaginated(ctx,
		models.PrefixProduct, models.PrefixProduct+"\x7f",
		int32(pageSize), bookmark,
		func(b []byte) error {
			var p models.Product
			if err := json.Unmarshal(b, &p); err != nil {
				return err
			}
			products = append(products, &p)
			return nil
		})
	if err != nil {
		return nil, err
	}
	if products == nil {
		products = make([]*models.Product, 0)
	}
	return &models.PagedProductResult{Products: products, Bookmark: bm, Count: len(products)}, nil
}

func (c *ProductContract) GetProductsByBatch(ctx contractapi.TransactionContextInterface, batchID string) ([]*models.Product, error) {
	if err := validation.ValidateSelectorValue(batchID); err != nil {
		return nil, err
	}
	q := fmt.Sprintf(`{"selector":{"docType":"PRODUCT","batchId":"%s"}}`, batchID)
	return c.queryProducts(ctx, q)
}

func (c *ProductContract) GetProductsByStatus(ctx contractapi.TransactionContextInterface, status string) ([]*models.Product, error) {
	if err := validation.ValidateSelectorValue(status); err != nil {
		return nil, err
	}
	q := fmt.Sprintf(`{"selector":{"docType":"PRODUCT","status":"%s"}}`, status)
	return c.queryProducts(ctx, q)
}

func (c *ProductContract) GetProductsByOwner(ctx contractapi.TransactionContextInterface, ownerMSP string) ([]*models.Product, error) {
	if err := validation.ValidateSelectorValue(ownerMSP); err != nil {
		return nil, err
	}
	q := fmt.Sprintf(`{"selector":{"docType":"PRODUCT","currentOwnerMSP":"%s"}}`, ownerMSP)
	return c.queryProducts(ctx, q)
}

func (c *ProductContract) GetProductHistory(ctx contractapi.TransactionContextInterface, id string) ([]ledger.HistoryEntry, error) {
	return store.GetHistory(ctx, models.PrefixProduct+id)
}

func (c *ProductContract) GetProvenance(ctx contractapi.TransactionContextInterface, productID string) (*models.ProvenanceResult, error) {
	p, err := c.ReadProduct(ctx, productID)
	if err != nil {
		return nil, err
	}

	history, err := c.GetProductHistory(ctx, productID)
	if err != nil {
		return nil, err
	}

	custodyContract := &CustodyContract{}
	custody := make([]*models.CustodyRecord, 0)
	if p.CurrentShipmentID != "" {
		custody, err = custodyContract.GetCustodyChain(ctx, p.CurrentShipmentID)
		if err != nil {
			return nil, err
		}
	}

	eventContract := &EventContract{}
	events, err := eventContract.GetEvents(ctx, productID)
	if err != nil {
		return nil, err
	}

	historyResults := make([]*models.HistoryQueryResult, len(history))
	for i, h := range history {
		historyResults[i] = &models.HistoryQueryResult{
			TxID: h.TxID, Timestamp: h.Timestamp, IsDelete: h.IsDelete, Value: h.Value,
		}
	}
	if historyResults == nil {
		historyResults = make([]*models.HistoryQueryResult, 0)
	}
	if events == nil {
		events = make([]*models.SupplyChainEvent, 0)
	}

	return &models.ProvenanceResult{Product: p, History: historyResults, Custody: custody, Events: events}, nil
}

func (c *ProductContract) queryProducts(ctx contractapi.TransactionContextInterface, query string) ([]*models.Product, error) {
	var products []*models.Product
	err := store.QueryBySelector(ctx, query, func(b []byte) error {
		var p models.Product
		if err := json.Unmarshal(b, &p); err != nil {
			return err
		}
		products = append(products, &p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if products == nil {
		products = make([]*models.Product, 0)
	}
	return products, nil
}
```

- [ ] **Step 5: Run tests**

```bash
cd packages/chaincode && go test ./contracts/... -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add packages/chaincode/contracts/helpers.go packages/chaincode/contracts/product.go packages/chaincode/contracts/product_test.go
git commit -m "feat: add ProductContract with validation and ledger layer"
```

---

## Task 6: Split Remaining Chaincode Contracts

Create ShipmentContract, CustodyContract, EventContract, RecallContract, SaleContract, and InitContract. Each mirrors the corresponding functions from the original `chaincode.go`.

**Files:**
- Create: `packages/chaincode/contracts/shipment.go`
- Create: `packages/chaincode/contracts/custody.go`
- Create: `packages/chaincode/contracts/event.go`
- Create: `packages/chaincode/contracts/recall.go`
- Create: `packages/chaincode/contracts/sale.go`
- Create: `packages/chaincode/contracts/init.go`

- [ ] **Step 1: Create ShipmentContract**

Create `packages/chaincode/contracts/shipment.go` — extract `CreateShipment`, `ReadShipment`, `DispatchShipment`, `ShipmentExists`, `GetAllShipments`, `GetShipmentsByStatus`, `GetShipmentHistory` from original. Use `store` and `ledger` package for all state operations. Use `validation.ValidateID` on input IDs. Use `models.Shipment` instead of inline struct. Follow same pattern as ProductContract.

Key function signatures:
```go
type ShipmentContract struct{ contractapi.Contract }

func (c *ShipmentContract) CreateShipment(ctx contractapi.TransactionContextInterface,
	id, name, description, receiverMSP, receiverName, origin, destination, productIDsJSON string) error

func (c *ShipmentContract) ReadShipment(ctx contractapi.TransactionContextInterface, id string) (*models.Shipment, error)

func (c *ShipmentContract) DispatchShipment(ctx contractapi.TransactionContextInterface, id string) error

func (c *ShipmentContract) ShipmentExists(ctx contractapi.TransactionContextInterface, id string) (bool, error)

func (c *ShipmentContract) GetAllShipments(ctx contractapi.TransactionContextInterface, pageSize int, bookmark string) (*models.PagedShipmentResult, error)

func (c *ShipmentContract) GetShipmentsByStatus(ctx contractapi.TransactionContextInterface, status string) ([]*models.Shipment, error)

func (c *ShipmentContract) GetShipmentHistory(ctx contractapi.TransactionContextInterface, id string) ([]ledger.HistoryEntry, error)
```

Port logic directly from `chaincode.go:561-721`, replacing:
- `getShipmentByID()` → `store.GetByID(ctx, models.PrefixShipment+id, &ship)`
- `getProductByID()` → `store.GetByID(ctx, models.PrefixProduct+pid, &p)`
- `putJSON()` → `store.PutJSON()`
- `callerMSPID()` → `ledger.CallerMSPID()`
- `txTimestamp()` → `ledger.TxTimestamp()`
- Inline `Shipment{}` → `models.Shipment{}`

- [ ] **Step 2: Create CustodyContract**

Create `packages/chaincode/contracts/custody.go` — extract `InitiateCustodyTransfer`, `AcceptCustodyTransfer`, `RejectCustodyTransfer`, `GetCustodyChain` from `chaincode.go:725-903`.

Key function signatures:
```go
type CustodyContract struct{ contractapi.Contract }

func (c *CustodyContract) InitiateCustodyTransfer(ctx contractapi.TransactionContextInterface,
	shipmentID, toMSP, toName, conditions string) error

func (c *CustodyContract) AcceptCustodyTransfer(ctx contractapi.TransactionContextInterface, shipmentID string) error

func (c *CustodyContract) RejectCustodyTransfer(ctx contractapi.TransactionContextInterface, shipmentID string) error

func (c *CustodyContract) GetCustodyChain(ctx contractapi.TransactionContextInterface, shipmentID string) ([]*models.CustodyRecord, error)

func (c *CustodyContract) findPendingCustody(ctx contractapi.TransactionContextInterface, shipmentID string) (int, *models.CustodyRecord, error)
```

Port logic from `chaincode.go:725-903` + `findPendingCustody` from line 315-342. `AcceptCustodyTransfer` also updates products and shipment — needs to read/write products via `store.GetByID`/`store.PutJSON`.

- [ ] **Step 3: Create EventContract**

Create `packages/chaincode/contracts/event.go` — extract `LogEvent`, `GetEvents` from `chaincode.go:907-967`.

```go
type EventContract struct{ contractapi.Contract }

func (c *EventContract) LogEvent(ctx contractapi.TransactionContextInterface,
	id, targetID, targetType, eventType, description, location, occurredAt, dataJSON string) error

func (c *EventContract) GetEvents(ctx contractapi.TransactionContextInterface, targetID string) ([]*models.SupplyChainEvent, error)
```

- [ ] **Step 4: Create RecallContract**

Create `packages/chaincode/contracts/recall.go` — extract `IssueRecall`, `ReadRecall`, `GetRecalledProducts` from `chaincode.go:971-1082`.

```go
type RecallContract struct{ contractapi.Contract }

func (c *RecallContract) IssueRecall(ctx contractapi.TransactionContextInterface,
	id, scope, targetIDsJSON, reason, severity, issuedByName, instructionsURL string) error

func (c *RecallContract) ReadRecall(ctx contractapi.TransactionContextInterface, id string) (*models.RecallNotice, error)

func (c *RecallContract) GetRecalledProducts(ctx contractapi.TransactionContextInterface) ([]*models.Product, error)
```

`IssueRecall` depends on `ProductContract.GetProductsByBatch` for BATCH scope recalls. Create instance: `pc := &ProductContract{}` and call `pc.GetProductsByBatch()`.

- [ ] **Step 5: Create SaleContract**

Create `packages/chaincode/contracts/sale.go` — extract `CreateSale`, `ReadSale`, `GetAllSales`, `GetSalesByCustomer`, `GetInventory` from `chaincode.go:1119-1302`.

```go
type SaleContract struct{ contractapi.Contract }

func (c *SaleContract) CreateSale(ctx contractapi.TransactionContextInterface,
	id, customerID, cashierID, cashierName, itemsJSON, taxAmountStr, currency, notes string) error

func (c *SaleContract) ReadSale(ctx contractapi.TransactionContextInterface, id string) (*models.SaleTransaction, error)

func (c *SaleContract) GetAllSales(ctx contractapi.TransactionContextInterface, pageSizeStr, bookmark string) (*models.PagedSaleResult, error)

func (c *SaleContract) GetSalesByCustomer(ctx contractapi.TransactionContextInterface, customerID string) ([]*models.SaleTransaction, error)

func (c *SaleContract) GetInventory(ctx contractapi.TransactionContextInterface, ownerMSP string) ([]*models.InventoryItem, error)
```

- [ ] **Step 6: Create InitContract**

Create `packages/chaincode/contracts/init.go` — extract `InitLedger` from `chaincode.go:346-382`.

```go
type InitContract struct{ contractapi.Contract }

func (c *InitContract) InitLedger(ctx contractapi.TransactionContextInterface) error
```

- [ ] **Step 7: Verify all contracts compile**

```bash
cd packages/chaincode && go build ./contracts/...
```

Expected: clean build.

- [ ] **Step 8: Commit**

```bash
git add packages/chaincode/contracts/
git commit -m "feat: split chaincode into domain-specific contracts"
```

---

## Task 7: Create Chaincode Main Entry Point

**Files:**
- Create: `packages/chaincode/main.go`

- [ ] **Step 1: Create main.go that registers all contracts**

Create `packages/chaincode/main.go`:

```go
package main

import (
	"log"

	"github.com/myindo/hlf-supply-chain/chaincode/contracts"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

func main() {
	cc, err := contractapi.NewChaincode(
		&contracts.InitContract{},
		&contracts.ProductContract{},
		&contracts.ShipmentContract{},
		&contracts.CustodyContract{},
		&contracts.EventContract{},
		&contracts.RecallContract{},
		&contracts.SaleContract{},
	)
	if err != nil {
		log.Panicf("Error creating supply chain chaincode: %v", err)
	}
	if err := cc.Start(); err != nil {
		log.Panicf("Error starting supply chain chaincode: %v", err)
	}
}
```

- [ ] **Step 2: Verify full chaincode builds**

```bash
cd packages/chaincode && go build .
```

Expected: clean build producing binary.

- [ ] **Step 3: Clean up build artifact**

```bash
rm -f packages/chaincode/chaincode
```

- [ ] **Step 4: Create `.gitignore` for chaincode**

Create `packages/chaincode/.gitignore`:

```
chaincode
```

- [ ] **Step 5: Commit**

```bash
git add packages/chaincode/main.go packages/chaincode/.gitignore
git commit -m "feat: add chaincode main with multi-contract registration"
```

---

## Task 8: Create API Config Module

**Files:**
- Create: `packages/api/internal/config/config.go`
- Test: `packages/api/internal/config/config_test.go`

- [ ] **Step 1: Write config tests**

Create `packages/api/internal/config/config_test.go`:

```go
package config_test

import (
	"os"
	"testing"

	"github.com/myindo/hlf-supply-chain/api/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long!!")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.ChannelID != "mychannel" {
		t.Errorf("expected default channel mychannel, got %s", cfg.ChannelID)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	os.Clearenv()

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET not set")
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	os.Clearenv()
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("CHANNEL_ID", "testchannel")
	os.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long!!")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.ChannelID != "testchannel" {
		t.Errorf("expected channel testchannel, got %s", cfg.ChannelID)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd packages/api && go test ./internal/config/... -v
```

Expected: FAIL

- [ ] **Step 3: Implement config module**

Create `packages/api/internal/config/config.go`:

```go
package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port              string
	Mode              string
	ChannelID         string
	ChaincodeID       string
	WalletPath        string
	TLSCertPath       string
	ConnectionProfile string
	CAURL             string
	CAName            string
	CAAdminMSPDir     string
	MSPID             string
	JWTSecret         string
	CORSAllowedOrigin string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:              env("SERVER_PORT", "8080"),
		Mode:              env("GIN_MODE", "debug"),
		ChannelID:         env("CHANNEL_ID", "mychannel"),
		ChaincodeID:       env("CHAINCODE_ID", "basic"),
		WalletPath:        env("WALLET_PATH", "./wallet"),
		TLSCertPath:       env("TLS_CERT_PATH", "./crypto"),
		ConnectionProfile: env("CONNECTION_PROFILE", "./crypto/connection-profile.yaml"),
		CAURL:             env("CA_URL", "http://localhost:8054"),
		CAName:            env("CA_NAME", "ca-org1"),
		CAAdminMSPDir:     env("CA_ADMIN_MSP_DIR", "./crypto/admin-msp"),
		MSPID:             env("MSP_ID", "Org1MSP"),
		JWTSecret:         env("JWT_SECRET", ""),
		CORSAllowedOrigin: env("CORS_ALLOWED_ORIGIN", "http://localhost:3000"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.ChannelID == "" {
		return fmt.Errorf("CHANNEL_ID is required")
	}
	if c.ChaincodeID == "" {
		return fmt.Errorf("CHAINCODE_ID is required")
	}
	return nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 4: Run tests**

```bash
cd packages/api && go test ./internal/config/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add packages/api/internal/config/
git commit -m "feat: add typed API config module with startup validation"
```

---

## Task 9: Create API Response Helpers

**Files:**
- Create: `packages/api/pkg/response/response.go`

- [ ] **Step 1: Create response helpers**

Create `packages/api/pkg/response/response.go`:

```go
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": msg})
}

func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, gin.H{"error": msg})
}

func Conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, gin.H{"error": msg})
}

func InternalError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
}

func Unavailable(c *gin.Context, msg string) {
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": msg})
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd packages/api && go build ./pkg/response/...
```

- [ ] **Step 3: Commit**

```bash
git add packages/api/pkg/response/
git commit -m "feat: add standard API response helpers"
```

---

## Task 10: Move Fabric Client Packages

**Files:**
- Copy: `fabric-network/client/fabric/connector.go` → `packages/api/internal/fabric/connector.go`
- Copy: `fabric-network/client/fabric/ca.go` → `packages/api/internal/fabric/ca.go`

- [ ] **Step 1: Copy fabric packages and update import paths**

```bash
cp fabric-network/client/fabric/connector.go packages/api/internal/fabric/connector.go
cp fabric-network/client/fabric/ca.go packages/api/internal/fabric/ca.go
```

- [ ] **Step 2: Update package path in both files**

In both files, the package declaration stays `package fabric` (unchanged). Update the module import path from `hlf-demo/fabric-network/client/fabric` to `github.com/myindo/hlf-supply-chain/api/internal/fabric` in any internal cross-references (there are none — these files don't import each other).

- [ ] **Step 3: Verify they compile**

```bash
cd packages/api && go build ./internal/fabric/...
```

If import path issues arise, run `go mod tidy` first.

- [ ] **Step 4: Commit**

```bash
git add packages/api/internal/fabric/
git commit -m "refactor: move fabric gateway and CA client to api/internal"
```

---

## Task 11: Split API Handlers by Domain

Extract from `fabric-network/client/rest/handlers.go` (673 lines) + `pos_handlers.go` + `auth.go` + `ca_handlers.go` into domain files.

**Files:**
- Create: `packages/api/internal/handler/types.go`
- Create: `packages/api/internal/handler/product.go`
- Create: `packages/api/internal/handler/shipment.go`
- Create: `packages/api/internal/handler/event.go`
- Create: `packages/api/internal/handler/recall.go`
- Create: `packages/api/internal/handler/pos.go`
- Create: `packages/api/internal/handler/network.go`
- Create: `packages/api/internal/handler/auth.go`
- Create: `packages/api/internal/handler/health.go`

- [ ] **Step 1: Create `packages/api/internal/handler/types.go`**

Move all request structs and the `FabricGateway` interface plus shared helpers (`evalJSON`, `paged`, `jsonMarshal`) here:

```go
package handler

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/myindo/hlf-supply-chain/api/internal/fabric"

	"github.com/hyperledger/fabric-gateway/pkg/client"
)

type FabricGateway interface {
	GetContract() *client.Contract
	SubmitTransaction(function string, args ...string) ([]byte, error)
	EvaluateTransaction(function string, args ...string) ([]byte, error)
	GetChannel() string
	GetChaincode() string
	GetConnectionProfile() *fabric.ConnectionProfile
}

type CreateProductRequest struct {
	ID               string            `json:"id"               binding:"required"`
	SKU              string            `json:"sku"              binding:"required"`
	Name             string            `json:"name"             binding:"required"`
	Description      string            `json:"description"`
	BatchID          string            `json:"batchId"          binding:"required"`
	ManufacturerName string            `json:"manufacturerName" binding:"required"`
	ExpiryDate       string            `json:"expiryDate"`
	Metadata         map[string]string `json:"metadata"`
}

type UpdateProductRequest struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	ExpiryDate  string            `json:"expiryDate"`
	Metadata    map[string]string `json:"metadata"`
}

type CreateShipmentRequest struct {
	ID           string   `json:"id"           binding:"required"`
	Name         string   `json:"name"         binding:"required"`
	Description  string   `json:"description"`
	ReceiverMSP  string   `json:"receiverMsp"  binding:"required"`
	ReceiverName string   `json:"receiverName" binding:"required"`
	Origin       string   `json:"origin"`
	Destination  string   `json:"destination"`
	ProductIDs   []string `json:"productIds"   binding:"required"`
}

type InitiateCustodyRequest struct {
	ToMSP      string `json:"toMsp"      binding:"required"`
	ToName     string `json:"toName"     binding:"required"`
	Conditions string `json:"conditions"`
}

type LogEventRequest struct {
	ID          string            `json:"id"          binding:"required"`
	TargetID    string            `json:"targetId"    binding:"required"`
	TargetType  string            `json:"targetType"  binding:"required"`
	EventType   string            `json:"eventType"   binding:"required"`
	Description string            `json:"description"`
	Location    string            `json:"location"`
	OccurredAt  string            `json:"occurredAt"`
	Data        map[string]string `json:"data"`
}

type IssueRecallRequest struct {
	ID              string   `json:"id"              binding:"required"`
	Scope           string   `json:"scope"           binding:"required"`
	TargetIDs       []string `json:"targetIds"       binding:"required"`
	Reason          string   `json:"reason"          binding:"required"`
	Severity        string   `json:"severity"        binding:"required"`
	IssuedByName    string   `json:"issuedByName"    binding:"required"`
	InstructionsURL string   `json:"instructionsUrl"`
}

type CreateSaleRequest struct {
	ID         string          `json:"id"         binding:"required"`
	CustomerID string          `json:"customerId"`
	CashierID  string          `json:"cashierId"  binding:"required"`
	CashierName string         `json:"cashierName" binding:"required"`
	Items      json.RawMessage `json:"items"      binding:"required"`
	TaxAmount  float64         `json:"taxAmount"`
	Currency   string          `json:"currency"`
	Notes      string          `json:"notes"`
}

func evalJSON(gw FabricGateway, fn string, args ...string) (json.RawMessage, error) {
	b, err := gw.EvaluateTransaction(fn, args...)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return json.RawMessage("null"), nil
	}
	return b, nil
}

func paged(pageSize, bookmark string) (string, string) {
	n, err := strconv.Atoi(pageSize)
	if err != nil || n <= 0 {
		n = 20
	}
	if n > 100 {
		n = 100
	}
	return strconv.Itoa(n), bookmark
}

func jsonMarshal(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal failed: %w", err)
	}
	return string(b), nil
}
```

- [ ] **Step 2: Create domain handler files**

Create each handler file following the same pattern as the original `handlers.go`, but now:
- Use `response.OK()`, `response.BadRequest()`, etc. from the response package
- Each file contains only handlers for its domain
- `product.go` gets: `GetAllProducts`, `GetProduct`, `CreateProduct`, `UpdateProduct`, `GetProductHistory`, `GetProductProvenance`, `GetProductsByBatch`, `GetProductsByStatus`
- `shipment.go` gets: `GetAllShipments`, `GetShipment`, `CreateShipment`, `DispatchShipment`, `GetShipmentHistory`, `GetShipmentsByStatus`, `GetCustodyChain`, `InitiateCustodyTransfer`, `AcceptCustodyTransfer`, `RejectCustodyTransfer`
- `event.go` gets: `LogEvent`, `GetEvents`
- `recall.go` gets: `IssueRecall`, `ReadRecall`, `GetRecalledProducts`
- `pos.go` gets: `CreateSale`, `GetAllSales`, `GetSale`, `GetInventory`, `VerifyProduct`
- `network.go` gets: `GetChannels`, `GetChannelInfo`, `GetChaincodes`, `GetChaincodeInfo`, `GetPeers`, `GetOrganizations`, `GetConnectionProfile`, `GetTransactions`, `GetTransaction`
- `auth.go` gets: `Login`
- `health.go` gets: `HealthCheck`

Port the logic 1:1 from the originals. The handler functions stay the same — only the file organization changes.

- [ ] **Step 3: Verify handlers compile**

```bash
cd packages/api && go build ./internal/handler/...
```

- [ ] **Step 4: Commit**

```bash
git add packages/api/internal/handler/
git commit -m "refactor: split API handlers into domain-specific files"
```

---

## Task 12: Create API Middleware and Router

**Files:**
- Create: `packages/api/internal/middleware/auth.go`
- Create: `packages/api/internal/middleware/cors.go`
- Create: `packages/api/internal/middleware/logging.go`
- Create: `packages/api/internal/router/router.go`

- [ ] **Step 1: Move middleware files**

Copy from current codebase and update imports:

- `fabric-network/client/middleware/auth.go` → `packages/api/internal/middleware/auth.go` (update import path)
- Extract `corsMiddleware()` from `main.go:281-296` → `packages/api/internal/middleware/cors.go`
- Extract `correlationIDMiddleware()` + `requestLogger()` from `main.go:390-430` → `packages/api/internal/middleware/logging.go`

- [ ] **Step 2: Create router.go**

Create `packages/api/internal/router/router.go` — extract all route registration from `main.go:80-191`:

```go
package router

import (
	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
	"github.com/myindo/hlf-supply-chain/api/internal/handler"
	"github.com/myindo/hlf-supply-chain/api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, gw handler.FabricGateway, caClient *fabric.CAClient, corsOrigin string) {
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(corsOrigin))
	r.Use(middleware.CorrelationID())
	r.Use(middleware.RequestLogger())

	r.GET("/health", handler.HealthCheck(gw))
	r.POST("/api/v1/auth/login", handler.Login())

	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWT())
	{
		products := v1.Group("/products")
		{
			products.GET("", handler.GetAllProducts(gw))
			products.POST("", handler.CreateProduct(gw))
			products.GET("/batch/:batchId", handler.GetProductsByBatch(gw))
			products.GET("/status/:status", handler.GetProductsByStatus(gw))
			products.GET("/:id", handler.GetProduct(gw))
			products.PUT("/:id", handler.UpdateProduct(gw))
			products.GET("/:id/history", handler.GetProductHistory(gw))
			products.GET("/:id/provenance", handler.GetProductProvenance(gw))
		}

		shipments := v1.Group("/shipments")
		{
			shipments.GET("", handler.GetAllShipments(gw))
			shipments.POST("", handler.CreateShipment(gw))
			shipments.GET("/status/:status", handler.GetShipmentsByStatus(gw))
			shipments.GET("/:id", handler.GetShipment(gw))
			shipments.POST("/:id/dispatch", handler.DispatchShipment(gw))
			shipments.GET("/:id/history", handler.GetShipmentHistory(gw))
			shipments.GET("/:id/custody", handler.GetCustodyChain(gw))
			shipments.POST("/:id/custody/initiate", handler.InitiateCustodyTransfer(gw))
			shipments.POST("/:id/custody/accept", handler.AcceptCustodyTransfer(gw))
			shipments.POST("/:id/custody/reject", handler.RejectCustodyTransfer(gw))
		}

		events := v1.Group("/events")
		{
			events.POST("", handler.LogEvent(gw))
			events.GET("/:targetId", handler.GetEvents(gw))
		}

		recalls := v1.Group("/recalls")
		{
			recalls.POST("", handler.IssueRecall(gw))
			recalls.GET("/products", handler.GetRecalledProducts(gw))
			recalls.GET("/:id", handler.ReadRecall(gw))
		}

		pos := v1.Group("/pos")
		{
			pos.POST("/sales", handler.CreateSale(gw))
			pos.GET("/sales", handler.GetAllSales(gw))
			pos.GET("/sales/:id", handler.GetSale(gw))
			pos.GET("/inventory", handler.GetInventory(gw))
			pos.GET("/verify/:id", handler.VerifyProduct(gw))
		}

		channels := v1.Group("/channels")
		{
			channels.GET("", handler.GetChannels(gw))
			channels.GET("/:channelId", handler.GetChannelInfo(gw))
		}

		chaincodes := v1.Group("/chaincodes")
		{
			chaincodes.GET("", handler.GetChaincodes(gw))
			chaincodes.GET("/:chaincodeId", handler.GetChaincodeInfo(gw))
		}

		transactions := v1.Group("/transactions")
		{
			transactions.GET("", handler.GetTransactions(gw))
			transactions.GET("/:txId", handler.GetTransaction(gw))
		}

		network := v1.Group("/network")
		{
			network.GET("/peers", handler.GetPeers(gw))
			network.GET("/organizations", handler.GetOrganizations(gw))
			network.GET("/connection-profile", handler.GetConnectionProfile(gw))
		}

		if caClient != nil {
			identities := v1.Group("/identities")
			{
				identities.GET("", handler.GetIdentities(caClient))
				identities.GET("/:id", handler.GetIdentity(caClient))
				identities.POST("/register", handler.RegisterIdentity(caClient))
				identities.POST("/enroll", handler.EnrollIdentity(caClient, ""))
				identities.DELETE("/:id", handler.DeleteIdentity(caClient, ""))
			}
		}
	}
}
```

- [ ] **Step 3: Verify middleware and router compile**

```bash
cd packages/api && go build ./internal/middleware/... ./internal/router/...
```

- [ ] **Step 4: Commit**

```bash
git add packages/api/internal/middleware/ packages/api/internal/router/
git commit -m "refactor: extract middleware and centralize route registration"
```

---

## Task 13: Create API Entry Point

**Files:**
- Create: `packages/api/cmd/server/main.go`

- [ ] **Step 1: Create slim main.go**

Create `packages/api/cmd/server/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/myindo/hlf-supply-chain/api/internal/config"
	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
	"github.com/myindo/hlf-supply-chain/api/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("starting fabric client application")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	gw, err := fabric.NewGateway(
		cfg.ChannelID, cfg.ChaincodeID,
		cfg.WalletPath, cfg.TLSCertPath, cfg.ConnectionProfile,
	)
	if err != nil {
		slog.Error("failed to initialize fabric gateway", "error", err)
		os.Exit(1)
	}
	defer gw.Close()
	slog.Info("connected to fabric gateway")

	caClient, err := fabric.NewCAClient(
		cfg.CAURL, cfg.CAName, cfg.CAAdminMSPDir, cfg.MSPID, cfg.WalletPath,
	)
	if err != nil {
		slog.Warn("CA client not available", "error", err)
		caClient = nil
	}

	r := gin.New()
	router.Setup(r, gw, caClient, cfg.CORSAllowedOrigin)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
```

- [ ] **Step 2: Verify full API builds**

```bash
cd packages/api && go build ./cmd/server/...
```

- [ ] **Step 3: Create `.gitignore` for API**

Create `packages/api/.gitignore`:

```
server
```

- [ ] **Step 4: Commit**

```bash
git add packages/api/cmd/ packages/api/.gitignore
git commit -m "feat: add slim API entrypoint with config-driven startup"
```

---

## Task 14: Create Root Makefile

**Files:**
- Create: `Makefile`

- [ ] **Step 1: Create Makefile**

```makefile
.PHONY: build-chaincode build-api test-chaincode test-api test lint clean

build-chaincode:
	cd packages/chaincode && go build -o ../../bin/chaincode .

build-api:
	cd packages/api && go build -o ../../bin/server ./cmd/server

build: build-chaincode build-api

test-chaincode:
	cd packages/chaincode && go test ./... -v -race -cover

test-api:
	cd packages/api && go test ./... -v -race -cover

test: test-chaincode test-api

lint-chaincode:
	cd packages/chaincode && go vet ./...

lint-api:
	cd packages/api && go vet ./...

lint: lint-chaincode lint-api

clean:
	rm -f bin/chaincode bin/server
```

- [ ] **Step 2: Create bin directory in .gitignore**

Add to root `.gitignore`:

```
/bin/
```

- [ ] **Step 3: Verify make targets work**

```bash
make lint
make test-chaincode
```

- [ ] **Step 4: Commit**

```bash
git add Makefile .gitignore
git commit -m "chore: add root Makefile for unified build/test/lint commands"
```

---

## Task 15: Verify Full Build and Run Tests

- [ ] **Step 1: Run full build**

```bash
make build
```

Expected: both `bin/chaincode` and `bin/server` built without errors.

- [ ] **Step 2: Run all tests**

```bash
make test
```

Expected: all tests pass.

- [ ] **Step 3: Run linters**

```bash
make lint
```

Expected: no issues.

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "chore: phase 1 restructure complete — verify build and tests"
```

---

## Notes for Implementer

1. **The original `fabric-network/` code stays untouched** during this phase. It remains the working deployment. The new `packages/` structure is built alongside it. Migration of deployment scripts to use the new structure is Phase 6.

2. **Go module paths**: The chaincode module is `github.com/myindo/hlf-supply-chain/chaincode` and the API is `github.com/myindo/hlf-supply-chain/api`. These are independent modules — no `replace` directives needed since they don't import each other.

3. **The `shimtest` package** (`github.com/hyperledger/fabric-chaincode-go/v2/shim/shimtest`) is used for chaincode unit tests. If it's not available in the v2 contract API, use `github.com/hyperledger/fabric-chaincode-go/v2/shimtest` or create minimal mock implementations of `ChaincodeStubInterface`.

4. **Task 6 (remaining contracts)** is the largest task. Each contract file follows the same pattern — port functions from `chaincode.go`, replace inline state operations with `store.*` calls, replace inline helper calls with `ledger.*` calls, add `validation.ValidateID()` on input IDs.

5. **CA handler imports**: The `handler.GetIdentities`, `handler.EnrollIdentity` etc. in router.go take `*fabric.CAClient` as argument. The handler functions for CA operations should be in `handler/identity.go` (not listed separately since they're a direct port of `ca_handlers.go`).
