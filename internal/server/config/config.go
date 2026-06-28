package config

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/duration"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config/db"
	"github.com/caarlos0/env/v11"
)

type Seconds = duration.Seconds

type ServerConfig struct {
	ServerAddress   string  `env:"ADDRESS"           json:"address"`
	StoreInterval   Seconds `env:"STORE_INTERVAL"    json:"store_interval"`
	FileStoragePath string  `env:"FILE_STORAGE_PATH" json:"store_file"`
	Restore         bool    `env:"RESTORE"           json:"restore"`
	DB              *db.Config
	Key             string `env:"KEY"               json:"key"`
	CryptoKey       string `env:"CRYPTO_KEY"        json:"crypto_key"`
	AuditFile       string `env:"AUDIT_FILE"        json:"audit_file"`
	AuditURL        string `env:"AUDIT_URL"         json:"audit_url"`
	PprofAddress    string `env:"PPROF_ADDRESS"     json:"pprof_address"`
	TrustedSubnet   string `env:"TRUSTED_SUBNET"    json:"trusted_subnet"`
	ConfigFile      string `env:"CONFIG"            json:"-"`
}

// UnmarshalJSON Поле database_dsn лежит в JSON на верхнем уровне, а в структуре
// — внутри вложенного DB, поэтому используем alias и вспомогательное поле, чтобы
// избежать рекурсии и вручную пробросить значение в cfg.DB.
func (cfg *ServerConfig) UnmarshalJSON(data []byte) error {
	type alias ServerConfig
	aux := &struct {
		DatabaseDSN string `json:"database_dsn"`
		*alias
	}{
		alias: (*alias)(cfg),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.DatabaseDSN != "" && cfg.DB != nil {
		cfg.DB.DatabaseDSN = aux.DatabaseDSN
	}
	return nil
}

// NewServerConfig собирает конфигурацию сервера
func NewServerConfig() (*ServerConfig, error) {
	return parseServerConfig(os.Args[1:], env.ToMap(os.Environ()))
}

// parseServerConfig Приоритет: default -> JSON-config -> flags -> env.
func parseServerConfig(args []string, environ map[string]string) (*ServerConfig, error) {
	cfg := &ServerConfig{
		StoreInterval: Seconds(300 * time.Second), // Флаг Var не принимает значение по умолчанию. инициируем тут
		DB:            &db.Config{},
	}

	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server address")
	fs.Var(&cfg.StoreInterval, "i", "store interval in seconds")
	fs.StringVar(&cfg.FileStoragePath, "f", "backup/metrics.json", "store file storage path")
	fs.BoolVar(&cfg.Restore, "r", true, "store restore")
	fs.StringVar(&cfg.Key, "k", "", "hash key")
	fs.StringVar(&cfg.CryptoKey, "crypto-key", "", "path to private key file for RSA decryption")
	fs.StringVar(&cfg.AuditFile, "audit-file", "", "path to audit log file (empty disable file audit)")
	fs.StringVar(&cfg.AuditURL, "audit-url", "", "URL for HTTP audit sink (empty disable HTTP audit)")
	fs.StringVar(&cfg.PprofAddress, "pprof-address", "localhost:6060", "pprof address")
	fs.StringVar(&cfg.TrustedSubnet, "t", "", "trusted subnet (CIDR) (empty disables check)")
	fs.StringVar(&cfg.ConfigFile, "c", "", "path to JSON config file")
	fs.StringVar(&cfg.ConfigFile, "config", "", "path to JSON config file")
	fs.StringVar(&cfg.DB.DatabaseDSN, "d", "", "database connection DSN")
	fs.StringVar(&cfg.DB.PostgresDriver, "pgx", "pgx", "database driver (pgx)")

	// #1 парс: нужен, чтобы узнать путь к JSON-конфигу из флага -c/-config.
	// Парсим сразу весь набор флагов (а не отдельный FlagSet только с -c),
	// потому что стандартный flag прерывается на первом НЕизвестном флаге —
	// частичный FlagSet споткнулся бы на любом другом флаге раньше -c.
	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Путь к файлу: env CONFIG имеет приоритет над флагом -c.
	configPath := cfg.ConfigFile
	if v := environ["CONFIG"]; v != "" {
		configPath = v
	}

	if configPath != "" {
		if err := loadFileConfig(configPath, cfg); err != nil {
			return nil, err
		}
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
	}

	// Переменные окружения перезаписывают всё.
	if err := env.ParseWithOptions(cfg, env.Options{Environment: environ}); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadFileConfig читает JSON-файл и десериализует его прямо в конфиг.
func loadFileConfig(path string, cfg *ServerConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}
