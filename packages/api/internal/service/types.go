// Package service holds the business-logic layer that sits between HTTP
// handlers and the Fabric gateway. Handlers must not call Fabric directly —
// they delegate to a service method which marshals arguments, invokes the
// gateway, maps low-level errors into pkg/errors AppErrors, and returns
// typed Go values.
package service

import (
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
)

// FabricGateway is the narrow interface services need from the Fabric
// gateway. Defined here (not in the handler package) so handlers can depend
// on services without pulling in Fabric types directly. The real
// *fabric.Gateway satisfies this; tests use mocks.
type FabricGateway interface {
	GetContract() *client.Contract
	SubmitTransaction(function string, args ...string) ([]byte, error)
	EvaluateTransaction(function string, args ...string) ([]byte, error)
	GetChannel() string
	GetChaincode() string
	GetConnectionProfile() *fabric.ConnectionProfile
}

// Services bundles every domain service. The router/handlers receive a
// single *Services and pick the field they need.
type Services struct {
	Product  *ProductService
	Shipment *ShipmentService
	Custody  *CustodyService
	Event    *EventService
	Recall   *RecallService
	Sale     *SaleService
	POS      *POSService
	Network  *NetworkService
	Identity *IdentityService
	Auth     *AuthService
	Health   *HealthService
}

// New builds the full Services bundle. caClient may be nil — identity
// endpoints are then disabled at the router level.
func New(gw FabricGateway, caClient *fabric.CAClient, walletPath, mspID, jwtSecret string) *Services {
	return &Services{
		Product:  NewProductService(gw),
		Shipment: NewShipmentService(gw),
		Custody:  NewCustodyService(gw),
		Event:    NewEventService(gw),
		Recall:   NewRecallService(gw),
		Sale:     NewSaleService(gw),
		POS:      NewPOSService(gw),
		Network:  NewNetworkService(gw),
		Identity: NewIdentityService(caClient, walletPath, mspID),
		Auth:     NewAuthService(jwtSecret, mspID),
		Health:   NewHealthService(gw),
	}
}
