package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Root     string
	HTTPAddr string
}

func Default() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return Config{
		Root:     filepath.Join(home, "agent_companion_data"),
		HTTPAddr: "127.0.0.1:9000",
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
	return cfg
}
