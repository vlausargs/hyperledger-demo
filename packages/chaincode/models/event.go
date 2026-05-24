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
