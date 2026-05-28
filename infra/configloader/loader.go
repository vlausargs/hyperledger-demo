package configloader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type NetworkConfig struct {
	FabricVersion     string        `yaml:"fabric_version"`
	CAVersion         string        `yaml:"ca_version"`
	Channel           string        `yaml:"channel"`
	Orderer           OrdererConfig `yaml:"orderer"`
	EndorsementPolicy string        `yaml:"endorsement_policy"`
}

type OrdererConfig struct {
	Type   string `yaml:"type"`
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	Domain string `yaml:"domain"`
}

type OrgConfig struct {
	Name       string         `yaml:"name"`
	Role       string         `yaml:"role"`
	Domain     string         `yaml:"domain"`
	Peer       PeerConfig     `yaml:"peer"`
	CA         CAConfig       `yaml:"ca"`
	API        APIConfig      `yaml:"api"`
	Postgres   PostgresConfig `yaml:"postgres"`
	Identities IdentityConfig `yaml:"identities"`
}

type PeerConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	CouchDBPort int    `yaml:"couchdb_port"`
	MetricsPort int    `yaml:"metrics_port"`
}

type CAConfig struct {
	Port    int    `yaml:"port"`
	AdminID string `yaml:"admin_id"`
}

type APIConfig struct {
	Port int `yaml:"port"`
}

type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
}

type IdentityConfig struct {
	Admin IdentityEntry   `yaml:"admin"`
	Users []IdentityEntry `yaml:"users"`
	Peer  IdentityEntry   `yaml:"peer"`
}

type IdentityEntry struct {
	ID string `yaml:"id"`
}

type FullConfig struct {
	Network NetworkConfig
	Orgs    []OrgConfig
}

func Load(configDir string) (*FullConfig, error) {
	networkPath := filepath.Join(configDir, "network.yml")
	networkData, err := os.ReadFile(networkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read network config: %w", err)
	}

	var network NetworkConfig
	if err := yaml.Unmarshal(networkData, &network); err != nil {
		return nil, fmt.Errorf("failed to parse network config: %w", err)
	}

	orgsDir := filepath.Join(configDir, "orgs")
	entries, err := os.ReadDir(orgsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read orgs directory: %w", err)
	}

	var orgs []OrgConfig
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		orgData, err := os.ReadFile(filepath.Join(orgsDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read org config %s: %w", entry.Name(), err)
		}
		var org OrgConfig
		if err := yaml.Unmarshal(orgData, &org); err != nil {
			return nil, fmt.Errorf("failed to parse org config %s: %w", entry.Name(), err)
		}
		orgs = append(orgs, org)
	}

	if len(orgs) == 0 {
		return nil, fmt.Errorf("no org config files found in %s", orgsDir)
	}

	return &FullConfig{Network: network, Orgs: orgs}, nil
}

func (c *FullConfig) GenerateEndorsementPolicy() string {
	switch c.Network.EndorsementPolicy {
	case "all":
		var parts []string
		for _, org := range c.Orgs {
			parts = append(parts, fmt.Sprintf("'%s.member'", org.Name))
		}
		return fmt.Sprintf("AND(%s)", strings.Join(parts, ","))
	case "majority":
		n := len(c.Orgs)/2 + 1
		var parts []string
		for _, org := range c.Orgs {
			parts = append(parts, fmt.Sprintf("'%s.member'", org.Name))
		}
		return fmt.Sprintf("OutOf(%d,%s)", n, strings.Join(parts, ","))
	case "any":
		var parts []string
		for _, org := range c.Orgs {
			parts = append(parts, fmt.Sprintf("'%s.member'", org.Name))
		}
		return fmt.Sprintf("OR(%s)", strings.Join(parts, ","))
	default:
		return c.Network.EndorsementPolicy
	}
}

func (c *FullConfig) Validate() error {
	if c.Network.Channel == "" {
		return fmt.Errorf("network.channel is required")
	}
	if c.Network.FabricVersion == "" {
		return fmt.Errorf("network.fabric_version is required")
	}
	if c.Network.Orderer.Port == 0 {
		return fmt.Errorf("network.orderer.port is required")
	}

	seen := make(map[string]bool)
	for _, org := range c.Orgs {
		if org.Name == "" {
			return fmt.Errorf("org name is required")
		}
		if seen[org.Name] {
			return fmt.Errorf("duplicate org name: %s", org.Name)
		}
		seen[org.Name] = true
		if org.Peer.Port == 0 {
			return fmt.Errorf("org %s: peer.port is required", org.Name)
		}
		if org.CA.Port == 0 {
			return fmt.Errorf("org %s: ca.port is required", org.Name)
		}
		if org.API.Port == 0 {
			return fmt.Errorf("org %s: api.port is required", org.Name)
		}
	}
	return nil
}

func (c *FullConfig) GetOrgByRole(role string) *OrgConfig {
	for i := range c.Orgs {
		if c.Orgs[i].Role == role {
			return &c.Orgs[i]
		}
	}
	return nil
}
