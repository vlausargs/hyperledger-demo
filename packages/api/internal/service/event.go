package service

import (
	"context"
	"encoding/json"

	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// LogEventInput is the typed payload for EventService.Log.
type LogEventInput struct {
	ID          string
	TargetID    string
	TargetType  string
	EventType   string
	Description string
	Location    string
	OccurredAt  string
	Data        map[string]string
}

// EventService records auditable supply-chain events.
type EventService struct {
	gw FabricGateway
}

func NewEventService(gw FabricGateway) *EventService {
	return &EventService{gw: gw}
}

// Log appends an event to the ledger.
func (s *EventService) Log(_ context.Context, in LogEventInput) (*CreateResult, *apperrors.AppError) {
	dataJSON, err := marshalMap(in.Data, "{}")
	if err != nil {
		return nil, apperrors.NewBadRequest("invalid data: " + err.Error())
	}
	if _, err := s.gw.SubmitTransaction("EventContract:LogEvent",
		in.ID, in.TargetID, in.TargetType, in.EventType,
		in.Description, in.Location, in.OccurredAt, dataJSON); err != nil {
		return nil, mapFabricError(err, "event")
	}
	return &CreateResult{ID: in.ID}, nil
}

// List returns events for a target (product or shipment).
func (s *EventService) List(_ context.Context, targetID string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "events", "EventContract:GetEvents", targetID)
}
