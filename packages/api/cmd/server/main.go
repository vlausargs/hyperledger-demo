package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/myindo/hlf-supply-chain/api/internal/config"
	"github.com/myindo/hlf-supply-chain/api/internal/fabric"
	"github.com/myindo/hlf-supply-chain/api/internal/router"
)

func main() {
	// Initialize structured JSON logger
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	slog.Info("starting fabric client application")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Set Gin mode
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Fabric gateway connection
	fabricGateway, err := fabric.NewGateway(
		cfg.ChannelID,
		cfg.ChaincodeID,
		cfg.WalletPath,
		cfg.TLSCertPath,
		cfg.ConnectionProfile,
	)
	if err != nil {
		slog.Error("failed to initialize fabric gateway", "error", err)
		os.Exit(1)
	}
	defer fabricGateway.Close()

	slog.Info("connected to fabric gateway")

	// Initialize CA client (optional — warns if misconfigured, does not exit)
	caClient, err := fabric.NewCAClient(
		cfg.CAURL,
		cfg.CAName,
		cfg.CAAdminMSPDir,
		cfg.MSPID,
		cfg.WalletPath,
	)
	if err != nil {
		slog.Warn("CA client not available — identity management endpoints disabled", "error", err)
		caClient = nil
	}

	// Create Gin router and register all routes
	r := gin.New()
	router.Setup(r, fabricGateway, caClient, cfg.CORSAllowedOrigin, cfg.WalletPath, cfg.JWTSecret, cfg.MSPID)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("server starting", "port", cfg.Port)
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
