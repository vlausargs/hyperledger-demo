package contracts

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"github.com/myindo/hlf-supply-chain/chaincode/ledger"
	"github.com/myindo/hlf-supply-chain/chaincode/models"
	"github.com/myindo/hlf-supply-chain/chaincode/validation"
)

type ShipmentContract struct {
	contractapi.Contract
}

func (c *ShipmentContract) CreateShipment(ctx contractapi.TransactionContextInterface,
	id, name, description, receiverMSP, receiverName, origin, destination, productIDsJSON string) error {

	if err := validation.ValidateID(id); err != nil {
		return err
	}
	caller, err := ledger.CallerMSPID(ctx)
	if err != nil {
		return err
	}
	now, err := ledger.TxTimestamp(ctx)
	if err != nil {
		return err
	}

	exists, err := c.ShipmentExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("shipment %s already exists", id)
	}

	var productIDs []string
	if err := json.Unmarshal([]byte(productIDsJSON), &productIDs); err != nil {
		return fmt.Errorf("invalid productIds JSON: %w", err)
	}
	if len(productIDs) == 0 {
		return fmt.Errorf("shipment must contain at least one product")
	}

	// Validate all products exist and are owned by caller
	for _, pid := range productIDs {
		var p models.Product
		if err := store.GetByID(ctx, models.PrefixProduct+pid, &p); err != nil {
			return err
		}
		if p.CurrentOwnerMSP != caller {
			return fmt.Errorf("caller MSP %s does not own product %s (owner: %s)", caller, pid, p.CurrentOwnerMSP)
		}
		if p.Status != models.ProductStatusActive {
			return fmt.Errorf("product %s is not in ACTIVE status (current: %s)", pid, p.Status)
		}
	}

	// Update product references
	for _, pid := range productIDs {
		var p models.Product
		_ = store.GetByID(ctx, models.PrefixProduct+pid, &p)
		p.CurrentShipmentID = id
		p.Status = models.ProductStatusShipped
		p.UpdatedAt = now
		if err := store.PutJSON(ctx, models.PrefixProduct+pid, p); err != nil {
			return err
		}
	}

	ship := models.Shipment{
		DocType: "SHIPMENT", ID: id, Name: name, Description: description,
		ProductIDs: productIDs, SenderMSP: caller, SenderName: caller,
		ReceiverMSP: receiverMSP, ReceiverName: receiverName,
		Status: models.ShipmentStatusDraft, Origin: origin, Destination: destination,
		CreatedAt: now, UpdatedAt: now,
	}
	return store.PutJSON(ctx, models.PrefixShipment+id, ship)
}

func (c *ShipmentContract) ReadShipment(ctx contractapi.TransactionContextInterface, id string) (*models.Shipment, error) {
	var s models.Shipment
	if err := store.GetByID(ctx, models.PrefixShipment+id, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *ShipmentContract) DispatchShipment(ctx contractapi.TransactionContextInterface, id string) error {
	ship, err := c.ReadShipment(ctx, id)
	if err != nil {
		return err
	}
	caller, err := ledger.CallerMSPID(ctx)
	if err != nil {
		return err
	}
	if caller != ship.SenderMSP {
		return fmt.Errorf("only sender MSP %s may dispatch shipment %s", ship.SenderMSP, id)
	}
	if ship.Status != models.ShipmentStatusDraft {
		return fmt.Errorf("shipment %s is not in DRAFT status (current: %s)", id, ship.Status)
	}
	now, err := ledger.TxTimestamp(ctx)
	if err != nil {
		return err
	}
	ship.Status = models.ShipmentStatusInTransit
	ship.DepartureAt = now
	ship.UpdatedAt = now
	return store.PutJSON(ctx, models.PrefixShipment+id, ship)
}

func (c *ShipmentContract) ShipmentExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	return store.Exists(ctx, models.PrefixShipment+id)
}

func (c *ShipmentContract) GetAllShipments(ctx contractapi.TransactionContextInterface, pageSize int, bookmark string) (*models.PagedShipmentResult, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var shipments []*models.Shipment
	bm, err := store.GetPaginated(ctx,
		models.PrefixShipment, models.PrefixShipment+"\x7f",
		int32(pageSize), bookmark,
		func(b []byte) error {
			var sh models.Shipment
			if err := json.Unmarshal(b, &sh); err != nil {
				return err
			}
			shipments = append(shipments, &sh)
			return nil
		})
	if err != nil {
		return nil, err
	}
	if shipments == nil {
		shipments = make([]*models.Shipment, 0)
	}
	return &models.PagedShipmentResult{Shipments: shipments, Bookmark: bm, Count: len(shipments)}, nil
}

func (c *ShipmentContract) GetShipmentsByStatus(ctx contractapi.TransactionContextInterface, status string) ([]*models.Shipment, error) {
	if err := validation.ValidateSelectorValue(status); err != nil {
		return nil, err
	}
	q := fmt.Sprintf(`{"selector":{"docType":"SHIPMENT","status":"%s"}}`, status)
	var shipments []*models.Shipment
	err := store.QueryBySelector(ctx, q, func(b []byte) error {
		var sh models.Shipment
		if err := json.Unmarshal(b, &sh); err != nil {
			return err
		}
		shipments = append(shipments, &sh)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if shipments == nil {
		shipments = make([]*models.Shipment, 0)
	}
	return shipments, nil
}

func (c *ShipmentContract) GetShipmentHistory(ctx contractapi.TransactionContextInterface, id string) ([]ledger.HistoryEntry, error) {
	return store.GetHistory(ctx, models.PrefixShipment+id)
}
