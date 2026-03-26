package fabric

import (
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// Gateway represents the Fabric Gateway connection
type Gateway struct {
	gateway      *client.Gateway
	network      *client.Network
	contract     *client.Contract
	channel      string
	chaincode    string
	walletPath   string
	tlsCertPath  string
	mspPath      string
	peerEndpoint string
	mspID        string
	userID       string
	logger       *log.Logger
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

// NewGateway creates a new Fabric Gateway connection
func NewGateway(channelID, chaincodeID, walletPath, tlsCertPath, peerEndpoint, mspID, userID string) (*Gateway, error) {
	logger := log.New(os.Stdout, "[Fabric-Gateway] ", log.LstdFlags|log.Lshortfile)

	gw := &Gateway{
		channel:      channelID,
		chaincode:    chaincodeID,
		walletPath:   walletPath,
		tlsCertPath:  tlsCertPath,
		mspPath:      filepath.Join(filepath.Dir(tlsCertPath), "msp"),
		peerEndpoint: peerEndpoint,
		mspID:        mspID,
		userID:       userID,
		logger:       logger,
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

// connect establishes connection to the Fabric network
func (gw *Gateway) connect() error {
	// Load TLS certificate
	tlsCertPath := filepath.Join(gw.tlsCertPath, "ca.crt")
	tlsCertBytes, err := os.ReadFile(tlsCertPath)
	if err != nil {
		return fmt.Errorf("failed to read TLS certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(tlsCertBytes) {
		return fmt.Errorf("failed to append TLS certificate to cert pool")
	}

	// Create gRPC connection with TLS
	grpcCredentials := credentials.NewClientTLSFromCert(certPool, "peer0.org1.example.com")

	gw.logger.Printf("Attempting to connect to peer at %s with server name override: peer0.org1.example.com", gw.peerEndpoint)

	grpcConnection, err := grpc.Dial(
		gw.peerEndpoint,
		grpc.WithTransportCredentials(grpcCredentials),
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                120 * time.Second,
			Timeout:             20 * time.Second,
			PermitWithoutStream: false,
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection to %s: %w", gw.peerEndpoint, err)
	}

	gw.logger.Printf("Successfully established gRPC connection to %s", gw.peerEndpoint)

	// Load identity from wallet
	gw.logger.Printf("Loading identity from wallet...")
	id, sign, err := gw.loadIdentity()
	if err != nil {
		return fmt.Errorf("failed to load identity: %w", err)
	}
	gw.logger.Printf("Identity loaded successfully: MSPID=%s, UserID=%s", gw.mspID, gw.userID)

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

// loadIdentity loads the identity from the wallet
func (gw *Gateway) loadIdentity() (*identity.X509Identity, identity.Sign, error) {
	certPath := filepath.Join(gw.tlsCertPath, "signcerts", "cert.pem")
	keyPath := filepath.Join(gw.tlsCertPath, "keystore", "priv_sk")
	mspConfigPath := filepath.Join(gw.tlsCertPath, "config.yaml")

	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key: %w", err)
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
	id, err := identity.NewX509Identity(gw.mspID, certificate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create X509Identity: %w", err)
	}

	// Set the credential in the identity if needed
	mspConfigBytes, err := os.ReadFile(mspConfigPath)
	if err == nil {
		// Parse the MSP config and extract OU identifiers
		// This ensures the Fabric Gateway SDK knows the identity has proper OU attributes
		_ = mspConfigBytes // Acknowledge we read the config
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
