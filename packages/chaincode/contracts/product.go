package contracts

import (
	"encoding/json"
	"fmt"

	"github.com/myindo/hlf-supply-chain/chaincode/ledger"
	"github.com/myindo/hlf-supply-chain/chaincode/models"
	"github.com/myindo/hlf-supply-chain/chaincode/validation"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type ProductContract struct {
	contractapi.Contract
}

func (c *ProductContract) CreateProduct(ctx contractapi.TransactionContextInterface,
	id, sku, name, description, batchID, manufacturerName, expiryDate, metaJSON string) error {

	if err := validation.ValidateID(id); err != nil {
		return err
	}
	caller, err := ledger.RequireMSP(ctx, "Org1MSP")
	if err != nil {
		return err
	}
	now, err := ledger.TxTimestamp(ctx)
	if err != nil {
		return err
	}
	exists, err := c.ProductExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("product %s already exists", id)
	}

	var meta map[string]string
	if metaJSON != "" {
		if err := json.Unmarshal([]byte(metaJSON), &meta); err != nil {
			return fmt.Errorf("invalid metadata JSON: %w", err)
		}
	}

	p := models.Product{
		DocType: "PRODUCT", ID: id, SKU: sku, Name: name, Description: description,
		BatchID: batchID, ManufacturerID: caller, ManufacturerName: manufacturerName,
		ManufacturedAt: now, ExpiryDate: expiryDate, Status: models.ProductStatusActive,
		CurrentOwnerMSP: caller, CurrentOwner: manufacturerName,
		Metadata: meta, CreatedAt: now, UpdatedAt: now,
	}
	return store.PutJSON(ctx, models.PrefixProduct+id, p)
}

func (c *ProductContract) ReadProduct(ctx contractapi.TransactionContextInterface, id string) (*models.Product, error) {
	var p models.Product
	if err := store.GetByID(ctx, models.PrefixProduct+id, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *ProductContract) UpdateProduct(ctx contractapi.TransactionContextInterface,
	id, name, description, expiryDate, metaJSON string) error {

	p, err := c.ReadProduct(ctx, id)
	if err != nil {
		return err
	}
	caller, err := ledger.CallerMSPID(ctx)
	if err != nil {
		return err
	}
	if caller != p.CurrentOwnerMSP {
		return fmt.Errorf("caller MSP %s is not the current owner of product %s", caller, id)
	}
	now, err := ledger.TxTimestamp(ctx)
	if err != nil {
		return err
	}

	if name != "" {
		p.Name = name
	}
	if description != "" {
		p.Description = description
	}
	if expiryDate != "" {
		p.ExpiryDate = expiryDate
	}
	if metaJSON != "" {
		var meta map[string]string
		if err := json.Unmarshal([]byte(metaJSON), &meta); err != nil {
			return fmt.Errorf("invalid metadata JSON: %w", err)
		}
		p.Metadata = meta
	}
	p.UpdatedAt = now
	return store.PutJSON(ctx, models.PrefixProduct+id, p)
}

func (c *ProductContract) ProductExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	return store.Exists(ctx, models.PrefixProduct+id)
}

func (c *ProductContract) GetAllProducts(ctx contractapi.TransactionContextInterface, pageSize int, bookmark string) (*models.PagedProductResult, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var products []*models.Product
	bm, err := store.GetPaginated(ctx,
		models.PrefixProduct, models.PrefixProduct+"\x7f",
		int32(pageSize), bookmark,
		func(b []byte) error {
			var p models.Product
			if err := json.Unmarshal(b, &p); err != nil {
				return err
			}
			products = append(products, &p)
			return nil
		})
	if err != nil {
		return nil, err
	}
	if products == nil {
		products = make([]*models.Product, 0)
	}
	return &models.PagedProductResult{Products: products, Bookmark: bm, Count: len(products)}, nil
}

func (c *ProductContract) GetProductsByBatch(ctx contractapi.TransactionContextInterface, batchID string) ([]*models.Product, error) {
	if err := validation.ValidateSelectorValue(batchID); err != nil {
		return nil, err
	}
	q := fmt.Sprintf(`{"selector":{"docType":"PRODUCT","batchId":"%s"}}`, batchID)
	return c.queryProducts(ctx, q)
}

func (c *ProductContract) GetProductsByStatus(ctx contractapi.TransactionContextInterface, status string) ([]*models.Product, error) {
	if err := validation.ValidateSelectorValue(status); err != nil {
		return nil, err
	}
	q := fmt.Sprintf(`{"selector":{"docType":"PRODUCT","status":"%s"}}`, status)
	return c.queryProducts(ctx, q)
}

func (c *ProductContract) GetProductsByOwner(ctx contractapi.TransactionContextInterface, ownerMSP string) ([]*models.Product, error) {
	if err := validation.ValidateSelectorValue(ownerMSP); err != nil {
		return nil, err
	}
	q := fmt.Sprintf(`{"selector":{"docType":"PRODUCT","currentOwnerMSP":"%s"}}`, ownerMSP)
	return c.queryProducts(ctx, q)
}

func (c *ProductContract) GetProductHistory(ctx contractapi.TransactionContextInterface, id string) ([]ledger.HistoryEntry, error) {
	return store.GetHistory(ctx, models.PrefixProduct+id)
}

func (c *ProductContract) GetProvenance(ctx contractapi.TransactionContextInterface, productID string) (*models.ProvenanceResult, error) {
	p, err := c.ReadProduct(ctx, productID)
	if err != nil {
		return nil, err
	}

	history, err := c.GetProductHistory(ctx, productID)
	if err != nil {
		return nil, err
	}

	custodyContract := &CustodyContract{}
	custody := make([]*models.CustodyRecord, 0)
	if p.CurrentShipmentID != "" {
		custody, err = custodyContract.GetCustodyChain(ctx, p.CurrentShipmentID)
		if err != nil {
			return nil, err
		}
	}

	eventContract := &EventContract{}
	events, err := eventContract.GetEvents(ctx, productID)
	if err != nil {
		return nil, err
	}

	historyResults := make([]*models.HistoryQueryResult, len(history))
	for i, h := range history {
		historyResults[i] = &models.HistoryQueryResult{
			TxID: h.TxID, Timestamp: h.Timestamp, IsDelete: h.IsDelete, Value: h.Value,
		}
	}
	if historyResults == nil {
		historyResults = make([]*models.HistoryQueryResult, 0)
	}
	if events == nil {
		events = make([]*models.SupplyChainEvent, 0)
	}

	return &models.ProvenanceResult{Product: p, History: historyResults, Custody: custody, Events: events}, nil
}

func (c *ProductContract) queryProducts(ctx contractapi.TransactionContextInterface, query string) ([]*models.Product, error) {
	var products []*models.Product
	err := store.QueryBySelector(ctx, query, func(b []byte) error {
		var p models.Product
		if err := json.Unmarshal(b, &p); err != nil {
			return err
		}
		products = append(products, &p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if products == nil {
		products = make([]*models.Product, 0)
	}
	return products, nil
}
