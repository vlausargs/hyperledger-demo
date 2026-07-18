package service

import (
	"context"
	"encoding/json"

	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// InitiateCustodyInput is the typed payload to start a custody handover.
type InitiateCustodyInput struct {
	ShipmentID string
	ToMSP      string
	ToName     string
	Conditions string
}

// CustodyService manages multi-party custody handoffs on a shipment.
type CustodyService struct {
	gw FabricGateway
}

func NewCustodyService(gw FabricGateway) *CustodyService {
	return &CustodyService{gw: gw}
}

// Chain returns the ordered custody history for a shipment.
func (s *CustodyService) Chain(_ context.Context, shipmentID string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "custody chain", "CustodyContract:GetCustodyChain", shipmentID)
}

// Initiate opens a pending custody transfer.
func (s *CustodyService) Initiate(_ context.Context, in InitiateCustodyInput) *apperrors.AppError {
	_, err := s.gw.SubmitTransaction("CustodyContract:InitiateCustodyTransfer", in.ShipmentID, in.ToMSP, in.ToName, in.Conditions)
	if err != nil {
		return mapFabricError(err, "custody transfer")
	}
	return nil
}

// Accept finalises a pending custody transfer.
func (s *CustodyService) Accept(_ context.Context, shipmentID string) *apperrors.AppError {
	if _, err := s.gw.SubmitTransaction("CustodyContract:AcceptCustodyTransfer", shipmentID); err != nil {
		return mapFabricError(err, "custody transfer")
	}
	return nil
}

// Reject cancels a pending custody transfer.
func (s *CustodyService) Reject(_ context.Context, shipmentID string) *apperrors.AppError {
	if _, err := s.gw.SubmitTransaction("CustodyContract:RejectCustodyTransfer", shipmentID); err != nil {
		return mapFabricError(err, "custody transfer")
	}
	return nil
}
