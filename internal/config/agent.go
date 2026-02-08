package config

import (
	"flag"
	"time"
)

type AgentConfig struct {
	ServerAddress  string
	ReportInterval time.Duration
	PollInterval   time.Duration
}

func NewAgentConfig() *AgentConfig {
	addr := flag.String("a", "localhost:8080", "server address")
	reportSec := flag.Int("r", 10, "report interval in seconds")
	pollSec := flag.Int("p", 2, "poll interval in seconds")

	flag.Parse()

	return &AgentConfig{
		ServerAddress:  *addr,
		ReportInterval: time.Duration(*reportSec) * time.Second,
		PollInterval:   time.Duration(*pollSec) * time.Second,
	}
}
