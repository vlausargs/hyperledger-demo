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
