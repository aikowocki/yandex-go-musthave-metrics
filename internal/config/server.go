package config

import (
	"flag"
	"time"

	"github.com/caarlos0/env/v11"
)

type ServerConfig struct {
	ServerAddress   string  `env:"ADDRESS"`
	StoreInterval   Seconds `env:"STORE_INTERVAL"`
	FileStoragePath string  `env:"FILE_STORAGE_PATH"`
	Restore         bool    `env:"RESTORE"`
}

func NewServerConfig() *ServerConfig {
	cfg := &ServerConfig{
		StoreInterval: Seconds(300 * time.Second), // Флаг Var не принимает значение по умолчанию. инициируем тут
	}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server address")
	flag.Var(&cfg.StoreInterval, "i", "store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", "backup/metrics.json", "store file storage path")
	flag.BoolVar(&cfg.Restore, "r", true, "store restore")

	flag.Parse()

	env.Parse(cfg)

	return cfg
}
