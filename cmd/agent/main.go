package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
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
	"go.uber.org/zap"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

func main() {
	printBuildInfo()
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

	go func() {
		zap.S().Infow("pprof starting", "address", cfg.PprofAddress)
		if pprofErr := http.ListenAndServe(cfg.PprofAddress, nil); pprofErr != nil {
			zap.S().Errorw("pprof server failed", "error", pprofErr)
		}
	}()

	storage := agent.NewLocalStorage()
	clientOpts := []agent.ClientOption{agent.WithServerKey(cfg.Key)}
	if cfg.CryptoKey != "" {
		clientOpts = append(clientOpts, agent.WithCryptoKey(cfg.CryptoKey))
	}

	var sender agent.MetricSender

	if cfg.GRPCAddress != "" {
		grpcClient, err := agent.NewGRPCClient(cfg.GRPCAddress)
		if err != nil {
			log.Fatal("failed to create gRPC client", err)
		}
		sender = grpcClient
	} else {
		sender = agent.NewClient("http://"+cfg.ServerAddress, clientOpts...)
	}
	// Закрываем транспорт после остановки воркеров (graceful path).
	defer func() {
		if err := sender.Close(); err != nil {
			zap.S().Errorw("failed to close sender", "error", err)
		}
	}()

	jobs := make(chan []api.MetricDTO, cfg.RateLimit)

	// wgProducers — все, кто пишет в jobs или живёт по ctx (сборщики + отправитель).
	// wgWorkers   — пул воркеров, читающих jobs.
	// Порядок shutdown: cancel -> wgProducers.Wait -> close(jobs) -> wgWorkers.Wait.
	var wgProducers, wgWorkers sync.WaitGroup

	// Воркер-пул (консьюмеры).
	for i := 0; i < cfg.RateLimit; i++ {
		wgWorkers.Go(func() {
			for job := range jobs {
				agent.SendBatch(ctx, sender, job)
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

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigCh
	log.Println("shutting down gracefully, press Ctrl+C again to force exit")

	cancel() // сигнализируем всем горутинам остановиться

	// Graceful shutdown выполняем в отдельной горутине, чтобы main мог
	// одновременно слушать второй сигнал.
	done := make(chan struct{})
	go func() {
		defer close(done)
		wgProducers.Wait() // ждём, пока продюсеры закончат и больше никто не пишет в jobs
		close(jobs)        // безопасно закрываем канал
		wgWorkers.Wait()   // воркеры дочитают остатки и выйдут из for range
	}()

	select {
	case <-done:
		// graceful shutdown завершился сам — последняя пачка отправлена.
	case <-sigCh:
		// второй сигнал пришёл раньше — выходим немедленно.
		// os.Exit не выполняет defer (loggerCleanup, cancel) т.к форсим завершение.
		log.Println("forced exit")
		os.Exit(1)
	}
}
