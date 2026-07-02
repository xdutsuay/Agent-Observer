package config

import (
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Root     string
	HTTPAddr string
	ApiKeys  map[string]string // Maps apiKey -> tenantID
}

func Default() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return Config{
		Root:     filepath.Join(home, "agent_companion_data"),
		HTTPAddr: "127.0.0.1:9000",
		ApiKeys:  make(map[string]string),
	}
}

func Load() Config {
	cfg := Default()
	if root := os.Getenv("AGENT_MEMORY_ROOT"); root != "" {
		cfg.Root = root
	}
	if addr := os.Getenv("AGENT_MEMORY_HTTP_ADDR"); addr != "" {
		cfg.HTTPAddr = addr
	}
	
	if keys := os.Getenv("AGENT_MEMORY_API_KEYS"); keys != "" {
		// format: "tenantA:key1,tenantB:key2"
		pairs := strings.Split(keys, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, ":", 2)
			if len(parts) == 2 {
				tenantID := strings.TrimSpace(parts[0])
				apiKey := strings.TrimSpace(parts[1])
				cfg.ApiKeys[apiKey] = tenantID
			}
		}
	}
	
	return cfg
}
