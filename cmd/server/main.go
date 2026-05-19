package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	application, err := app.NewServerApp(ctx, cfg)
	if err != nil {
		zap.S().Fatalw("failed to init app", "error", err)
	}
	defer application.Close()

	go application.Run(ctx)

	<-ctx.Done()

	zap.S().Infow("shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err = application.Shutdown(shutdownCtx); err != nil {
		zap.S().Fatalw("shutdown server error", "error", err)
	}
	zap.S().Infow("server stopped")
}
