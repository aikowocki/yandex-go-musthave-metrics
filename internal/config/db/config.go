package db

import (
	"flag"
)

type Config struct {
	DatabaseDSN string `env:"DATABASE_DSN"`
}

func NewDBConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connection DSN")

	return cfg
}
