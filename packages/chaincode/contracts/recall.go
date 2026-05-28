package contracts

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"github.com/myindo/hlf-supply-chain/chaincode/ledger"
	"github.com/myindo/hlf-supply-chain/chaincode/models"
	"github.com/myindo/hlf-supply-chain/chaincode/validation"
)

type RecallContract struct {
	contractapi.Contract
}

func (c *RecallContract) IssueRecall(ctx contractapi.TransactionContextInterface,
	id, scope, targetIDsJSON, reason, severity, issuedByName, instructionsURL string) error {

	if err := validation.ValidateID(id); err != nil {
		return err
	}
	for _, f := range []struct{ name, value string }{
		{"scope", scope}, {"reason", reason}, {"severity", severity}, {"issuedByName", issuedByName},
	} {
		if err := validation.ValidateRequired(f.name, f.value); err != nil {
			return err
		}
	}
	caller, err := ledger.CallerMSPID(ctx)
	if err != nil {
		return err
	}
	now, err := ledger.TxTimestamp(ctx)
	if err != nil {
		return err
	}

	var targetIDs []string
	if err := json.Unmarshal([]byte(targetIDsJSON), &targetIDs); err != nil {
		return fmt.Errorf("invalid targetIds JSON: %w", err)
	}
	if len(targetIDs) == 0 {
		return fmt.Errorf("recall must target at least one item")
	}

	affectedCount := 0

	switch scope {
	case models.RecallScopeProduct:
		for _, pid := range targetIDs {
			var p models.Product
			if err := store.GetByID(ctx, models.PrefixProduct+pid, &p); err != nil {
				return err
			}
			if err := validation.ValidateProductStatusTransition(p.Status, models.ProductStatusRecalled); err != nil {
				return err
			}
			p.Status = models.ProductStatusRecalled
			p.RecallID = id
			p.UpdatedAt = now
			if err := store.PutJSON(ctx, models.PrefixProduct+pid, p); err != nil {
				return err
			}
			affectedCount++
		}
	case models.RecallScopeShipment:
		for _, sid := range targetIDs {
			var ship models.Shipment
			if err := store.GetByID(ctx, models.PrefixShipment+sid, &ship); err != nil {
				return err
			}
			if err := validation.ValidateShipmentStatusTransition(ship.Status, models.ShipmentStatusRecalled); err != nil {
				return err
			}
			ship.Status = models.ShipmentStatusRecalled
			ship.RecallID = id
			ship.UpdatedAt = now
			if err := store.PutJSON(ctx, models.PrefixShipment+sid, ship); err != nil {
				return err
			}
			// Also recall all products in shipment
			for _, pid := range ship.ProductIDs {
				var p models.Product
				if err := store.GetByID(ctx, models.PrefixProduct+pid, &p); err != nil {
					return err
				}
				if err := validation.ValidateProductStatusTransition(p.Status, models.ProductStatusRecalled); err != nil {
					return err
				}
				p.Status = models.ProductStatusRecalled
				p.RecallID = id
				p.UpdatedAt = now
				if err := store.PutJSON(ctx, models.PrefixProduct+pid, p); err != nil {
					return err
				}
				affectedCount++
			}
		}
	case models.RecallScopeBatch:
		// targetIDs are batch IDs -- find all matching products
		pc := &ProductContract{}
		for _, batchID := range targetIDs {
			products, err := pc.GetProductsByBatch(ctx, batchID)
			if err != nil {
				return err
			}
			for _, p := range products {
				if err := validation.ValidateProductStatusTransition(p.Status, models.ProductStatusRecalled); err != nil {
					return err
				}
				p.Status = models.ProductStatusRecalled
				p.RecallID = id
				p.UpdatedAt = now
				if err := store.PutJSON(ctx, models.PrefixProduct+p.ID, p); err != nil {
					return err
				}
				affectedCount++
			}
		}
	default:
		return fmt.Errorf("invalid recall scope %s; must be PRODUCT, SHIPMENT, or BATCH", scope)
	}

	recall := models.RecallNotice{
		DocType: "RECALL", ID: id, Scope: scope, TargetIDs: targetIDs,
		Reason: reason, IssuedBy: caller, IssuedByName: issuedByName,
		Severity: severity, InstructionsURL: instructionsURL,
		AffectedCount: affectedCount, CreatedAt: now,
	}
	return store.PutJSON(ctx, models.PrefixRecall+id, recall)
}

func (c *RecallContract) ReadRecall(ctx contractapi.TransactionContextInterface, id string) (*models.RecallNotice, error) {
	var r models.RecallNotice
	if err := store.GetByID(ctx, models.PrefixRecall+id, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *RecallContract) GetRecalledProducts(ctx contractapi.TransactionContextInterface) ([]*models.Product, error) {
	pc := &ProductContract{}
	return pc.GetProductsByStatus(ctx, models.ProductStatusRecalled)
}
