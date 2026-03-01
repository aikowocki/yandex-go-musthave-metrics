package main

import (
	"log"
	"net/http"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/handler"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/logger"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/middleware"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/repository"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/service"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/storage/metric"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	zapLogger, cleanup, err := logger.New()
	if err != nil {
		panic(err)
	}

	defer cleanup()

	cfg := config.NewServerConfig()

	stor := metric.NewMetricMemoryStorage()

	repo := repository.NewMetricRepository(stor)

	svc := service.NewMetricService(repo)

	metricHandler := handler.NewMetricHandler(svc)

	r := chi.NewRouter()
	r.Use(chimw.StripSlashes)
	r.Use(middleware.WithLogging(zapLogger))
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
	log.Printf("Server starting on port: %s", cfg.Address)
	zapLogger.Infow(
		"Server starting",
		"address", cfg.Address,
	)
	if err := http.ListenAndServe(cfg.Address, r); err != nil {
		zapLogger.Fatalw(err.Error(), "event", "start server")

	}

}
