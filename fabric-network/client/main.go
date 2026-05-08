package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hlf-demo/fabric-network/client/fabric"
	"hlf-demo/fabric-network/client/middleware"
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
	defaultConnectionProfile = "./crypto/connection-profile.yaml"
	defaultCAURL             = "http://localhost:8054"
	defaultCAName            = "ca-org1"
)

func main() {
	// Initialize structured logger
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("starting fabric client application")

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
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
		config.ConnectionProfile,
	)
	if err != nil {
		slog.Error("failed to initialize fabric gateway", "error", err)
		os.Exit(1)
	}
	defer fabricGateway.Close()

	slog.Info("connected to fabric gateway")

	// Initialize CA client (optional — warns if misconfigured, does not exit)
	caClient, err := fabric.NewCAClient(
		config.CAURL,
		config.CAName,
		config.CAAdminMSPDir,
		config.MSPID,
		config.WalletPath,
	)
	if err != nil {
		slog.Warn("CA client not available — identity management endpoints disabled", "error", err)
		caClient = nil
	}

	// Create Gin router
	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(correlationIDMiddleware())
	router.Use(requestLogger())

	// Health check endpoint (no auth required)
	router.GET("/health", rest.HealthCheck(fabricGateway))

	// Auth endpoints (no JWT required)
	router.POST("/api/v1/auth/login", rest.Login())

	// API v1 routes — all protected by JWT
	v1 := router.Group("/api/v1")
	v1.Use(middleware.JWTMiddleware())
	{
		// Product endpoints (Manufacturer creates, all orgs read)
		products := v1.Group("/products")
		{
			products.GET("", rest.GetAllProducts(fabricGateway))
			products.POST("", rest.CreateProduct(fabricGateway))
			products.GET("/batch/:batchId", rest.GetProductsByBatch(fabricGateway))
			products.GET("/status/:status", rest.GetProductsByStatus(fabricGateway))
			products.GET("/:id", rest.GetProduct(fabricGateway))
			products.PUT("/:id", rest.UpdateProduct(fabricGateway))
			products.GET("/:id/history", rest.GetProductHistory(fabricGateway))
			products.GET("/:id/provenance", rest.GetProductProvenance(fabricGateway))
		}

		// Shipment endpoints
		shipments := v1.Group("/shipments")
		{
			shipments.GET("", rest.GetAllShipments(fabricGateway))
			shipments.POST("", rest.CreateShipment(fabricGateway))
			shipments.GET("/status/:status", rest.GetShipmentsByStatus(fabricGateway))
			shipments.GET("/:id", rest.GetShipment(fabricGateway))
			shipments.POST("/:id/dispatch", rest.DispatchShipment(fabricGateway))
			shipments.GET("/:id/history", rest.GetShipmentHistory(fabricGateway))
			shipments.GET("/:id/custody", rest.GetCustodyChain(fabricGateway))
			shipments.POST("/:id/custody/initiate", rest.InitiateCustodyTransfer(fabricGateway))
			shipments.POST("/:id/custody/accept", rest.AcceptCustodyTransfer(fabricGateway))
			shipments.POST("/:id/custody/reject", rest.RejectCustodyTransfer(fabricGateway))
		}

		// Event endpoints
		events := v1.Group("/events")
		{
			events.POST("", rest.LogEvent(fabricGateway))
			events.GET("/:targetId", rest.GetEvents(fabricGateway))
		}

		// Recall endpoints
		recalls := v1.Group("/recalls")
		{
			recalls.POST("", rest.IssueRecall(fabricGateway))
			recalls.GET("/products", rest.GetRecalledProducts(fabricGateway))
			recalls.GET("/:id", rest.ReadRecall(fabricGateway))
		}

		// POS endpoints (Org3 Retailer)
		pos := v1.Group("/pos")
		{
			pos.POST("/sales", rest.CreateSale(fabricGateway))
			pos.GET("/sales", rest.GetAllSales(fabricGateway))
			pos.GET("/sales/:id", rest.GetSale(fabricGateway))
			pos.GET("/inventory", rest.GetInventory(fabricGateway))
			pos.GET("/verify/:id", rest.VerifyProduct(fabricGateway))
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

		// Transaction endpoints
		transactions := v1.Group("/transactions")
		{
			transactions.GET("", rest.GetTransactions(fabricGateway))
			transactions.GET("/:txId", rest.GetTransaction(fabricGateway))
		}

		// Network endpoints
		network := v1.Group("/network")
		{
			network.GET("/peers", rest.GetPeers(fabricGateway))
			network.GET("/organizations", rest.GetOrganizations(fabricGateway))
			network.GET("/connection-profile", getConnectionProfileHandler(fabricGateway))
		}

		// Identity endpoints (CA management) — only if CA client is available
		if caClient != nil {
			identities := v1.Group("/identities")
			{
				identities.GET("", rest.GetIdentities(caClient))
				identities.GET("/:id", rest.GetIdentity(caClient))
				identities.POST("/register", rest.RegisterIdentity(caClient))
				identities.POST("/enroll", rest.EnrollIdentity(caClient, config.WalletPath))
				identities.DELETE("/:id", rest.DeleteIdentity(caClient, config.WalletPath))
			}
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
		slog.Info("server starting", "port", config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

// Configuration structure
type Config struct {
	WalletPath        string
	ChannelID         string
	ChaincodeID       string
	Port              string
	Mode              string
	TLSCertPath       string
	ConnectionProfile string
	CAURL             string
	CAName            string
	CAAdminMSPDir     string
	MSPID             string
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
		ConnectionProfile: getEnv("CONNECTION_PROFILE", defaultConnectionProfile),
		CAURL:             getEnv("CA_URL", defaultCAURL),
		CAName:            getEnv("CA_NAME", defaultCAName),
		CAAdminMSPDir:     getEnv("CA_ADMIN_MSP_DIR", "./crypto/admin-msp"),
		MSPID:             getEnv("MSP_ID", "Org1MSP"),
	}

	// Validate required configuration
	if config.ChannelID == "" {
		return nil, fmt.Errorf("CHANNEL_ID is required")
	}
	if config.ChaincodeID == "" {
		return nil, fmt.Errorf("CHAINCODE_ID is required")
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
	allowedOrigin := getEnv("CORS_ALLOWED_ORIGIN", "http://localhost:3000")
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
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

// getConnectionProfileHandler returns the connection profile information
func getConnectionProfileHandler(gw *fabric.Gateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get connection profile from gateway
		connProfile := gw.GetConnectionProfile()
		if connProfile == nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Connection profile not loaded",
			})
			return
		}

		// Build a response with key information from the connection profile
		orgName := connProfile.Client.Organization
		org, orgExists := connProfile.Organizations[orgName]

		response := gin.H{
			"profile": gin.H{
				"name":    connProfile.Name,
				"version": connProfile.Version,
			},
			"client": gin.H{
				"organization": connProfile.Client.Organization,
			},
			"organizations": connProfile.Organizations,
			"peers": gin.H{
				"count": len(connProfile.Peers),
				"list":  getPeerInfo(connProfile),
			},
			"certificateAuthorities": gin.H{
				"count": len(connProfile.CAs),
				"list":  getCAInfo(connProfile),
			},
		}

		if orgExists {
			response["currentOrganization"] = gin.H{
				"name":  orgName,
				"mspid": org.MSPID,
				"peers": org.Peers,
				"cas":   org.CertificateAuthorities,
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

// getPeerInfo extracts peer information from connection profile
func getPeerInfo(cp *fabric.ConnectionProfile) []gin.H {
	var peers []gin.H
	for name, peer := range cp.Peers {
		peerInfo := gin.H{
			"name": name,
			"url":  peer.URL,
		}
		if peer.TLSCACerts.Pem != "" {
			peerInfo["hasTLSCert"] = true
			peerInfo["tlsCertLength"] = len(peer.TLSCACerts.Pem)
		} else {
			peerInfo["hasTLSCert"] = false
		}
		if peer.GRPCOptions != nil {
			peerInfo["grpcOptions"] = peer.GRPCOptions
		}
		peers = append(peers, peerInfo)
	}
	return peers
}

// getCAInfo extracts CA information from connection profile
func getCAInfo(cp *fabric.ConnectionProfile) []gin.H {
	var cas []gin.H
	for name, ca := range cp.CAs {
		caInfo := gin.H{
			"name":   name,
			"url":    ca.URL,
			"caName": ca.CAName,
		}
		if ca.TLSCACerts.Pem != "" {
			caInfo["hasTLSCert"] = true
			caInfo["tlsCertLength"] = len(ca.TLSCACerts.Pem)
		} else {
			caInfo["hasTLSCert"] = false
		}
		cas = append(cas, caInfo)
	}
	return cas
}

// correlationIDMiddleware attaches a correlation ID to every request.
// Reads X-Correlation-ID header or generates one; echoes it in the response.
func correlationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cid := c.GetHeader("X-Correlation-ID")
		if cid == "" {
			cid = newCorrelationID()
		}
		c.Set("correlationID", cid)
		c.Header("X-Correlation-ID", cid)
		c.Next()
	}
}

// newCorrelationID generates a random hex correlation ID.
func newCorrelationID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// requestLogger logs incoming requests as structured JSON.
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if q := c.Request.URL.RawQuery; q != "" {
			path = path + "?" + q
		}

		c.Next()

		cid, _ := c.Get("correlationID")
		slog.Info("request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
			"correlation_id", cid,
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
