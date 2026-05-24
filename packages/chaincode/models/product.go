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
