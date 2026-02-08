package main

import (
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/agent"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/config"
)

func main() {
	cfg := config.NewAgentConfig()
	storage := agent.NewLocalStorage()
	client := agent.NewClient("http://" + cfg.ServerAddress)

	pollInterval := int(cfg.PollInterval.Seconds())
	reportInterval := int(cfg.ReportInterval.Seconds())
	i := 0
	for {
		if i%pollInterval == 0 {
			agent.CollectMetrics(storage)
		}
		if i%reportInterval == 0 {
			agent.Report(storage, client)
		}
		i++
		time.Sleep(time.Second)
	}
}
