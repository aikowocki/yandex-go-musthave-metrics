package main

import (
	"log"
	"net/http"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/config"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/handler"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/repository"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/service"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/storage/metric"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.NewServerConfig()

	stor := metric.NewMetricMemoryStorage()

	repo := repository.NewMetricRepository(stor)

	svc := service.NewMetricService(repo)

	metricHandler := handler.NewMetricHandler(svc)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", metricHandler.Update)
	r.Get("/value/{type}/{name}", metricHandler.Get)
	r.Get("/", metricHandler.List)
	log.Printf("Server starting on port: %s", cfg.Address)
	if err := http.ListenAndServe(cfg.Address, r); err != nil {
		log.Fatal(err)
	}

}
