package config

import (
	"flag"
	"os"
)

type ServerConfig struct {
	Address string
}

func NewServerConfig() *ServerConfig {
	addr := flag.String("a", "localhost:8080", "server address")
	flag.Parse()

	cfg := &ServerConfig{
		Address: *addr,
	}

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.Address = envAddr
	}

	return cfg
}
