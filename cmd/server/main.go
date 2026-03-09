package main

import (
	"log"
	"net/http"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/database"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/handler"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/logger"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/middleware"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/repository"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/service"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/storage/metric"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	cleanup, err := logger.New()
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	if err = godotenv.Load(); err != nil {
		zap.S().Warnw("failed to load .env", "error", err)
	}

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

	db, err := database.NewPostgresDB(cfg.DB.DatabaseDSN)
	if err != nil {
		zap.S().Error(err)
	}
	if db != nil {
		defer db.Close()
	}

	r := setupRouter(handler.NewMetricHandler(svc), handler.NewHealthcheck(db))

	zap.S().Infow(
		"Server starting",
		"address", cfg.ServerAddress,
	)

	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		zap.S().Fatalw(err.Error(), "event", "start server")

	}
}

func setupRouter(h *handler.MetricHandler, hc *handler.Healthcheck) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.StripSlashes)
	r.Use(middleware.WithLogging())
	r.Use(middleware.WithGzipCompression())

	r.Post("/update/{type}/{name}/{value}", h.Update)

	//REST
	r.Group(func(r chi.Router) {
		r.Use(chimw.AllowContentType("application/json"))
		r.Post("/update", h.UpdateJSON)
		r.Post("/value", h.GetJSON)
	})

	r.Get("/value/{type}/{name}", h.Get)
	r.Get("/ping", hc.Ping)
	r.Get("/", h.List)

	return r
}
