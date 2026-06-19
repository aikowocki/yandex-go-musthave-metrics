package config

import (
	"flag"
	"strconv"
	"time"

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

type AgentConfig struct {
	ServerAddress  string  `env:"ADDRESS"`
	ReportInterval Seconds `env:"REPORT_INTERVAL"`
	PollInterval   Seconds `env:"POLL_INTERVAL"`
	Key            string  `env:"KEY"`
	CryptoKey      string  `env:"CRYPTO_KEY"`
	RateLimit      int     `env:"RATE_LIMIT"`
	PprofAddress   string  `env:"PPROF_ADDRESS"`
}

func NewAgentConfig() (*AgentConfig, error) {
	cfg := &AgentConfig{
		ReportInterval: Seconds(10 * time.Second),
		PollInterval:   Seconds(2 * time.Second),
		RateLimit:      1,
	}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server address")
	flag.Var(&cfg.ReportInterval, "r", "report interval in seconds")
	flag.Var(&cfg.PollInterval, "p", "poll interval in seconds")
	flag.StringVar(&cfg.Key, "k", "", "hash key")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", "", "path to public key file for RSA encryption")
	flag.IntVar(&cfg.RateLimit, "l", 1, "rate limit for concurrent requests")
	flag.StringVar(&cfg.PprofAddress, "pprof-address", "localhost:6061", "pprof server address")

	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
