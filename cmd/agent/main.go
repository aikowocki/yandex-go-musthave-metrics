package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/agent"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/agent/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/api"
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

	cfg, err := config.NewAgentConfig()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatal("failed to load .env", err)
		}
	}

	storage := agent.NewLocalStorage()
	client := agent.NewClient("http://"+cfg.ServerAddress, agent.WithServerKey(cfg.Key))

	jobs := make(chan []api.MetricDTO, cfg.RateLimit)
	var wg sync.WaitGroup

	for i := 0; i < cfg.RateLimit; i++ { // запускаем N воркеров
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				agent.SendBatch(ctx, client, job)
			}
		}()
	}

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

	// Горутина для сбора системных метрик
	go func() {
		pollTicker := time.NewTicker(time.Duration(cfg.PollInterval))
		defer pollTicker.Stop()
		for {
			select {
			case <-pollTicker.C:
				agent.CollectSystemMetrics(storage)
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
				metrics := agent.CollectBatch(storage)
				if len(metrics) > 0 {
					jobs <- metrics
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	cancel()
	if metrics := agent.CollectBatch(storage); len(metrics) > 0 {
		jobs <- metrics
	}
	close(jobs)
	wg.Wait()
}
