package handler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/go-chi/chi/v5"
)

func BenchmarkMetricJSONHandler_Update(b *testing.B) {
	h := NewMetricJSONHandler(newTestUseCase(), nil)

	body := []byte(`{"id":"cpu","type":"gauge","value":3.14}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.Update(w, req)
	}
}

func BenchmarkMetricJSONHandler_Get(b *testing.B) {
	uc := newTestUseCase()
	_ = uc.Save(context.Background(), entity.NewGaugeMetric("cpu", 42.5))
	h := NewMetricJSONHandler(uc, nil)

	body := []byte(`{"id":"cpu","type":"gauge"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.Get(w, req)
	}
}

func BenchmarkMetricJSONHandler_BatchUpdate(b *testing.B) {
	h := NewMetricJSONHandler(newTestUseCase(), nil)

	body := []byte(`[{"id":"cpu","type":"gauge","value":3.14},{"id":"mem","type":"gauge","value":128.5},{"id":"hits","type":"counter","delta":5}]`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.BatchUpdate(w, req)
	}
}

func BenchmarkMetricHandler_Update(b *testing.B) {
	h := NewMetricHandler(newTestUseCase(), nil)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.Update)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/3.14", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkMetricHandler_List(b *testing.B) {
	uc := newTestUseCase()
	ctx := context.Background()
	// предзаполняем
	for i := 0; i < 30; i++ {
		_ = uc.Save(ctx, entity.NewGaugeMetric(fmt.Sprintf("gauge_%d", i), float64(i)))
	}
	for i := 0; i < 10; i++ {
		_ = uc.Save(ctx, entity.NewCounterMetric(fmt.Sprintf("counter_%d", i), int64(i)))
	}

	h := NewMetricHandler(uc, nil)
	r := chi.NewRouter()
	r.Get("/", h.List)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}
