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
	defer cancel()

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

	// wgProducers — все, кто пишет в jobs или живёт по ctx (сборщики + отправитель).
	// wgWorkers   — пул воркеров, читающих jobs.
	// Порядок shutdown: cancel -> wgProducers.Wait -> close(jobs) -> wgWorkers.Wait.
	var wgProducers, wgWorkers sync.WaitGroup

	// Воркер-пул (консьюмеры).
	for i := 0; i < cfg.RateLimit; i++ {
		wgWorkers.Go(func() {
			for job := range jobs {
				agent.SendBatch(ctx, client, job)
			}
		})
	}

	wgProducers.Go(func() {
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
	})

	// Сборщик системных метрик.
	wgProducers.Go(func() {
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
	})

	// Отправитель — единственный продюсер jobs.
	// Перед выходом делает финальный flush, чтобы не потерять последнюю пачку.
	wgProducers.Go(func() {
		ticker := time.NewTicker(time.Duration(cfg.ReportInterval))
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if metrics := agent.CollectBatch(storage); len(metrics) > 0 {
					jobs <- metrics
				}
			case <-ctx.Done():
				if metrics := agent.CollectBatch(storage); len(metrics) > 0 {
					jobs <- metrics
				}
				return
			}
		}
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	cancel()           // 1. сигнализируем всем горутинам остановиться
	wgProducers.Wait() // 2. ждём, пока продюсеры закончат и больше никто не пишет в jobs
	close(jobs)        // 3. безопасно закрываем канал
	wgWorkers.Wait()   // 4. воркеры дочитают остатки и выйдут из for range
}
