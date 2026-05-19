package config

import (
	"flag"
	"strconv"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config/db"
	"github.com/caarlos0/env/v11"
)

type Seconds time.Duration

func (s *Seconds) UnmarshalText(text []byte) error {
	v, err := strconv.Atoi(string(text))
	if err != nil {
		return err
	}
	*s = Seconds(time.Duration(v) * time.Second)
	return nil
}
func (s *Seconds) String() string {
	return strconv.Itoa(int(time.Duration(*s) / time.Second))
}

func (s *Seconds) Set(val string) error {
	return s.UnmarshalText([]byte(val))
}

type ServerConfig struct {
	ServerAddress   string  `env:"ADDRESS"`
	StoreInterval   Seconds `env:"STORE_INTERVAL"`
	FileStoragePath string  `env:"FILE_STORAGE_PATH"`
	Restore         bool    `env:"RESTORE"`
	DB              *db.Config
	Key             string `env:"KEY"`
}

func NewServerConfig() (*ServerConfig, error) {
	cfg := &ServerConfig{
		StoreInterval: Seconds(300 * time.Second), // Флаг Var не принимает значение по умолчанию. инициируем тут
		DB:            db.NewDBConfig(),
	}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server address")
	flag.Var(&cfg.StoreInterval, "i", "store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", "backup/metrics.json", "store file storage path")
	flag.BoolVar(&cfg.Restore, "r", true, "store restore")
	flag.StringVar(&cfg.Key, "k", "", "hash key")

	flag.Parse()
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
