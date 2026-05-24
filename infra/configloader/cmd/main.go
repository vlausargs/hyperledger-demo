package main

import (
	"fmt"
	"os"
	"path/filepath"

	configloader "github.com/myindo/hlf-supply-chain/configloader"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: configloader <validate|policy|list-orgs>")
		os.Exit(1)
	}

	configDir := findConfigDir()
	cfg, err := configloader.Load(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		if err := cfg.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Config valid.")
		fmt.Printf("Network: %s (Fabric %s)\n", cfg.Network.Channel, cfg.Network.FabricVersion)
		fmt.Printf("Orgs: %d\n", len(cfg.Orgs))
		for _, org := range cfg.Orgs {
			fmt.Printf("  - %s (%s) peer:%d api:%d\n", org.Name, org.Role, org.Peer.Port, org.API.Port)
		}
	case "policy":
		fmt.Println(cfg.GenerateEndorsementPolicy())
	case "list-orgs":
		for _, org := range cfg.Orgs {
			fmt.Printf("%s\t%s\t%s\n", org.Name, org.Role, org.Domain)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func findConfigDir() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "infra", "config")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return "infra/config"
}
