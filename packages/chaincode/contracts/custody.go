package contracts

import (
	"github.com/myindo/hlf-supply-chain/chaincode/models"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type CustodyContract struct {
	contractapi.Contract
}

func (c *CustodyContract) GetCustodyChain(ctx contractapi.TransactionContextInterface, shipmentID string) ([]*models.CustodyRecord, error) {
	// Will be fully implemented in Task 6
	return make([]*models.CustodyRecord, 0), nil
}
