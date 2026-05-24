package contracts

import (
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"github.com/myindo/hlf-supply-chain/chaincode/ledger"
	"github.com/myindo/hlf-supply-chain/chaincode/models"
)

type InitContract struct {
	contractapi.Contract
}

func (c *InitContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	caller, err := ledger.CallerMSPID(ctx)
	if err != nil {
		return err
	}
	now, err := ledger.TxTimestamp(ctx)
	if err != nil {
		return err
	}

	products := []models.Product{
		{
			DocType: "PRODUCT", ID: "PROD-001", SKU: "SKU-A100", Name: "Widget Alpha",
			Description: "Industrial widget", BatchID: "BATCH-2026-01",
			ManufacturerID: caller, ManufacturerName: "Acme Manufacturing",
			ManufacturedAt: now, Status: models.ProductStatusActive,
			CurrentOwnerMSP: caller, CurrentOwner: "Acme Manufacturing",
			Metadata: map[string]string{"weight_kg": "1.5", "material": "steel"},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			DocType: "PRODUCT", ID: "PROD-002", SKU: "SKU-B200", Name: "Widget Beta",
			Description: "Consumer widget", BatchID: "BATCH-2026-01",
			ManufacturerID: caller, ManufacturerName: "Acme Manufacturing",
			ManufacturedAt: now, Status: models.ProductStatusActive,
			CurrentOwnerMSP: caller, CurrentOwner: "Acme Manufacturing",
			Metadata: map[string]string{"weight_kg": "0.8", "material": "plastic"},
			CreatedAt: now, UpdatedAt: now,
		},
	}
	for _, p := range products {
		if err := store.PutJSON(ctx, models.PrefixProduct+p.ID, p); err != nil {
			return err
		}
	}
	return nil
}
