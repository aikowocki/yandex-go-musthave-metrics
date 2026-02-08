package config

import "flag"

type ServerConfig struct {
	Address string
}

func NewServerConfig() *ServerConfig {
	addr := flag.String("a", "localhost:8080", "server address")

	flag.Parse()

	return &ServerConfig{
		Address: *addr,
	}
}
