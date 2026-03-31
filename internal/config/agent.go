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
}

func NewAgentConfig() *AgentConfig {
	cfg := &AgentConfig{
		ReportInterval: Seconds(10 * time.Second),
		PollInterval:   Seconds(2 * time.Second),
	}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "server address")
	flag.Var(&cfg.ReportInterval, "r", "report interval in seconds")
	flag.Var(&cfg.PollInterval, "p", "poll interval in seconds")
	flag.StringVar(&cfg.Key, "k", "", "hash key")

	flag.Parse()

	env.Parse(cfg)

	return cfg
}
