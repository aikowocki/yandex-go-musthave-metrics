package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/agent"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/logger"
	"github.com/joho/godotenv"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	if err := godotenv.Load(); err != nil {
		log.Println("failed to load .env", "error", err)
	}

	loggerCleanup, err := logger.New("agent")
	if err != nil {
		log.Fatal(err)
	}
	defer loggerCleanup()

	cfg := config.NewAgentConfig()
	storage := agent.NewLocalStorage()
	client := agent.NewClient("http://"+cfg.ServerAddress, agent.WithServerKey(cfg.Key))

	// Горутина для сбора метрик
	go func() {
		pollTicker := time.NewTicker(time.Duration(cfg.PollInterval))
		defer pollTicker.Stop()
		for {
			select {
			case <-pollTicker.C:
				agent.CollectMetrics(storage)
			case <-ctx.Done():
				return
			}
		}
	}()
	// Горутина для отправки метрик
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.ReportInterval))
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				agent.ReportBatch(storage, client)
			case <-ctx.Done():
				return
			}
		}
	}()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	cancel()
	agent.ReportBatch(storage, client)
}
