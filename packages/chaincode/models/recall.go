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
