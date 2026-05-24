package contracts

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"github.com/myindo/hlf-supply-chain/chaincode/ledger"
	"github.com/myindo/hlf-supply-chain/chaincode/models"
	"github.com/myindo/hlf-supply-chain/chaincode/validation"
)

type EventContract struct {
	contractapi.Contract
}

func (c *EventContract) LogEvent(ctx contractapi.TransactionContextInterface,
	id, targetID, targetType, eventType, description, location, occurredAt, dataJSON string) error {

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
	txID := ctx.GetStub().GetTxID()

	var data map[string]string
	if dataJSON != "" {
		if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
			return fmt.Errorf("invalid data JSON: %w", err)
		}
	}

	if occurredAt == "" {
		occurredAt = now
	}

	ev := models.SupplyChainEvent{
		DocType: "EVENT", ID: id, TargetID: targetID, TargetType: targetType,
		EventType: eventType, Description: description, Location: location,
		RecordedBy: caller, Data: data, OccurredAt: occurredAt,
		TxID: txID, CreatedAt: now,
	}
	// Key: EVENT~{targetID}~{txTimestamp}~{eventID}
	key := fmt.Sprintf("%s%s~%s~%s", models.PrefixEvent, targetID, now, id)
	return store.PutJSON(ctx, key, ev)
}

func (c *EventContract) GetEvents(ctx contractapi.TransactionContextInterface, targetID string) ([]*models.SupplyChainEvent, error) {
	start := models.PrefixEvent + targetID + "~"
	end := models.PrefixEvent + targetID + "~\x7f"

	var events []*models.SupplyChainEvent
	err := store.GetByRange(ctx, start, end, func(b []byte) error {
		var ev models.SupplyChainEvent
		if err := json.Unmarshal(b, &ev); err != nil {
			return err
		}
		events = append(events, &ev)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = make([]*models.SupplyChainEvent, 0)
	}
	return events, nil
}
