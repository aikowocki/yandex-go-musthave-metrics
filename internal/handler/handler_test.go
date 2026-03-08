package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
			stor := metric.NewMetricMemoryStorage()
			repo := repository.NewMetricRepository(stor)
			svc := service.NewMetricService(repo)
			handler := NewMetricHandler(svc)

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
				ctx := context.Background()
				repo.Save(ctx, model.NewGaugeMetric("cpu", 0.5))
			},
			url:            "/value/gauge/cpu",
			wantStatusCode: http.StatusOK,
			wantBody:       "0.5",
		},
		{
			name: "get counter success",
			setupMetric: func(repo *repository.MetricRepository) {
				ctx := context.Background()
				repo.Save(ctx, model.NewCounterMetric("requests", 10))
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

			if tt.setupMetric != nil {
				tt.setupMetric(repo)
			}

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
	ctx := context.Background()

	// Добавляем метрики
	repo.Save(ctx, model.NewGaugeMetric("cpu", 0.5))
	repo.Save(ctx, model.NewGaugeMetric("memory", 128.0))
	repo.Save(ctx, model.NewCounterMetric("requests", 10))

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

func TestHandler_UpdateJSON(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantStatusCode int
		wantJSON       bool
	}{
		{
			name:           "success gauge",
			body:           `{"id":"cpu","type":"gauge","value":3.14}`,
			wantStatusCode: http.StatusOK,
			wantJSON:       true,
		},
		{
			name:           "success counter",
			body:           `{"id":"requests","type":"counter","delta":5}`,
			wantStatusCode: http.StatusOK,
			wantJSON:       true,
		},
		{
			name:           "invalid json",
			body:           `{broken`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "empty name",
			body:           `{"id":"","type":"gauge","value":1.0}`,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "unknown type",
			body:           `{"id":"test","type":"unknown","value":1.0}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "gauge without value",
			body:           `{"id":"test","type":"gauge"}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "counter without delta",
			body:           `{"id":"test","type":"counter"}`,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stor := metric.NewMetricMemoryStorage()
			repo := repository.NewMetricRepository(stor)
			svc := service.NewMetricService(repo)
			h := NewMetricHandler(svc)

			r := chi.NewRouter()
			r.Post("/update", h.UpdateJSON)

			req := httptest.NewRequest("POST", "/update", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantJSON {
				assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
			}
		})
	}
}

func TestHandler_UpdateJSON_ResponseBody(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := repository.NewMetricRepository(stor)
	svc := service.NewMetricService(repo)
	h := NewMetricHandler(svc)

	r := chi.NewRouter()
	r.Post("/update", h.UpdateJSON)

	// Отправляем gauge
	req := httptest.NewRequest("POST", "/update", strings.NewReader(`{"id":"cpu","type":"gauge","value":3.14}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.MetricDTO
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "cpu", resp.ID)
	assert.Equal(t, "gauge", resp.MType)
	assert.NotNil(t, resp.Value)
	assert.Equal(t, 3.14, *resp.Value)
}

func TestHandler_UpdateJSON_CounterSum(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := repository.NewMetricRepository(stor)
	svc := service.NewMetricService(repo)
	h := NewMetricHandler(svc)

	r := chi.NewRouter()
	r.Post("/update", h.UpdateJSON)

	// Первый запрос — delta 10
	req := httptest.NewRequest("POST", "/update", strings.NewReader(`{"id":"hits","type":"counter","delta":10}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Второй запрос — delta 5
	req = httptest.NewRequest("POST", "/update", strings.NewReader(`{"id":"hits","type":"counter","delta":5}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем что в ответе сумма 15
	var resp model.MetricDTO
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "hits", resp.ID)
	assert.NotNil(t, resp.Delta)
	assert.Equal(t, int64(15), *resp.Delta)
}

func TestHandler_GetJSON(t *testing.T) {
	tests := []struct {
		name           string
		setupMetric    func(*repository.MetricRepository)
		body           string
		wantStatusCode int
		wantJSON       bool
	}{
		{
			name: "get gauge success",
			setupMetric: func(repo *repository.MetricRepository) {
				ctx := context.Background()
				repo.Save(ctx, model.NewGaugeMetric("cpu", 0.5))
			},
			body:           `{"id":"cpu","type":"gauge"}`,
			wantStatusCode: http.StatusOK,
			wantJSON:       true,
		},
		{
			name: "get counter success",
			setupMetric: func(repo *repository.MetricRepository) {
				ctx := context.Background()
				repo.Save(ctx, model.NewCounterMetric("requests", 10))
			},
			body:           `{"id":"requests","type":"counter"}`,
			wantStatusCode: http.StatusOK,
			wantJSON:       true,
		},
		{
			name:           "not found",
			setupMetric:    func(repo *repository.MetricRepository) {},
			body:           `{"id":"missing","type":"gauge"}`,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "invalid json",
			setupMetric:    func(repo *repository.MetricRepository) {},
			body:           `{broken`,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stor := metric.NewMetricMemoryStorage()
			repo := repository.NewMetricRepository(stor)
			svc := service.NewMetricService(repo)
			h := NewMetricHandler(svc)

			tt.setupMetric(repo)

			r := chi.NewRouter()
			r.Post("/value", h.GetJSON)

			req := httptest.NewRequest("POST", "/value", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantJSON {
				assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
			}
		})
	}
}

func TestHandler_GetJSON_ResponseBody(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := repository.NewMetricRepository(stor)
	svc := service.NewMetricService(repo)
	h := NewMetricHandler(svc)
	ctx := context.Background()

	repo.Save(ctx, model.NewGaugeMetric("cpu", 42.5))

	r := chi.NewRouter()
	r.Post("/value", h.GetJSON)

	req := httptest.NewRequest("POST", "/value", strings.NewReader(`{"id":"cpu","type":"gauge"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.MetricDTO
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "cpu", resp.ID)
	assert.Equal(t, "gauge", resp.MType)
	assert.NotNil(t, resp.Value)
	assert.Equal(t, 42.5, *resp.Value)
}
