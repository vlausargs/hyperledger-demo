package service

import (
	"encoding/json"

	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
	"github.com/hyperledger/fabric-gateway/pkg/client"
)

type FabricService struct {
	gw *fabric.Gateway
}

func NewFabricService(gw *fabric.Gateway) *FabricService {
	return &FabricService{gw: gw}
}

func (s *FabricService) GetContract() *client.Contract {
	return s.gw.GetContract()
}

func (s *FabricService) Submit(fn string, args ...string) ([]byte, error) {
	return s.gw.SubmitTransaction(fn, args...)
}

func (s *FabricService) Evaluate(fn string, args ...string) ([]byte, error) {
	return s.gw.EvaluateTransaction(fn, args...)
}

func (s *FabricService) EvaluateJSON(fn string, args ...string) (json.RawMessage, error) {
	b, err := s.gw.EvaluateTransaction(fn, args...)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return json.RawMessage("null"), nil
	}
	return b, nil
}

func (s *FabricService) GetChannel() string {
	return s.gw.GetChannel()
}

func (s *FabricService) GetChaincode() string {
	return s.gw.GetChaincode()
}

func (s *FabricService) GetConnectionProfile() *fabric.ConnectionProfile {
	return s.gw.GetConnectionProfile()
}
