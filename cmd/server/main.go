package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/logger"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/app"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/config"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := godotenv.Load(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			// файл найден, но битый
			log.Fatal("failed to load .env", err)
		}
	}

	loggerCleanup, err := logger.New("server")
	if err != nil {
		log.Fatal(err)
	}
	defer loggerCleanup()

	cfg, err := config.NewServerConfig()
	if err != nil {
		zap.S().Fatalw("failed to load config", "error", err)
	}

	go func() {
		zap.S().Infow("pprof starting", "address", cfg.PprofAddress)
		if err := http.ListenAndServe(cfg.PprofAddress, nil); err != nil {
			zap.S().Errorw("pprof server failed", "error", err)
		}
	}()

	application, err := app.NewServerApp(ctx, cfg)
	if err != nil {
		zap.S().Fatalw("failed to init app", "error", err)
	}

	go application.Run(ctx)

	<-ctx.Done()

	zap.S().Infow("shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Сначала останавливаем приём HTTP-запросов, затем освобождаем ресурсы.
	// Close вызываем безусловно (даже при ошибке Shutdown) и с тем же ctx,
	// чтобы весь graceful shutdown укладывался в единый бюджет времени.
	if err = application.Shutdown(shutdownCtx); err != nil {
		zap.S().Errorw("shutdown server error", "error", err)
	}
	application.Close(shutdownCtx)
	zap.S().Infow("server stopped")
}
