package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/api"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/out/memory"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/usecase"
	"github.com/go-chi/chi/v5"
)

func newExampleUseCase() *usecase.MetricUseCase {
	store := memory.NewMetricStorage()
	repo := memory.NewMetricRepo(store)
	return usecase.NewMetricUseCase(repo)
}

// ExampleMetricJSONHandler_Update демонстрирует обновление gauge-метрики через JSON API.
func ExampleMetricJSONHandler_Update() {
	uc := newExampleUseCase()
	h := NewMetricJSONHandler(uc, nil)
	r := chi.NewRouter()
	r.Post("/update", h.Update)

	body := `{"id":"cpu_usage","type":"gauge","value":45.7}`
	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp api.MetricDTO
	_ = json.NewDecoder(w.Body).Decode(&resp)
	fmt.Printf("Status: %d, ID: %s, Value: %.1f\n", w.Code, resp.ID, *resp.Value)
	// Output: Status: 200, ID: cpu_usage, Value: 45.7
}

// ExampleMetricJSONHandler_Get демонстрирует получение метрики через JSON API.
func ExampleMetricJSONHandler_Get() {
	uc := newExampleUseCase()
	_ = uc.Save(context.Background(), entity.NewGaugeMetric("temperature", 36.6))

	h := NewMetricJSONHandler(uc, nil)
	r := chi.NewRouter()
	r.Post("/value", h.Get)

	body := `{"id":"temperature","type":"gauge"}`
	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp api.MetricDTO
	_ = json.NewDecoder(w.Body).Decode(&resp)
	fmt.Printf("Status: %d, ID: %s, Value: %.1f\n", w.Code, resp.ID, *resp.Value)
	// Output: Status: 200, ID: temperature, Value: 36.6
}

// ExampleMetricJSONHandler_BatchUpdate демонстрирует пакетное обновление метрик.
func ExampleMetricJSONHandler_BatchUpdate() {
	uc := newExampleUseCase()
	h := NewMetricJSONHandler(uc, nil)
	r := chi.NewRouter()
	r.Post("/updates", h.BatchUpdate)

	body := `[
		{"id":"cpu","type":"gauge","value":55.5},
		{"id":"requests","type":"counter","delta":100}
	]`
	req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	// Output: Status: 200
}

// ExampleMetricHandler_Update демонстрирует обновление метрики через URL-параметры.
func ExampleMetricHandler_Update() {
	uc := newExampleUseCase()
	h := NewMetricHandler(uc, nil)
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.Update)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/99.9", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	fmt.Printf("Status: %d\n", w.Code)
	// Output: Status: 200
}
