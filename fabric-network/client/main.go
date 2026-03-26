package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hlf-demo/fabric-network/client/fabric"
	"hlf-demo/fabric-network/client/rest"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-gateway/pkg/client"
)

const (
	defaultWalletPath        = "./wallet"
	defaultChannelID         = "mychannel"
	defaultChaincodeID       = "basic"
	defaultServerPort        = "8080"
	defaultTLSCertPath       = "./crypto"
	defaultPeerEndpoint      = "localhost:8051"
	defaultGatewayPeer       = "peer0.org1.example.com"
	defaultConnectionProfile = "./crypto/connection-profile.yaml"
)

func main() {
	// Initialize logger
	logger := log.New(os.Stdout, "[Fabric-Client] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting Fabric Client Application...")

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode
	if config.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Fabric gateway connection
	fabricGateway, err := fabric.NewGateway(
		config.ChannelID,
		config.ChaincodeID,
		config.WalletPath,
		config.TLSCertPath,
		config.PeerEndpoint,
		config.MSPID,
		config.UserID,
	)
	if err != nil {
		logger.Fatalf("Failed to initialize Fabric gateway: %v", err)
	}
	defer fabricGateway.Close()

	logger.Println("Successfully connected to Fabric gateway")

	// Create Gin router
	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(requestLogger())

	// Health check endpoint
	router.GET("/health", rest.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Asset endpoints
		assets := v1.Group("/assets")
		{
			assets.GET("", rest.GetAllAssets(fabricGateway))
			assets.GET("/:id", rest.GetAsset(fabricGateway))
			assets.POST("", rest.CreateAsset(fabricGateway))
			assets.PUT("/:id", rest.UpdateAsset(fabricGateway))
			assets.DELETE("/:id", rest.DeleteAsset(fabricGateway))
			assets.POST("/:id/transfer", rest.TransferAsset(fabricGateway))
			assets.GET("/:id/history", rest.GetAssetHistory(fabricGateway))
			assets.GET("/range", rest.GetAssetsByRange(fabricGateway))
		}

		// Channel endpoints
		channels := v1.Group("/channels")
		{
			channels.GET("", rest.GetChannels(fabricGateway))
			channels.GET("/:channelId", rest.GetChannelInfo(fabricGateway))
		}

		// Chaincode endpoints
		chaincodes := v1.Group("/chaincodes")
		{
			chaincodes.GET("", rest.GetChaincodes(fabricGateway))
			chaincodes.GET("/:chaincodeId", rest.GetChaincodeInfo(fabricGateway))
		}

		// Network endpoints
		network := v1.Group("/network")
		{
			network.GET("/peers", rest.GetPeers(fabricGateway))
			network.GET("/organizations", rest.GetOrganizations(fabricGateway))
		}

		// Transaction endpoints
		transactions := v1.Group("/transactions")
		{
			transactions.GET("", rest.GetTransactions(fabricGateway))
			transactions.GET("/:txId", rest.GetTransaction(fabricGateway))
		}
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", config.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Printf("Server starting on port %s...", config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server stopped")
}

// Configuration structure
type Config struct {
	WalletPath        string
	ChannelID         string
	ChaincodeID       string
	Port              string
	Mode              string
	TLSCertPath       string
	PeerEndpoint      string
	GatewayPeer       string
	MSPID             string
	UserID            string
	ConnectionProfile string
	OrgName           string
}

// loadConfig loads configuration from environment variables
func loadConfig() (*Config, error) {
	config := &Config{
		WalletPath:        getEnv("WALLET_PATH", defaultWalletPath),
		ChannelID:         getEnv("CHANNEL_ID", defaultChannelID),
		ChaincodeID:       getEnv("CHAINCODE_ID", defaultChaincodeID),
		Port:              getEnv("SERVER_PORT", defaultServerPort),
		Mode:              getEnv("GIN_MODE", "debug"),
		TLSCertPath:       getEnv("TLS_CERT_PATH", defaultTLSCertPath),
		PeerEndpoint:      getEnv("PEER_ENDPOINT", defaultPeerEndpoint),
		GatewayPeer:       getEnv("GATEWAY_PEER", defaultGatewayPeer),
		MSPID:             getEnv("MSP_ID", "Org1MSP"),
		UserID:            getEnv("USER_ID", "appUser"),
		ConnectionProfile: getEnv("CONNECTION_PROFILE", defaultConnectionProfile),
		OrgName:           getEnv("ORG_NAME", "Org1"),
	}

	// Validate required configuration
	if config.ChannelID == "" {
		return nil, fmt.Errorf("CHANNEL_ID is required")
	}
	if config.ChaincodeID == "" {
		return nil, fmt.Errorf("CHAINCODE_ID is required")
	}
	if config.MSPID == "" {
		return nil, fmt.Errorf("MSP_ID is required")
	}

	return config, nil
}

// getEnv retrieves environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// corsMiddleware handles CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// requestLogger logs incoming requests
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()

		if query != "" {
			path = path + "?" + query
		}

		log.Printf("[%s] %s | %d | %v | %s | %s",
			method,
			path,
			statusCode,
			latency,
			clientIP,
			userAgent,
		)
	}
}

// FabricGateway interface for mocking in tests
type FabricGateway interface {
	GetContract() *client.Contract
	Close()
	GetChannel() string
	GetChaincode() string
}
