package main

import (
	"fmt"
	"os"
	"path/filepath"

	configloader "github.com/myindo/hlf-supply-chain/configloader"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: configloader <validate|policy|list-orgs|generate-compose>")
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
	case "generate-compose":
		// TODO: render per-org compose fragment (peer, couchdb, ca, postgres,
		// api) by templating compose.fabric.yml + compose.api.yml against the
		// org's YAML config. Stub prints what would be generated.
		orgFlag := ""
		for _, a := range os.Args[2:] {
			if len(a) > 6 && a[:6] == "--org=" {
				orgFlag = a[6:]
			}
		}
		if orgFlag == "" {
			fmt.Fprintln(os.Stderr, "generate-compose requires --org=<name>")
			os.Exit(1)
		}
		var match *struct{}
		for _, org := range cfg.Orgs {
			if org.Name == orgFlag || org.Domain == orgFlag {
				fmt.Printf("# Would generate compose for %s (%s) peer:%d api:%d ca:%d postgres:%d\n",
					org.Name, org.Role, org.Peer.Port, org.API.Port, org.CA.Port, org.Postgres.Port)
				match = &struct{}{}
				break
			}
		}
		if match == nil {
			fmt.Fprintf(os.Stderr, "Org %q not found in infra/config/orgs/\n", orgFlag)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func findConfigDir() string {
	dir, _ := os.Getwd()
	for range 10 {
		candidate := filepath.Join(dir, "infra", "config")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	return "infra/config"
}
