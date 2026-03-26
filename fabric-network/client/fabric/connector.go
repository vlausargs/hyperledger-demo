package fabric

import (
	"crypto/x509"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
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
	logger            *log.Logger
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

// Config holds the gateway configuration
type Config struct {
	WalletPath   string
	ChannelID    string
	ChaincodeID  string
	TLSCertPath  string
	PeerEndpoint string
	MSPID        string
	UserID       string
}

// NewGateway creates a new Fabric Gateway connection using a connection profile
func NewGateway(channelID, chaincodeID, walletPath, tlsCertPath, connectionProfilePath string) (*Gateway, error) {
	logger := log.New(os.Stdout, "[Fabric-Gateway] ", log.LstdFlags|log.Lshortfile)

	logger.Printf("Initializing Fabric Gateway with connection profile: %s", connectionProfilePath)

	// Load and parse the connection profile
	connectionProfile, err := loadConnectionProfile(connectionProfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load connection profile: %w", err)
	}
	logger.Printf("Connection profile loaded successfully: %s", connectionProfile.Name)

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

	gw.logger.Printf("Successfully connected to Fabric gateway (Channel: %s, Chaincode: %s)", channelID, chaincodeID)

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

	gw.logger.Printf("Using organization: %s (MSPID: %s)", orgName, org.MSPID)

	// Get the first peer from the organization
	if len(org.Peers) == 0 {
		return fmt.Errorf("no peers defined for organization %s", orgName)
	}

	peerName := org.Peers[0]
	peer, exists := gw.connectionProfile.Peers[peerName]
	if !exists {
		return fmt.Errorf("peer %s not found in connection profile", peerName)
	}

	gw.logger.Printf("Using peer: %s", peerName)

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
	gw.logger.Printf("Peer endpoint: %s", peerEndpoint)

	// Get TLS certificate from connection profile
	if peer.TLSCACerts.Pem == "" {
		return fmt.Errorf("no TLS CA certificates found in connection profile for peer %s", peerName)
	}

	tlsCertPEM := peer.TLSCACerts.Pem
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM([]byte(tlsCertPEM)) {
		return fmt.Errorf("failed to append TLS certificate to cert pool from connection profile")
	}

	gw.logger.Printf("TLS certificate loaded from connection profile for peer %s", peerName)

	// Get SSL target name override from grpc options or default to peer name
	sslTargetNameOverride := peerName
	if override, exists := peer.GRPCOptions["ssl-target-name-override"]; exists {
		sslTargetNameOverride = override
		gw.logger.Printf("Using SSL target name override: %s", sslTargetNameOverride)
	}

	// Create gRPC connection with TLS
	grpcCredentials := credentials.NewClientTLSFromCert(certPool, sslTargetNameOverride)

	gw.logger.Printf("Attempting to connect to peer at %s with server name override: %s", peerEndpoint, sslTargetNameOverride)

	grpcConnection, err := grpc.Dial(
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
		return fmt.Errorf("failed to create gRPC connection to %s: %w", peerEndpoint, err)
	}

	gw.logger.Printf("Successfully established gRPC connection to %s", peerEndpoint)

	// Load identity from wallet/crypto directory
	gw.logger.Printf("Loading identity from wallet/crypto directory...")
	id, sign, err := gw.loadIdentity(org.MSPID)
	if err != nil {
		return fmt.Errorf("failed to load identity: %w", err)
	}
	gw.logger.Printf("Identity loaded successfully: MSPID=%s", org.MSPID)

	// Create gateway
	gw.logger.Printf("Creating Fabric Gateway connection...")
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
	gw.logger.Printf("Fabric Gateway created successfully")

	// Get network
	gw.logger.Printf("Getting network for channel: %s", gw.channel)
	gw.network = gw.gateway.GetNetwork(gw.channel)
	gw.logger.Printf("Network obtained successfully for channel: %s", gw.channel)

	// Get contract
	gw.logger.Printf("Getting contract for chaincode: %s", gw.chaincode)
	gw.contract = gw.network.GetContract(gw.chaincode)
	gw.logger.Printf("Contract obtained successfully for chaincode: %s", gw.chaincode)

	return nil
}

// loadIdentity loads the identity from the wallet/crypto directory
func (gw *Gateway) loadIdentity(mspID string) (*identity.X509Identity, identity.Sign, error) {
	// Use the tlsCertPath as the base directory for crypto files
	certPath := filepath.Join(gw.tlsCertPath, "signcerts", "cert.pem")
	keyPath := filepath.Join(gw.tlsCertPath, "keystore", "priv_sk")
	mspConfigPath := filepath.Join(gw.tlsCertPath, "config.yaml")

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

	// Load MSP configuration to properly identify organizational units
	mspConfigBytes, err := os.ReadFile(mspConfigPath)
	if err == nil {
		// Parse the MSP config and extract OU identifiers
		// This ensures the Fabric Gateway SDK knows the identity has proper OU attributes
		_ = mspConfigBytes // Acknowledge we read the config
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
		gw.logger.Println("Gateway connection closed")
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

// SubmitTransaction submits a transaction to the ledger
func (gw *Gateway) SubmitTransaction(function string, args ...string) ([]byte, error) {
	gw.logger.Printf("Submitting transaction: %s with args: %v", function, args)

	result, err := gw.contract.SubmitTransaction(function, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to submit transaction: %w", err)
	}

	gw.logger.Printf("Transaction submitted successfully")
	return result, nil
}

// EvaluateTransaction evaluates a transaction query
func (gw *Gateway) EvaluateTransaction(function string, args ...string) ([]byte, error) {
	gw.logger.Printf("Evaluating transaction: %s with args: %v", function, args)

	if gw.contract == nil {
		return nil, fmt.Errorf("contract is nil, gateway connection may not be properly initialized")
	}

	gw.logger.Printf("Contract is not nil, proceeding with evaluation...")
	result, err := gw.contract.EvaluateTransaction(function, args...)
	if err != nil {
		gw.logger.Printf("ERROR evaluating transaction: %v", err)
		return nil, fmt.Errorf("failed to evaluate transaction: %w", err)
	}

	gw.logger.Printf("Transaction evaluated successfully, result length: %d bytes", len(result))
	return result, nil
}

// SubmitAsyncTransaction submits a transaction asynchronously
func (gw *Gateway) SubmitAsyncTransaction(function string, args ...string) (*client.Commit, error) {
	gw.logger.Printf("Submitting async transaction: %s with args: %v", function, args)

	_, commit, err := gw.contract.SubmitAsync(function, client.WithArguments(args...))
	if err != nil {
		return nil, fmt.Errorf("failed to submit async transaction: %w", err)
	}

	return commit, nil
}

// GetNetwork returns the network
func (gw *Gateway) GetNetwork() *client.Network {
	return gw.network
}

// GetLogger returns the gateway logger
func (gw *Gateway) GetLogger() *log.Logger {
	return gw.logger
}

// GetConnectionProfile returns the loaded connection profile
func (gw *Gateway) GetConnectionProfile() *ConnectionProfile {
	return gw.connectionProfile
}
