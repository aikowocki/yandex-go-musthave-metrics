package main

import (
	"log"
	"net/http"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/handler"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/logger"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/middleware"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/repository"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/service"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/storage/metric"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func main() {
	cleanup, err := logger.New()
	if err != nil {
		panic(err)
	}

	defer cleanup()

	cfg := config.NewServerConfig()

	stor := metric.NewMetricMemoryStorage()

	fileBackup := metric.NewFileBackup(cfg.FileStoragePath, stor, time.Duration(cfg.StoreInterval))

	if cfg.Restore {
		if err = fileBackup.Restore(); err != nil {
			zap.S().Fatalw("failed to restore backup", "error", err)
		}
	}
	fileBackup.Start()
	if cfg.StoreInterval == 0 {
		stor = metric.NewSyncBackupStorage(stor, fileBackup)
	}
	repo := repository.NewMetricRepository(stor)

	svc := service.NewMetricService(repo)

	metricHandler := handler.NewMetricHandler(svc)

	r := chi.NewRouter()
	r.Use(chimw.StripSlashes)
	r.Use(middleware.WithLogging())
	r.Use(middleware.WithGzipCompression())

	r.Post("/update/{type}/{name}/{value}", metricHandler.Update)

	//REST
	r.Group(func(r chi.Router) {
		r.Use(chimw.AllowContentType("application/json"))
		r.Post("/update", metricHandler.UpdateJSON)
		r.Post("/value", metricHandler.GetJSON)
	})

	r.Get("/value/{type}/{name}", metricHandler.Get)
	r.Get("/", metricHandler.List)
	log.Printf("Server starting on port: %s", cfg.ServerAddress)
	zap.S().Infow(
		"Server starting",
		"address", cfg.ServerAddress,
	)

	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		zap.S().Fatalw(err.Error(), "event", "start server")

	}

}
