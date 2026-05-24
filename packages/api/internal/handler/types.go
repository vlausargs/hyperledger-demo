package handler

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
)

// FabricGateway interface for interacting with Fabric
type FabricGateway interface {
	GetContract() *client.Contract
	SubmitTransaction(function string, args ...string) ([]byte, error)
	EvaluateTransaction(function string, args ...string) ([]byte, error)
	GetChannel() string
	GetChaincode() string
	GetConnectionProfile() *fabric.ConnectionProfile
}

// ── Product request types ────────────────────────────────────────────────────

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

// ── Shipment request types ───────────────────────────────────────────────────

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

// ── Event request types ──────────────────────────────────────────────────────

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

// ── Recall request types ─────────────────────────────────────────────────────

type IssueRecallRequest struct {
	ID              string   `json:"id"              binding:"required"`
	Scope           string   `json:"scope"           binding:"required"`
	TargetIDs       []string `json:"targetIds"       binding:"required"`
	Reason          string   `json:"reason"          binding:"required"`
	Severity        string   `json:"severity"        binding:"required"`
	IssuedByName    string   `json:"issuedByName"    binding:"required"`
	InstructionsURL string   `json:"instructionsUrl"`
}

// ── POS request types ────────────────────────────────────────────────────────

type SaleItemRequest struct {
	ProductID   string  `json:"productId"   binding:"required"`
	SKU         string  `json:"sku"`
	ProductName string  `json:"productName"`
	UnitPrice   float64 `json:"unitPrice"   binding:"required"`
}

type CreateSaleRequest struct {
	ID          string            `json:"id"          binding:"required"`
	CustomerID  string            `json:"customerId"  binding:"required"`
	CashierID   string            `json:"cashierId"   binding:"required"`
	CashierName string            `json:"cashierName" binding:"required"`
	Items       []SaleItemRequest `json:"items"       binding:"required,min=1"`
	TaxAmount   float64           `json:"taxAmount"`
	Currency    string            `json:"currency"`
	Notes       string            `json:"notes"`
}

// ── Identity request types ───────────────────────────────────────────────────

type RegisterRequest struct {
	Name        string `json:"name"        binding:"required"`
	Type        string `json:"type"`
	Affiliation string `json:"affiliation"`
	Secret      string `json:"secret"`
	MaxEnroll   int    `json:"maxEnrollments"`
}

type EnrollRequest struct {
	Name   string `json:"name"   binding:"required"`
	Secret string `json:"secret" binding:"required"`
}

// ── Auth request types ───────────────────────────────────────────────────────

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// ── Validation helpers ───────────────────────────────────────────────────────

var validID = regexp.MustCompile(`^[a-zA-Z0-9_\-~]{1,128}$`)

func validateID(kind, id string) error {
	if id == "" {
		return fmt.Errorf("%s ID required", kind)
	}
	if !validID.MatchString(id) {
		return fmt.Errorf("invalid %s ID: alphanumeric, underscore, hyphen, tilde only, max 128 chars", kind)
	}
	return nil
}

func validateProductID(id string) error  { return validateID("product", id) }
func validateSaleID(id string) error     { return validateID("sale", id) }

// ── Shared helpers ───────────────────────────────────────────────────────────

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

func paged(c *gin.Context) (string, string) {
	ps := c.DefaultQuery("pageSize", "20")
	bm := c.DefaultQuery("bookmark", "")
	n, err := strconv.Atoi(ps)
	if err != nil || n <= 0 {
		n = 20
	}
	if n > 100 {
		n = 100
	}
	return strconv.Itoa(n), bm
}

func jsonMarshal(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal failed: %w", err)
	}
	return string(b), nil
}
