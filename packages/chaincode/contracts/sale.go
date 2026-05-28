package contracts

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"github.com/myindo/hlf-supply-chain/chaincode/ledger"
	"github.com/myindo/hlf-supply-chain/chaincode/models"
	"github.com/myindo/hlf-supply-chain/chaincode/validation"
)

type SaleContract struct {
	contractapi.Contract
}

func (c *SaleContract) CreateSale(ctx contractapi.TransactionContextInterface,
	id, customerID, cashierID, cashierName, itemsJSON, taxAmountStr, currency, notes string) error {

	if err := validation.ValidateID(id); err != nil {
		return err
	}
	for _, f := range []struct{ name, value string }{
		{"cashierID", cashierID}, {"cashierName", cashierName}, {"itemsJSON", itemsJSON},
	} {
		if err := validation.ValidateRequired(f.name, f.value); err != nil {
			return err
		}
	}
	caller, err := ledger.RequireMSP(ctx, "Org3MSP")
	if err != nil {
		return err
	}

	exists, err := store.Exists(ctx, models.PrefixSale+id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("sale %s already exists", id)
	}

	var items []models.SaleItem
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return fmt.Errorf("failed to unmarshal items: %w", err)
	}
	if len(items) == 0 {
		return fmt.Errorf("sale must have at least one item")
	}

	taxAmount, err := strconv.ParseFloat(taxAmountStr, 64)
	if err != nil {
		return fmt.Errorf("invalid taxAmount %q: %w", taxAmountStr, err)
	}

	now, err := ledger.TxTimestamp(ctx)
	if err != nil {
		return err
	}

	var subTotal float64
	for i, item := range items {
		var p models.Product
		if err := store.GetByID(ctx, models.PrefixProduct+item.ProductID, &p); err != nil {
			return err
		}
		if p.Status != models.ProductStatusDelivered {
			return fmt.Errorf("product %s is not DELIVERED (status: %s)", item.ProductID, p.Status)
		}
		if p.CurrentOwnerMSP != "Org3MSP" {
			return fmt.Errorf("product %s is not owned by Org3MSP", item.ProductID)
		}
		if p.RecallID != "" {
			return fmt.Errorf("product %s is under recall %s", item.ProductID, p.RecallID)
		}
		if err := validation.ValidateProductStatusTransition(p.Status, models.ProductStatusSold); err != nil {
			return err
		}

		p.Status = models.ProductStatusSold
		p.UpdatedAt = now
		if err := store.PutJSON(ctx, models.PrefixProduct+item.ProductID, p); err != nil {
			return err
		}

		items[i].SKU = p.SKU
		items[i].ProductName = p.Name
		subTotal += item.UnitPrice
	}

	if currency == "" {
		currency = "IDR"
	}

	sale := models.SaleTransaction{
		DocType:     "SALE",
		ID:          id,
		Items:       items,
		CustomerID:  customerID,
		CashierID:   cashierID,
		CashierName: cashierName,
		SubTotal:    subTotal,
		TaxAmount:   taxAmount,
		TotalAmount: subTotal + taxAmount,
		Currency:    currency,
		RetailerMSP: caller,
		Notes:       notes,
		TxID:        ctx.GetStub().GetTxID(),
		CreatedAt:   now,
	}
	return store.PutJSON(ctx, models.PrefixSale+id, sale)
}

func (c *SaleContract) ReadSale(ctx contractapi.TransactionContextInterface, id string) (*models.SaleTransaction, error) {
	var sale models.SaleTransaction
	if err := store.GetByID(ctx, models.PrefixSale+id, &sale); err != nil {
		return nil, err
	}
	return &sale, nil
}

func (c *SaleContract) GetAllSales(ctx contractapi.TransactionContextInterface, pageSizeStr, bookmark string) (*models.PagedSaleResult, error) {
	pageSize, err := strconv.ParseInt(pageSizeStr, 10, 32)
	if err != nil || pageSize <= 0 {
		pageSize = 20
	}

	var sales []*models.SaleTransaction
	bm, err := store.GetPaginated(ctx,
		models.PrefixSale, models.PrefixSale+"\x7f",
		int32(pageSize), bookmark,
		func(b []byte) error {
			var sale models.SaleTransaction
			if err := json.Unmarshal(b, &sale); err != nil {
				return err
			}
			sales = append(sales, &sale)
			return nil
		})
	if err != nil {
		return nil, err
	}
	if sales == nil {
		sales = make([]*models.SaleTransaction, 0)
	}
	return &models.PagedSaleResult{Sales: sales, Bookmark: bm, Count: len(sales)}, nil
}

func (c *SaleContract) GetSalesByCustomer(ctx contractapi.TransactionContextInterface, customerID string) ([]*models.SaleTransaction, error) {
	if err := validation.ValidateSelectorValue(customerID); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`{"selector":{"docType":"SALE","customerId":"%s"},"use_index":["indexSaleByCustomerDoc","indexSaleByCustomer"]}`, customerID)
	var sales []*models.SaleTransaction
	err := store.QueryBySelector(ctx, query, func(b []byte) error {
		var sale models.SaleTransaction
		if err := json.Unmarshal(b, &sale); err != nil {
			return err
		}
		sales = append(sales, &sale)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if sales == nil {
		sales = make([]*models.SaleTransaction, 0)
	}
	return sales, nil
}

