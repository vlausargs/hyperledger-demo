package router

import (
	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/handler"
	"github.com/myindo/hlf-supply-chain/api/internal/middleware"
	"github.com/myindo/hlf-supply-chain/api/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Setup wires every middleware and route. It accepts a fully constructed
// *service.Services bundle (built in cmd/server/main.go) so handlers stay
// decoupled from Fabric construction details.
func Setup(r *gin.Engine, svcs *service.Services, corsOrigin string, jwtSecret string) {
	r.Use(gin.Recovery())
	r.Use(middleware.Metrics())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RateLimit(100, 200))
	r.Use(middleware.CORS(corsOrigin))
	r.Use(middleware.CorrelationID())
	r.Use(middleware.RequestLogger())

	// Build per-domain handler structs from the services bundle.
	productH := handler.NewProductHandler(svcs.Product)
	shipmentH := handler.NewShipmentHandler(svcs.Shipment, svcs.Custody)
	eventH := handler.NewEventHandler(svcs.Event)
	recallH := handler.NewRecallHandler(svcs.Recall)
	posH := handler.NewPOSHandler(svcs.Sale, svcs.POS)
	networkH := handler.NewNetworkHandler(svcs.Network)
	authH := handler.NewAuthHandler(svcs.Auth)
	healthH := handler.NewHealthHandler(svcs.Health)

	// Metrics endpoint (no auth required)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check endpoint (no auth required)
	r.GET("/health", healthH.Check)

	// Auth endpoints (no JWT required)
	r.POST("/api/v1/auth/login", authH.Login)

	// API v1 routes — all protected by JWT
	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWT(jwtSecret))
	v1.Use(middleware.AuditLog())
	{
		// Product endpoints
		products := v1.Group("/products")
		{
			products.GET("", productH.List)
			products.POST("", productH.Create)
			products.GET("/batch/:batchId", productH.ByBatch)
			products.GET("/status/:status", productH.ByStatus)
			products.GET("/:id", productH.Get)
			products.PUT("/:id", productH.Update)
			products.GET("/:id/history", productH.History)
			products.GET("/:id/provenance", productH.Provenance)
		}

		// Shipment endpoints
		shipments := v1.Group("/shipments")
		{
			shipments.GET("", shipmentH.List)
			shipments.POST("", shipmentH.Create)
			shipments.GET("/status/:status", shipmentH.ByStatus)
			shipments.GET("/:id", shipmentH.Get)
			shipments.POST("/:id/dispatch", shipmentH.Dispatch)
			shipments.GET("/:id/history", shipmentH.History)
			shipments.GET("/:id/custody", shipmentH.CustodyChain)
			shipments.POST("/:id/custody/initiate", shipmentH.InitiateCustody)
			shipments.POST("/:id/custody/accept", shipmentH.AcceptCustody)
			shipments.POST("/:id/custody/reject", shipmentH.RejectCustody)
		}

		// Event endpoints
		events := v1.Group("/events")
		{
			events.POST("", eventH.Log)
			events.GET("/:targetId", eventH.List)
		}

		// Recall endpoints
		recalls := v1.Group("/recalls")
		{
			recalls.POST("", recallH.Issue)
			recalls.GET("/products", recallH.RecalledProducts)
			recalls.GET("/:id", recallH.Get)
		}

		// POS endpoints (Org3 Retailer)
		pos := v1.Group("/pos")
		{
			pos.POST("/sales", posH.CreateSale)
			pos.GET("/sales", posH.ListSales)
			pos.GET("/sales/:id", posH.GetSale)
			pos.GET("/inventory", posH.Inventory)
			pos.GET("/verify/:id", posH.VerifyProduct)
		}

		// Channel endpoints
		channels := v1.Group("/channels")
		{
			channels.GET("", networkH.Channels)
			channels.GET("/:channelId", networkH.ChannelInfo)
		}

		// Chaincode endpoints
		chaincodes := v1.Group("/chaincodes")
		{
			chaincodes.GET("", networkH.Chaincodes)
			chaincodes.GET("/:chaincodeId", networkH.ChaincodeInfo)
		}

		// Transaction endpoints
		transactions := v1.Group("/transactions")
		{
			transactions.GET("", networkH.Transactions)
			transactions.GET("/:txId", networkH.Transaction)
		}

		// Network endpoints
		network := v1.Group("/network")
		{
			network.GET("/peers", networkH.Peers)
			network.GET("/organizations", networkH.Organizations)
			network.GET("/connection-profile", networkH.ConnectionProfile)
		}

		// Identity endpoints — only mount if CA client is available
		if svcs.Identity.Available() {
			identityH := handler.NewIdentityHandler(svcs.Identity)
			identities := v1.Group("/identities")
			{
				identities.GET("", identityH.List)
				identities.GET("/:id", identityH.Get)
				identities.POST("/register", identityH.Register)
				identities.POST("/enroll", identityH.Enroll)
				identities.DELETE("/:id", identityH.Delete)
			}
		}
	}
}
