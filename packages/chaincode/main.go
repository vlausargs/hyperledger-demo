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
