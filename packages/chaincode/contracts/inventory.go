package contracts

import (
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"github.com/myindo/hlf-supply-chain/chaincode/models"
	"github.com/myindo/hlf-supply-chain/chaincode/validation"
)

type InventoryContract struct {
	contractapi.Contract
}

func (c *InventoryContract) GetInventory(ctx contractapi.TransactionContextInterface, ownerMSP string) ([]*models.InventoryItem, error) {
	if ownerMSP == "" {
		ownerMSP = "Org3MSP"
	}
	if err := validation.ValidateSelectorValue(ownerMSP); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`{"selector":{"docType":"PRODUCT","status":"DELIVERED","currentOwnerMSP":"%s"},"use_index":["indexByStatusAndOwnerDoc","indexByStatusAndOwner"]}`, ownerMSP)

	pc := &ProductContract{}
	products, err := pc.queryProducts(ctx, query)
	if err != nil {
		return nil, err
	}

	bySKU := make(map[string]*models.InventoryItem)
	var order []string
	for _, p := range products {
		if _, seen := bySKU[p.SKU]; !seen {
			bySKU[p.SKU] = &models.InventoryItem{SKU: p.SKU, Name: p.Name, ProductIDs: []string{}}
			order = append(order, p.SKU)
		}
		item := bySKU[p.SKU]
		item.Count++
		item.ProductIDs = append(item.ProductIDs, p.ID)
	}

	result := make([]*models.InventoryItem, 0, len(order))
	for _, sku := range order {
		result = append(result, bySKU[sku])
	}
	return result, nil
}
