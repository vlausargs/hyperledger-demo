package service

import (
	"context"
	"encoding/json"

	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// CreateShipmentInput is the typed payload for ShipmentService.Create.
type CreateShipmentInput struct {
	ID           string
	Name         string
	Description  string
	ReceiverMSP  string
	ReceiverName string
	Origin       string
	Destination  string
	ProductIDs   []string
}

// ShipmentService handles shipment lifecycle operations.
type ShipmentService struct {
	gw FabricGateway
}

func NewShipmentService(gw FabricGateway) *ShipmentService {
	return &ShipmentService{gw: gw}
}

// Create registers a new shipment. Returns ErrConflict on duplicate ID.
func (s *ShipmentService) Create(_ context.Context, in CreateShipmentInput) (*CreateResult, *apperrors.AppError) {
	pidJSON, err := json.Marshal(in.ProductIDs)
	if err != nil {
		return nil, apperrors.NewBadRequest("invalid productIds")
	}
	if _, err := s.gw.SubmitTransaction("ShipmentContract:CreateShipment",
		in.ID, in.Name, in.Description,
		in.ReceiverMSP, in.ReceiverName,
		in.Origin, in.Destination, string(pidJSON)); err != nil {
		return nil, mapFabricError(err, "shipment")
	}
	return &CreateResult{ID: in.ID}, nil
}

// Dispatch moves a shipment into the IN_TRANSIT state.
func (s *ShipmentService) Dispatch(_ context.Context, id string) (*CreateResult, *apperrors.AppError) {
	if _, err := s.gw.SubmitTransaction("ShipmentContract:DispatchShipment", id); err != nil {
		return nil, mapFabricError(err, "shipment")
	}
	return &CreateResult{ID: id}, nil
}

// Get returns a single shipment.
func (s *ShipmentService) Get(_ context.Context, id string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "shipment", "ShipmentContract:ReadShipment", id)
}

// List returns all shipments with paging.
func (s *ShipmentService) List(_ context.Context, pageSize, bookmark string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "shipments", "ShipmentContract:GetAllShipments", pageSize, bookmark)
}

// History returns the immutable change history for a shipment.
func (s *ShipmentService) History(_ context.Context, id string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "shipment history", "ShipmentContract:GetShipmentHistory", id)
}

// ByStatus lists shipments in a given status.
func (s *ShipmentService) ByStatus(_ context.Context, status string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "shipments", "ShipmentContract:GetShipmentsByStatus", status)
}
