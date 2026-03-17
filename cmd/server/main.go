package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	ctx, cancel := context.WithCancel(context.Background())
	loggerCleanup, err := logger.New()
	if err != nil {
		log.Fatal(err)
	}
	defer loggerCleanup()

	if err = godotenv.Load(); err != nil {
		zap.S().Warnw("failed to load .env", "error", err)
	}

	cfg := config.NewServerConfig()

	db, dbCleanup := getDB(*cfg)
	defer dbCleanup()

	storage := getStorage(ctx, db, *cfg)
	repo := repository.NewMetricRepository(storage)

	svc := service.NewMetricService(repo)

	r := setupRouter(handler.NewMetricHandler(svc), handler.NewHealthcheck(db))

	zap.S().Infow(
		"Server starting",
		"address", cfg.ServerAddress,
	)

	srv := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			zap.S().Fatalw(err.Error(), "event", "start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
	zap.S().Infow("shutting down...")
	shutdownContext, shutdownСancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownСancel()
	if err = srv.Shutdown(shutdownContext); err != nil {
		zap.S().Fatalw(err.Error(), "event", "Shutdown server error")
	}
	cancel()
	zap.S().Infow("server stopped")
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

func getDB(cfg config.ServerConfig) (*sql.DB, func()) {
	var db *sql.DB
	var err error

	if cfg.DB.DatabaseDSN != "" {
		db, err = database.NewPostgresDB(cfg.DB.DatabaseDSN)
		if err != nil {
			zap.S().Error(err)
		}
		if db != nil {
			if err := database.RunMigrations(db, "file://migrations"); err != nil {
				zap.S().Fatalw("failed to run migrations", "error", err)
			}
		}

	}

	cleanup := func() {
		if db != nil {
			db.Close()
		}
	}

	return db, cleanup
}

func getStorage(ctx context.Context, db *sql.DB, cfg config.ServerConfig) metric.Storage {

	var storage metric.Storage

	if db != nil {
		storage = metric.NewMetricPostgresStorage(db)
	} else {
		memStorage := metric.NewMetricMemoryStorage()
		storage = memStorage

		if cfg.FileStoragePath != "" {

			fileBackup := metric.NewFileBackup(cfg.FileStoragePath, memStorage, time.Duration(cfg.StoreInterval))

			if cfg.Restore {
				if err := fileBackup.Restore(); err != nil {
					zap.S().Fatalw("failed to restore backup", "error", err)
				}
			}

			fileBackup.Start(ctx)

			if cfg.StoreInterval == 0 {
				storage = metric.NewSyncBackupStorage(memStorage, fileBackup)
			}

		}
	}
	return storage
}
