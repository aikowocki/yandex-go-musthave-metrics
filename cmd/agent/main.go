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

	// Горутина для сбора метрик
	go func() {
		pollTicker := time.NewTicker(time.Duration(cfg.PollInterval))
		defer pollTicker.Stop()
		for range pollTicker.C {
			agent.CollectMetrics(storage)
		}
	}()
	// Горутина для отправки метрик
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.ReportInterval))
		defer ticker.Stop()
		for range ticker.C {
			agent.ReportJSON(storage, client)
		}
	}()
	// Блокируем main, что бы программа не завершилась
	select {}
}
