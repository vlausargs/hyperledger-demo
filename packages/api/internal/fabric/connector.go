package fabric

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	gwproto "github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

// Gateway represents the Fabric Gateway connection
type Gateway struct {
	gateway           *client.Gateway
	network           *client.Network
	contract          *client.Contract
	channel           string
	chaincode         string
	walletPath        string
	tlsCertPath       string
	mspPath           string
	connectionProfile *ConnectionProfile
	logger            *slog.Logger
}

// ConnectionProfile represents the parsed connection profile YAML
type ConnectionProfile struct {
	Name          string                  `yaml:"name"`
	Version       string                  `yaml:"version"`
	Client        Client                  `yaml:"client"`
	Organizations map[string]Organization `yaml:"organizations"`
	Peers         map[string]Peer         `yaml:"peers"`
	CAs           map[string]CA           `yaml:"certificateAuthorities"`
}

// Client configuration
type Client struct {
	Organization string           `yaml:"organization"`
	Connection   ClientConnection `yaml:"connection"`
}

type ClientConnection struct {
	Timeout map[string]map[string]string `yaml:"timeout"`
}

// Organization configuration
type Organization struct {
	MSPID                  string   `yaml:"mspid"`
	Peers                  []string `yaml:"peers"`
	CertificateAuthorities []string `yaml:"certificateAuthorities"`
}

// Peer configuration
type Peer struct {
	URL         string            `yaml:"url"`
	TLSCACerts  TLSCerts          `yaml:"tlsCACerts"`
	GRPCOptions map[string]string `yaml:"grpcOptions"`
}

// CA configuration
type CA struct {
	URL         string      `yaml:"url"`
	CAName      string      `yaml:"caName"`
	TLSCACerts  TLSCerts    `yaml:"tlsCACerts"`
	HTTPOptions HTTPOptions `yaml:"httpOptions"`
}

// TLSCerts configuration
type TLSCerts struct {
	Pem string `yaml:"pem"`
}

// HTTPOptions configuration
type HTTPOptions struct {
	Verify bool `yaml:"verify"`
}

// NewGateway creates a new Fabric Gateway connection using a connection profile
func NewGateway(channelID, chaincodeID, walletPath, tlsCertPath, connectionProfilePath string) (*Gateway, error) {
	logger := slog.Default().With("component", "fabric-gateway")

	logger.Info("initializing fabric gateway", "connection_profile", connectionProfilePath)

	// Load and parse the connection profile
	connectionProfile, err := loadConnectionProfile(connectionProfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load connection profile: %w", err)
	}
	logger.Info("connection profile loaded", "name", connectionProfile.Name)

	gw := &Gateway{
		channel:           channelID,
		chaincode:         chaincodeID,
		walletPath:        walletPath,
		tlsCertPath:       tlsCertPath,
		mspPath:           filepath.Join(filepath.Dir(tlsCertPath), "msp"),
		connectionProfile: connectionProfile,
		logger:            logger,
	}

	// Create wallet directory if it doesn't exist
	if err := os.MkdirAll(walletPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create wallet directory: %w", err)
	}

	// Initialize gateway connection
	if err := gw.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to gateway: %w", err)
	}

	gw.logger.Info("connected to fabric gateway", "channel", channelID, "chaincode", chaincodeID)

	return gw, nil
}

// loadConnectionProfile loads and parses the connection profile YAML file
func loadConnectionProfile(path string) (*ConnectionProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read connection profile file: %w", err)
	}

	var profile ConnectionProfile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse connection profile YAML: %w", err)
	}

	return &profile, nil
}

// connect establishes connection to the Fabric network using the connection profile
func (gw *Gateway) connect() error {
	// Get client organization
	orgName := gw.connectionProfile.Client.Organization
	org, exists := gw.connectionProfile.Organizations[orgName]
	if !exists {
		return fmt.Errorf("organization %s not found in connection profile", orgName)
	}

	gw.logger.Info("using organization", "name", orgName, "mspid", org.MSPID)

	// Get the first peer from the organization
	if len(org.Peers) == 0 {
		return fmt.Errorf("no peers defined for organization %s", orgName)
	}

	peerName := org.Peers[0]
	peer, exists := gw.connectionProfile.Peers[peerName]
	if !exists {
		return fmt.Errorf("peer %s not found in connection profile", peerName)
	}

	gw.logger.Info("using peer", "peer", peerName)

	// Parse peer URL
	peerURL, err := url.Parse(peer.URL)
	if err != nil {
		return fmt.Errorf("failed to parse peer URL %s: %w", peer.URL, err)
	}

	peerHost := peerURL.Hostname()
	peerPort := peerURL.Port()
	if peerPort == "" {
		if peerURL.Scheme == "grpcs" {
			peerPort = "7051"
		} else {
			peerPort = "7051"
		}
	}

	peerEndpoint := fmt.Sprintf("%s:%s", peerHost, peerPort)
	gw.logger.Info("peer endpoint", "endpoint", peerEndpoint)

	// Get TLS certificate from connection profile
	if peer.TLSCACerts.Pem == "" {
		return fmt.Errorf("no TLS CA certificates found in connection profile for peer %s", peerName)
	}

	tlsCertPEM := peer.TLSCACerts.Pem
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM([]byte(tlsCertPEM)) {
		return fmt.Errorf("failed to append TLS certificate to cert pool from connection profile")
	}

	gw.logger.Info("TLS certificate loaded", "peer", peerName)

	// Get SSL target name override from grpc options or default to peer name
	sslTargetNameOverride := peerName
	if override, exists := peer.GRPCOptions["ssl-target-name-override"]; exists {
		sslTargetNameOverride = override
		gw.logger.Info("using SSL target name override", "override", sslTargetNameOverride)
	}

	// Create gRPC connection with TLS
	grpcCredentials := credentials.NewClientTLSFromCert(certPool, sslTargetNameOverride)

	gw.logger.Info("connecting to peer", "endpoint", peerEndpoint, "server_name", sslTargetNameOverride)

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dialCancel()

	//nolint:staticcheck // grpc.DialContext deprecated in gRPC 1.69+ but WithBlock requires it
	grpcConnection, err := grpc.DialContext(
		dialCtx,
		peerEndpoint,
		grpc.WithTransportCredentials(grpcCredentials),
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                120 * time.Second,
			Timeout:             20 * time.Second,
			PermitWithoutStream: false,
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection to %s within 30s: %w", peerEndpoint, err)
	}

	gw.logger.Info("gRPC connection established", "endpoint", peerEndpoint)

	// Load identity from wallet/crypto directory
	gw.logger.Info("loading identity", "mspid", org.MSPID)
	id, sign, err := gw.loadIdentity(org.MSPID)
	if err != nil {
		return fmt.Errorf("failed to load identity: %w", err)
	}
	gw.logger.Info("identity loaded", "mspid", org.MSPID)

	// Create gateway
	gw.logger.Info("creating fabric gateway connection")
	gw.gateway, err = client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(grpcConnection),
		client.WithEvaluateTimeout(30*time.Second),
		client.WithEndorseTimeout(30*time.Second),
		client.WithSubmitTimeout(30*time.Second),
		client.WithCommitStatusTimeout(60*time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to create gateway: %w", err)
	}
	gw.logger.Info("fabric gateway created")
	gw.network = gw.gateway.GetNetwork(gw.channel)
	gw.logger.Info("network obtained", "channel", gw.channel)
	gw.contract = gw.network.GetContract(gw.chaincode)
	gw.logger.Info("contract obtained", "chaincode", gw.chaincode)

	return nil
}

