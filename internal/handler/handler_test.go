package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/repository"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/service"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/storage/metric"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestHandler_Update(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		url            string
		wantStatusCode int
	}{
		{
			name:           "success gauge",
			method:         "POST",
			url:            "/update/gauge/test/3.14",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "success counter",
			method:         "POST",
			url:            "/update/counter/test/5",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid metric type",
			method:         "POST",
			url:            "/update/unknown/test/1",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid gauge value",
			method:         "POST",
			url:            "/update/gauge/test/abc",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invalid counter value",
			method:         "POST",
			url:            "/update/counter/test/3.14",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаём storage и repository
			stor := metric.NewMetricMemoryStorage()
			repo := repository.NewMetricRepository(stor)
			svc := service.NewMetricService(repo)
			handler := NewMetricHandler(svc)

			// Создаём роутер с chi
			r := chi.NewRouter()
			r.Post("/update/{type}/{name}/{value}", handler.Update)

			req := httptest.NewRequest(tt.method, tt.url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

func TestHandler_Get(t *testing.T) {
	tests := []struct {
		name           string
		setupMetric    func(*repository.MetricRepository)
		url            string
		wantStatusCode int
		wantBody       string
	}{
		{
			name: "get gauge success",
			setupMetric: func(repo *repository.MetricRepository) {
				repo.Save(&model.GaugeMetric{Name: "cpu", Value: 0.5})
			},
			url:            "/value/gauge/cpu",
			wantStatusCode: http.StatusOK,
			wantBody:       "0.5",
		},
		{
			name: "get counter success",
			setupMetric: func(repo *repository.MetricRepository) {
				repo.Save(&model.CounterMetric{Name: "requests", Value: 10})
			},
			url:            "/value/counter/requests",
			wantStatusCode: http.StatusOK,
			wantBody:       "10",
		},
		{
			name:           "get not found",
			setupMetric:    func(repo *repository.MetricRepository) {},
			url:            "/value/gauge/nonexistent",
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stor := metric.NewMetricMemoryStorage()
			repo := repository.NewMetricRepository(stor)
			svc := service.NewMetricService(repo)
			handler := NewMetricHandler(svc)

			// Настраиваем метрики
			if tt.setupMetric != nil {
				tt.setupMetric(repo)
			}

			// Создаём роутер
			r := chi.NewRouter()
			r.Get("/value/{type}/{name}", handler.Get)

			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantBody != "" {
				assert.Contains(t, w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestHandler_List(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := repository.NewMetricRepository(stor)
	svc := service.NewMetricService(repo)
	handler := NewMetricHandler(svc)

	// Добавляем метрики
	repo.Save(&model.GaugeMetric{Name: "cpu", Value: 0.5})
	repo.Save(&model.GaugeMetric{Name: "memory", Value: 128.0})
	repo.Save(&model.CounterMetric{Name: "requests", Value: 10})

	// Создаём роутер
	r := chi.NewRouter()
	r.Get("/", handler.List)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html", w.Header().Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(t, body, "<html>")
	assert.Contains(t, body, "cpu")
	assert.Contains(t, body, "0.5")
	assert.Contains(t, body, "memory")
	assert.Contains(t, body, "128")
	assert.Contains(t, body, "requests")
	assert.Contains(t, body, "10")
}
