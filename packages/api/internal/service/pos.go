package service

import (
	"context"
	"encoding/json"

	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// POSService backs the retailer point-of-sale UI: inventory snapshot and
// pre-sale provenance verification.
type POSService struct {
	gw FabricGateway
}

func NewPOSService(gw FabricGateway) *POSService {
	return &POSService{gw: gw}
}

// Inventory lists products currently owned by an MSP (default Org3MSP).
func (s *POSService) Inventory(_ context.Context, ownerMSP string) (json.RawMessage, *apperrors.AppError) {
	if ownerMSP == "" {
		ownerMSP = "Org3MSP"
	}
	return evalRaw(s.gw, "inventory", "GetInventory", ownerMSP)
}

// VerifyProduct returns the upstream provenance — used at checkout to flag
// recalled or untraceable items before sale.
func (s *POSService) VerifyProduct(_ context.Context, productID string) (json.RawMessage, *apperrors.AppError) {
	return evalRaw(s.gw, "product", "GetProvenance", productID)
}
