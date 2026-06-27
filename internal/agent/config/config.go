package config

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/duration"
	"github.com/caarlos0/env/v11"
)

type Seconds = duration.Seconds

type AgentConfig struct {
	ServerAddress  string  `env:"ADDRESS"         json:"address"`
	ReportInterval Seconds `env:"REPORT_INTERVAL" json:"report_interval"`
	PollInterval   Seconds `env:"POLL_INTERVAL"   json:"poll_interval"`
	Key            string  `env:"KEY"             json:"key"`
	CryptoKey      string  `env:"CRYPTO_KEY"      json:"crypto_key"`
	RateLimit      int     `env:"RATE_LIMIT"      json:"rate_limit"`
	PprofAddress   string  `env:"PPROF_ADDRESS"   json:"pprof_address"`
	ConfigFile     string  `env:"CONFIG"          json:"-"`
}

// NewAgentConfig собирает конфигурацию агента
func NewAgentConfig() (*AgentConfig, error) {
	return parseAgentConfig(os.Args[1:], env.ToMap(os.Environ()))
}

// parseAgentConfig Приоритет: default -> JSON-config -> flags -> env.
func parseAgentConfig(args []string, environ map[string]string) (*AgentConfig, error) {
	cfg := &AgentConfig{
		ReportInterval: Seconds(10 * time.Second), // Флаг Var не принимает значение по умолчанию. инициируем тут
		PollInterval:   Seconds(2 * time.Second),
		RateLimit:      1,
	}

	// Локальный FlagSet вместо глобального flag.CommandLine: можно создавать сколько
	// угодно раз без паники "flag redefined", а ContinueOnError возвращает ошибку
	// вместо os.Exit.
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server address")
	fs.Var(&cfg.ReportInterval, "r", "report interval in seconds")
	fs.Var(&cfg.PollInterval, "p", "poll interval in seconds")
	fs.StringVar(&cfg.Key, "k", "", "hash key")
	fs.StringVar(&cfg.CryptoKey, "crypto-key", "", "path to public key file for RSA encryption")
	fs.IntVar(&cfg.RateLimit, "l", 1, "rate limit for concurrent requests")
	fs.StringVar(&cfg.PprofAddress, "pprof-address", "localhost:6061", "pprof server address")
	fs.StringVar(&cfg.ConfigFile, "c", "", "path to JSON config file")
	fs.StringVar(&cfg.ConfigFile, "config", "", "path to JSON config file")

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
	// flags > JSON config
	if configPath != "" {
		if err := loadFileConfig(configPath, cfg); err != nil {
			return nil, err
		}
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
	}

	// env > flags
	if err := env.ParseWithOptions(cfg, env.Options{Environment: environ}); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadFileConfig читает JSON-файл и десериализует его прямо в конфиг.
func loadFileConfig(path string, cfg *AgentConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}
