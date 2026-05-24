package config_test

import (
	"os"
	"testing"

	"github.com/myindo/hlf-supply-chain/api/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long!!")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.ChannelID != "mychannel" {
		t.Errorf("expected default channel mychannel, got %s", cfg.ChannelID)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	os.Clearenv()

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET not set")
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	os.Clearenv()
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("CHANNEL_ID", "testchannel")
	os.Setenv("JWT_SECRET", "test-secret-at-least-32-chars-long!!")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.ChannelID != "testchannel" {
		t.Errorf("expected channel testchannel, got %s", cfg.ChannelID)
	}
}
