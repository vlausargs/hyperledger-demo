package configloader_test

import (
	"os"
	"path/filepath"
	"testing"

	configloader "github.com/myindo/hlf-supply-chain/configloader"
)

func TestLoad(t *testing.T) {
	// Use the actual config files
	configDir := findConfigDir(t)

	cfg, err := configloader.Load(configDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Network.Channel != "mychannel" {
		t.Errorf("expected channel mychannel, got %s", cfg.Network.Channel)
	}
	if len(cfg.Orgs) != 3 {
		t.Errorf("expected 3 orgs, got %d", len(cfg.Orgs))
	}
}

func TestValidate(t *testing.T) {
	configDir := findConfigDir(t)
	cfg, err := configloader.Load(configDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate failed: %v", err)
	}
}

func TestGenerateEndorsementPolicy_All(t *testing.T) {
	configDir := findConfigDir(t)
	cfg, err := configloader.Load(configDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	policy := cfg.GenerateEndorsementPolicy()
	if policy == "" {
		t.Error("expected non-empty policy")
	}
	// Should contain AND and all 3 org MSPs
	t.Logf("Generated policy: %s", policy)
}

func TestGetOrgByRole(t *testing.T) {
	configDir := findConfigDir(t)
	cfg, err := configloader.Load(configDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	mfg := cfg.GetOrgByRole("manufacturer")
	if mfg == nil {
		t.Fatal("expected manufacturer org")
	}
	if mfg.Name != "Org1MSP" {
		t.Errorf("expected Org1MSP, got %s", mfg.Name)
	}
}

func findConfigDir(t *testing.T) string {
	// Walk up to find infra/config relative to this test file
	dir, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, "infra", "config")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("could not find infra/config directory")
	return ""
}
