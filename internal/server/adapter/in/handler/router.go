package handler

import (
	"crypto/rsa"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/in/handler/middleware"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// RouterOptions содержит опциональную конфигурацию для HTTP-роутера.
type RouterOptions struct {
	CryptoPrivateKey *rsa.PrivateKey
}

// NewRouter собирает HTTP-роутер со всеми эндпоинтами и middleware.
// Порядок middleware: StripSlashes → RequestID → MaxBody → Logging → Gzip → Decrypt → Hash.
func NewRouter(
	metric *MetricHandler,
	metricJSON *MetricJSONHandler,
	health *HealthHandler,
	key string,
	opts ...RouterOptions,
) *chi.Mux {
	var opt RouterOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	r := chi.NewRouter()
	r.Use(chimw.StripSlashes)
	r.Use(chimw.RequestID)
	r.Use(
		middleware.WithMaxBodySize(middleware.DefaultMaxBodyBytes),
	) // ограничение размера сырого тела до gzip/hash/handler
	r.Use(middleware.WithLogging())                        // порядок важен
	r.Use(middleware.WithGzipCompression())                // gzip сначала декомпрессирует тело
	r.Use(middleware.WithDecryption(opt.CryptoPrivateKey)) // потом расшифровка
	r.Use(
		middleware.WithHashValidation(key),
	) // потом hash middleware проверяет хеш от уже расшифрованного тела

	//r.Use(middleware.WithRecovery) //toDO

	r.Post("/update/{type}/{name}/{value}", metric.Update)

	//REST
	r.Group(func(r chi.Router) {
		r.Use(chimw.AllowContentType("application/json"))
		r.Post("/update", metricJSON.Update)
		r.Post("/updates", metricJSON.BatchUpdate)
		r.Post("/value", metricJSON.Get)
	})

	r.Get("/value/{type}/{name}", metric.Get)
	r.Get("/ping", health.Ping)
	r.Get("/", metric.List)

	return r
}
