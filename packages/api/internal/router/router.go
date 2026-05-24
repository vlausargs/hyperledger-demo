package router

import (
	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
	"github.com/myindo/hlf-supply-chain/api/internal/handler"
	"github.com/myindo/hlf-supply-chain/api/internal/middleware"
)

// Setup registers all middleware and routes on the given engine.
func Setup(r *gin.Engine, gw handler.FabricGateway, caClient *fabric.CAClient, corsOrigin, walletPath, jwtSecret, mspID string) {
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(corsOrigin))
	r.Use(middleware.CorrelationID())
	r.Use(middleware.RequestLogger())

	// Health check endpoint (no auth required)
	r.GET("/health", handler.HealthCheck(gw))

	// Auth endpoints (no JWT required)
	r.POST("/api/v1/auth/login", handler.Login(jwtSecret, mspID))

	// API v1 routes — all protected by JWT
	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWT(jwtSecret))
	{
		// Product endpoints
		products := v1.Group("/products")
		{
			products.GET("", handler.GetAllProducts(gw))
			products.POST("", handler.CreateProduct(gw))
			products.GET("/batch/:batchId", handler.GetProductsByBatch(gw))
			products.GET("/status/:status", handler.GetProductsByStatus(gw))
			products.GET("/:id", handler.GetProduct(gw))
			products.PUT("/:id", handler.UpdateProduct(gw))
			products.GET("/:id/history", handler.GetProductHistory(gw))
			products.GET("/:id/provenance", handler.GetProductProvenance(gw))
		}

		// Shipment endpoints
		shipments := v1.Group("/shipments")
		{
			shipments.GET("", handler.GetAllShipments(gw))
			shipments.POST("", handler.CreateShipment(gw))
			shipments.GET("/status/:status", handler.GetShipmentsByStatus(gw))
			shipments.GET("/:id", handler.GetShipment(gw))
			shipments.POST("/:id/dispatch", handler.DispatchShipment(gw))
			shipments.GET("/:id/history", handler.GetShipmentHistory(gw))
			shipments.GET("/:id/custody", handler.GetCustodyChain(gw))
			shipments.POST("/:id/custody/initiate", handler.InitiateCustodyTransfer(gw))
			shipments.POST("/:id/custody/accept", handler.AcceptCustodyTransfer(gw))
			shipments.POST("/:id/custody/reject", handler.RejectCustodyTransfer(gw))
		}

		// Event endpoints
		events := v1.Group("/events")
		{
			events.POST("", handler.LogEvent(gw))
			events.GET("/:targetId", handler.GetEvents(gw))
		}

		// Recall endpoints
		recalls := v1.Group("/recalls")
		{
			recalls.POST("", handler.IssueRecall(gw))
			recalls.GET("/products", handler.GetRecalledProducts(gw))
			recalls.GET("/:id", handler.ReadRecall(gw))
		}

		// POS endpoints (Org3 Retailer)
		pos := v1.Group("/pos")
		{
			pos.POST("/sales", handler.CreateSale(gw))
			pos.GET("/sales", handler.GetAllSales(gw))
			pos.GET("/sales/:id", handler.GetSale(gw))
			pos.GET("/inventory", handler.GetInventory(gw))
			pos.GET("/verify/:id", handler.VerifyProduct(gw))
		}

		// Channel endpoints
		channels := v1.Group("/channels")
		{
			channels.GET("", handler.GetChannels(gw))
			channels.GET("/:channelId", handler.GetChannelInfo(gw))
		}

		// Chaincode endpoints
		chaincodes := v1.Group("/chaincodes")
		{
			chaincodes.GET("", handler.GetChaincodes(gw))
			chaincodes.GET("/:chaincodeId", handler.GetChaincodeInfo(gw))
		}

		// Transaction endpoints
		transactions := v1.Group("/transactions")
		{
			transactions.GET("", handler.GetTransactions(gw))
			transactions.GET("/:txId", handler.GetTransaction(gw))
		}

		// Network endpoints
		network := v1.Group("/network")
		{
			network.GET("/peers", handler.GetPeers(gw))
			network.GET("/organizations", handler.GetOrganizations(gw))
			network.GET("/connection-profile", handler.GetConnectionProfile(gw))
		}

		// Identity endpoints (CA management) — only if CA client is available
		if caClient != nil {
			identities := v1.Group("/identities")
			{
				identities.GET("", handler.GetIdentities(caClient))
				identities.GET("/:id", handler.GetIdentity(caClient))
				identities.POST("/register", handler.RegisterIdentity(caClient))
				identities.POST("/enroll", handler.EnrollIdentity(caClient, walletPath))
				identities.DELETE("/:id", handler.DeleteIdentity(caClient, walletPath))
			}
		}
	}
}
