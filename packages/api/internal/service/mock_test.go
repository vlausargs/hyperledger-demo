package service

import (
	"errors"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
)

// mockGateway is a hand-rolled FabricGateway double used across service
// tests. The submit/evaluate funcs let each test inject behaviour without
// pulling in a mocking framework.
type mockGateway struct {
	submitFn   func(fn string, args ...string) ([]byte, error)
	evaluateFn func(fn string, args ...string) ([]byte, error)
	channel    string
	chaincode  string
	profile    *fabric.ConnectionProfile
}

func (m *mockGateway) GetContract() *client.Contract { return nil }

func (m *mockGateway) SubmitTransaction(fn string, args ...string) ([]byte, error) {
	if m.submitFn == nil {
		return nil, errors.New("submitFn not set")
	}
	return m.submitFn(fn, args...)
}

func (m *mockGateway) EvaluateTransaction(fn string, args ...string) ([]byte, error) {
	if m.evaluateFn == nil {
		return nil, errors.New("evaluateFn not set")
	}
	return m.evaluateFn(fn, args...)
}

func (m *mockGateway) GetChannel() string                            { return m.channel }
func (m *mockGateway) GetChaincode() string                          { return m.chaincode }
func (m *mockGateway) GetConnectionProfile() *fabric.ConnectionProfile { return m.profile }
