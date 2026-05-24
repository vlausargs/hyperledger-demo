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
