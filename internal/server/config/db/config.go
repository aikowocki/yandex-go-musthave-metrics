package db

import (
	"flag"
)

type Config struct {
	DatabaseDSN    string `env:"DATABASE_DSN"`
	PostgresDriver string `env:"POSTGRES_DRIVER"`
}

func NewDBConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connection DSN")
	flag.StringVar(&cfg.PostgresDriver, "pgx", "pgx", "database driver (pgx)")

	return cfg
}
