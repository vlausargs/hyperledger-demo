package service

import (
	"context"

	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// HealthService verifies the API can still reach the ledger.
type HealthService struct {
	gw FabricGateway
}

func NewHealthService(gw FabricGateway) *HealthService {
	return &HealthService{gw: gw}
}

// Check runs a cheap evaluate against the ledger. Returns ErrUnavailable
// when the underlying gateway fails.
func (s *HealthService) Check(_ context.Context) *apperrors.AppError {
	if _, err := s.gw.EvaluateTransaction("ProductContract:GetAllProducts", "1", ""); err != nil {
		return apperrors.NewUnavailable("ledger unreachable")
	}
	return nil
}
