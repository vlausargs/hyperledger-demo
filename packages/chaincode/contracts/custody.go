package contracts

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"github.com/myindo/hlf-supply-chain/chaincode/ledger"
	"github.com/myindo/hlf-supply-chain/chaincode/models"
)

type CustodyContract struct {
	contractapi.Contract
}

// findPendingCustody scans CUSTODY~{shipmentID}~ range and returns (count, pendingRecord, error).
func (c *CustodyContract) findPendingCustody(ctx contractapi.TransactionContextInterface, shipmentID string) (int, *models.CustodyRecord, error) {
	start := models.PrefixCustody + shipmentID + "~"
	end := models.PrefixCustody + shipmentID + "~\x7f"

	count := 0
	var pending *models.CustodyRecord
	err := store.GetByRange(ctx, start, end, func(b []byte) error {
		count++
		var cr models.CustodyRecord
		if err := json.Unmarshal(b, &cr); err != nil {
			return err
		}
		if cr.Status == models.CustodyStatusPending {
			cp := cr
			pending = &cp
		}
		return nil
	})
	if err != nil {
		return 0, nil, err
	}
	return count, pending, nil
}

func (c *CustodyContract) InitiateCustodyTransfer(ctx contractapi.TransactionContextInterface,
	shipmentID, toMSP, toName, conditions string) error {

	var ship models.Shipment
	if err := store.GetByID(ctx, models.PrefixShipment+shipmentID, &ship); err != nil {
		return err
	}
	caller, err := ledger.CallerMSPID(ctx)
	if err != nil {
		return err
	}
	if caller != ship.SenderMSP {
		return fmt.Errorf("only sender MSP %s may initiate custody transfer for shipment %s", ship.SenderMSP, shipmentID)
	}
	if ship.Status != models.ShipmentStatusInTransit {
		return fmt.Errorf("shipment %s must be IN_TRANSIT to initiate custody transfer (current: %s)", shipmentID, ship.Status)
	}

	count, pending, err := c.findPendingCustody(ctx, shipmentID)
	if err != nil {
		return err
	}
	if pending != nil {
		return fmt.Errorf("shipment %s already has a pending custody transfer", shipmentID)
	}

	now, err := ledger.TxTimestamp(ctx)
	if err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()

	seq := count + 1
	custodyID := fmt.Sprintf("%s~%04d", shipmentID, seq)
	cr := models.CustodyRecord{
		DocType: "CUSTODY", ID: custodyID, ShipmentID: shipmentID,
		Sequence: seq, FromMSP: caller, FromName: caller,
		ToMSP: toMSP, ToName: toName, Status: models.CustodyStatusPending,
		Conditions: conditions, SenderSignature: txID,
		TxID: txID, CreatedAt: now, UpdatedAt: now,
	}
	return store.PutJSON(ctx, models.PrefixCustody+custodyID, cr)
}

func (c *CustodyContract) AcceptCustodyTransfer(ctx contractapi.TransactionContextInterface, shipmentID string) error {
	var ship models.Shipment
	if err := store.GetByID(ctx, models.PrefixShipment+shipmentID, &ship); err != nil {
		return err
	}
	caller, err := ledger.CallerMSPID(ctx)
	if err != nil {
		return err
	}

	_, pending, err := c.findPendingCustody(ctx, shipmentID)
	if err != nil {
		return err
	}
	if pending == nil {
		return fmt.Errorf("no pending custody transfer for shipment %s", shipmentID)
	}
	if caller != pending.ToMSP {
		return fmt.Errorf("caller MSP %s is not the intended receiver %s for custody transfer", caller, pending.ToMSP)
	}

	now, err := ledger.TxTimestamp(ctx)
	if err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()

	pending.Status = models.CustodyStatusCompleted
	pending.ReceiverSignature = txID
	pending.TransferredAt = now
	pending.UpdatedAt = now
	if err := store.PutJSON(ctx, models.PrefixCustody+pending.ID, pending); err != nil {
		return err
	}

	// Update shipment: new sender = receiver of this custody
	ship.SenderMSP = pending.ToMSP
	ship.SenderName = pending.ToName
	ship.UpdatedAt = now

	// Check if receiver is the final destination
	if pending.ToMSP == ship.ReceiverMSP {
		ship.Status = models.ShipmentStatusDelivered
		ship.ArrivalAt = now
		// Update all products: delivered
		for _, pid := range ship.ProductIDs {
			var p models.Product
			if err := store.GetByID(ctx, models.PrefixProduct+pid, &p); err != nil {
				return err
			}
			p.CurrentOwnerMSP = pending.ToMSP
			p.CurrentOwner = pending.ToName
			p.Status = models.ProductStatusDelivered
			p.CurrentShipmentID = ""
			p.UpdatedAt = now
			if err := store.PutJSON(ctx, models.PrefixProduct+pid, p); err != nil {
				return err
			}
		}
	} else {
		// Intermediate handoff -- update products' owner but keep them SHIPPED
		for _, pid := range ship.ProductIDs {
			var p models.Product
			if err := store.GetByID(ctx, models.PrefixProduct+pid, &p); err != nil {
				return err
			}
			p.CurrentOwnerMSP = pending.ToMSP
			p.CurrentOwner = pending.ToName
			p.UpdatedAt = now
			if err := store.PutJSON(ctx, models.PrefixProduct+pid, p); err != nil {
				return err
			}
		}
	}

	return store.PutJSON(ctx, models.PrefixShipment+shipmentID, ship)
}

func (c *CustodyContract) RejectCustodyTransfer(ctx contractapi.TransactionContextInterface, shipmentID string) error {
	if err := store.GetByID(ctx, models.PrefixShipment+shipmentID, &models.Shipment{}); err != nil {
		return err
	}
	caller, err := ledger.CallerMSPID(ctx)
	if err != nil {
		return err
	}

	_, pending, err := c.findPendingCustody(ctx, shipmentID)
	if err != nil {
		return err
	}
	if pending == nil {
		return fmt.Errorf("no pending custody transfer for shipment %s", shipmentID)
	}
	if caller != pending.ToMSP {
		return fmt.Errorf("caller MSP %s is not the intended receiver %s for custody transfer", caller, pending.ToMSP)
	}

	now, err := ledger.TxTimestamp(ctx)
	if err != nil {
		return err
	}

	pending.Status = models.CustodyStatusRejected
	pending.UpdatedAt = now
	return store.PutJSON(ctx, models.PrefixCustody+pending.ID, pending)
}

func (c *CustodyContract) GetCustodyChain(ctx contractapi.TransactionContextInterface, shipmentID string) ([]*models.CustodyRecord, error) {
	start := models.PrefixCustody + shipmentID + "~"
	end := models.PrefixCustody + shipmentID + "~\x7f"

	var records []*models.CustodyRecord
	err := store.GetByRange(ctx, start, end, func(b []byte) error {
		var cr models.CustodyRecord
		if err := json.Unmarshal(b, &cr); err != nil {
			return err
		}
		records = append(records, &cr)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if records == nil {
		records = make([]*models.CustodyRecord, 0)
	}
	return records, nil
}
