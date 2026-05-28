package service

import (
	"context"

	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
	apperrors "github.com/myindo/hlf-supply-chain/api/pkg/errors"
)

// NetworkService exposes static topology info derived from the connection
// profile. None of these calls hit Fabric over the wire.
type NetworkService struct {
	gw FabricGateway
}

func NewNetworkService(gw FabricGateway) *NetworkService {
	return &NetworkService{gw: gw}
}

// ChannelID returns the channel this connection is bound to.
func (s *NetworkService) ChannelID() string { return s.gw.GetChannel() }

// ChaincodeID returns the chaincode this connection talks to.
func (s *NetworkService) ChaincodeID() string { return s.gw.GetChaincode() }

// Profile returns the parsed connection profile or an ErrInternal if the
// gateway was constructed without one (should not happen in practice).
func (s *NetworkService) Profile(_ context.Context) (*fabric.ConnectionProfile, *apperrors.AppError) {
	cp := s.gw.GetConnectionProfile()
	if cp == nil {
		return nil, apperrors.NewInternal("connection profile not loaded", "")
	}
	return cp, nil
}
