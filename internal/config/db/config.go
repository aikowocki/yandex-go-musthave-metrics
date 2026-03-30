package db

import (
	"flag"
)

type Config struct {
	DatabaseDSN string `env:"DATABASE_DSN"`
	UsePgxPool  bool   `env:"USE_PGX_POOL"`
}

func NewDBConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connection DSN")
	flag.BoolVar(&cfg.UsePgxPool, "pgx", false, "use pgx pool")

	return cfg
}
