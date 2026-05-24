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