// loadIdentity loads the identity from the wallet/crypto directory
func (gw *Gateway) loadIdentity(mspID string) (*identity.X509Identity, identity.Sign, error) {
	// Use the tlsCertPath as the base directory for crypto files
	certPath := filepath.Join(gw.tlsCertPath, "signcerts", "cert.pem")
	keyPath := filepath.Join(gw.tlsCertPath, "keystore", "priv_sk")

	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate from %s: %w", certPath, err)
	}

	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key from %s: %w", keyPath, err)
	}

	certificate, err := identity.CertificateFromPEM(certBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate from PEM: %w", err)
	}

	privateKey, err := identity.PrivateKeyFromPEM(keyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create private key from PEM: %w", err)
	}

	// Create X509Identity using the MSPID from connection profile
	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create X509Identity: %w", err)
	}

	// Create a sign function using the private key
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create sign function: %w", err)
	}

	return id, sign, nil
}

// GetContract returns the contract for interacting with chaincode
func (gw *Gateway) GetContract() *client.Contract {
	return gw.contract
}

// Close closes the gateway connection
func (gw *Gateway) Close() {
	if gw.gateway != nil {
		gw.gateway.Close()
		gw.logger.Info("gateway connection closed")
	}
}

// GetChannel returns the channel ID
func (gw *Gateway) GetChannel() string {
	return gw.channel
}

// GetChaincode returns the chaincode ID
func (gw *Gateway) GetChaincode() string {
	return gw.chaincode
}

// extractEndorseError pulls the chaincode error message from gRPC status Details
// when the gateway returns "see attached details for more info".
func extractEndorseError(err error) error {
	var endorseErr *client.EndorseError
	if !errors.As(err, &endorseErr) {
		return err
	}
	for _, detail := range status.Convert(endorseErr).Details() {
		if ed, ok := detail.(*gwproto.ErrorDetail); ok && ed.GetMessage() != "" {
			return fmt.Errorf("%s", ed.GetMessage())
		}
	}
	return err
}

// SubmitTransaction submits a transaction and waits for commit confirmation.
func (gw *Gateway) SubmitTransaction(function string, args ...string) ([]byte, error) {
	gw.logger.Info("submitting transaction", "function", function, "args", args)

	result, commit, err := gw.contract.SubmitAsync(function, client.WithArguments(args...))
	if err != nil {
		return nil, fmt.Errorf("failed to submit transaction %s: %w", function, extractEndorseError(err))
	}

	status, err := commit.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit status for %s: %w", function, err)
	}
	if !status.Successful {
		return nil, fmt.Errorf("transaction %s committed with failure code: %v", function, status.Code)
	}

	gw.logger.Info("transaction committed", "function", function, "block", status.BlockNumber)
	return result, nil
}

// EvaluateTransaction evaluates a transaction query
func (gw *Gateway) EvaluateTransaction(function string, args ...string) ([]byte, error) {
	gw.logger.Info("evaluating transaction", "function", function)

	if gw.contract == nil {
		return nil, fmt.Errorf("contract is nil, gateway connection may not be properly initialized")
	}

	result, err := gw.contract.EvaluateTransaction(function, args...)
	if err != nil {
		gw.logger.Error("evaluate transaction failed", "function", function, "error", err)
		return nil, fmt.Errorf("failed to evaluate transaction: %w", err)
	}

	gw.logger.Info("transaction evaluated", "function", function, "result_bytes", len(result))
	return result, nil
}

// GetNetwork returns the network
func (gw *Gateway) GetNetwork() *client.Network {
	return gw.network
}

// GetConnectionProfile returns the loaded connection profile
func (gw *Gateway) GetConnectionProfile() *ConnectionProfile {
	return gw.connectionProfile
}
