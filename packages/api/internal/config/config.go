package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port              string
	Mode              string
	ChannelID         string
	ChaincodeID       string
	WalletPath        string
	TLSCertPath       string
	ConnectionProfile string
	CAURL             string
	CAName            string
	CAAdminMSPDir     string
	MSPID             string
	JWTSecret         string
	CORSAllowedOrigin string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:              env("SERVER_PORT", "8080"),
		Mode:              env("GIN_MODE", "debug"),
		ChannelID:         env("CHANNEL_ID", "mychannel"),
		ChaincodeID:       env("CHAINCODE_ID", "basic"),
		WalletPath:        env("WALLET_PATH", "./wallet"),
		TLSCertPath:       env("TLS_CERT_PATH", "./crypto"),
		ConnectionProfile: env("CONNECTION_PROFILE", "./crypto/connection-profile.yaml"),
		CAURL:             env("CA_URL", "http://localhost:8054"),
		CAName:            env("CA_NAME", "ca-org1"),
		CAAdminMSPDir:     env("CA_ADMIN_MSP_DIR", "./crypto/admin-msp"),
		MSPID:             env("MSP_ID", "Org1MSP"),
		JWTSecret:         env("JWT_SECRET", ""),
		CORSAllowedOrigin: env("CORS_ALLOWED_ORIGIN", "http://localhost:3000"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.ChannelID == "" {
		return fmt.Errorf("CHANNEL_ID is required")
	}
	if c.ChaincodeID == "" {
		return fmt.Errorf("CHAINCODE_ID is required")
	}
	return nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
