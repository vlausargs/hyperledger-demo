package contracts

import (
	"github.com/myindo/hlf-supply-chain/chaincode/models"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type EventContract struct {
	contractapi.Contract
}

func (c *EventContract) GetEvents(ctx contractapi.TransactionContextInterface, targetID string) ([]*models.SupplyChainEvent, error) {
	// Will be fully implemented in Task 6
	return make([]*models.SupplyChainEvent, 0), nil
}
