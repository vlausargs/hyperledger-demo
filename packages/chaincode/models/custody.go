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
