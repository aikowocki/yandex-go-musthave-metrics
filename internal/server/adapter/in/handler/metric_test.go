package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/api"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/out/memory"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestUseCase собирает чистую цепочку memory storage → repo → use case
// для использования в тестах хендлеров.
func newTestUseCase() *usecase.MetricUseCase {
	store := memory.NewMetricStorage()
	repo := memory.NewMetricRepo(store)
	return usecase.NewMetricUseCase(repo)
}

// ---------- MetricHandler (path-based) ----------

func TestMetricHandler_Update(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		wantStatusCode int
	}{
		{name: "success gauge", url: "/update/gauge/test/3.14", wantStatusCode: http.StatusOK},
		{name: "success counter", url: "/update/counter/test/5", wantStatusCode: http.StatusOK},
		{name: "invalid metric type", url: "/update/unknown/test/1", wantStatusCode: http.StatusBadRequest},
		{name: "invalid gauge value", url: "/update/gauge/test/abc", wantStatusCode: http.StatusBadRequest},
		{name: "invalid counter value", url: "/update/counter/test/3.14", wantStatusCode: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewMetricHandler(newTestUseCase())
			r := chi.NewRouter()
			r.Post("/update/{type}/{name}/{value}", h.Update)

			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

func TestMetricHandler_Get(t *testing.T) {
	tests := []struct {
		name           string
		seed           func(context.Context, *usecase.MetricUseCase) error
		url            string
		wantStatusCode int
		wantBody       string
	}{
		{
			name: "get gauge success",
			seed: func(ctx context.Context, uc *usecase.MetricUseCase) error {
				return uc.Save(ctx, entity.NewGaugeMetric("cpu", 0.5))
			},
			url:            "/value/gauge/cpu",
			wantStatusCode: http.StatusOK,
			wantBody:       "0.5",
		},
		{
			name: "get counter success",
			seed: func(ctx context.Context, uc *usecase.MetricUseCase) error {
				return uc.Save(ctx, entity.NewCounterMetric("requests", 10))
			},
			url:            "/value/counter/requests",
			wantStatusCode: http.StatusOK,
			wantBody:       "10",
		},
		{
			name:           "get not found",
			seed:           func(ctx context.Context, uc *usecase.MetricUseCase) error { return nil },
			url:            "/value/gauge/nonexistent",
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newTestUseCase()
			require.NoError(t, tt.seed(context.Background(), uc))

			h := NewMetricHandler(uc)
			r := chi.NewRouter()
			r.Get("/value/{type}/{name}", h.Get)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			if tt.wantBody != "" {
				assert.Contains(t, w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestMetricHandler_List(t *testing.T) {
	uc := newTestUseCase()
	ctx := context.Background()

	require.NoError(t, uc.Save(ctx, entity.NewGaugeMetric("cpu", 0.5)))
	require.NoError(t, uc.Save(ctx, entity.NewGaugeMetric("memory", 128.0)))
	require.NoError(t, uc.Save(ctx, entity.NewCounterMetric("requests", 10)))

	h := NewMetricHandler(uc)
	r := chi.NewRouter()
	r.Get("/", h.List)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
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

// ---------- MetricJSONHandler ----------

func TestMetricJSONHandler_Update(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantStatusCode int
		wantJSON       bool
	}{
		{name: "success gauge", body: `{"id":"cpu","type":"gauge","value":3.14}`, wantStatusCode: http.StatusOK, wantJSON: true},
		{name: "success counter", body: `{"id":"hits","type":"counter","delta":5}`, wantStatusCode: http.StatusOK, wantJSON: true},
		{name: "invalid json", body: `{broken`, wantStatusCode: http.StatusBadRequest},
		{name: "empty name", body: `{"id":"","type":"gauge","value":1.0}`, wantStatusCode: http.StatusNotFound},
		{name: "unknown type", body: `{"id":"test","type":"unknown","value":1.0}`, wantStatusCode: http.StatusBadRequest},
		{name: "gauge without value", body: `{"id":"test","type":"gauge"}`, wantStatusCode: http.StatusBadRequest},
		{name: "counter without delta", body: `{"id":"test","type":"counter"}`, wantStatusCode: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewMetricJSONHandler(newTestUseCase())
			r := chi.NewRouter()
			r.Post("/update", h.Update)

			req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(tt.body))
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

func TestMetricJSONHandler_Update_ResponseBody(t *testing.T) {
	h := NewMetricJSONHandler(newTestUseCase())
	r := chi.NewRouter()
	r.Post("/update", h.Update)

	req := httptest.NewRequest(http.MethodPost, "/update",
		strings.NewReader(`{"id":"cpu","type":"gauge","value":3.14}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp api.MetricDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "cpu", resp.ID)
	assert.Equal(t, api.MetricTypeGauge, resp.MType)
	require.NotNil(t, resp.Value)
	assert.Equal(t, 3.14, *resp.Value)
}

func TestMetricJSONHandler_Update_CounterAccumulates(t *testing.T) {
	h := NewMetricJSONHandler(newTestUseCase())
	r := chi.NewRouter()
	r.Post("/update", h.Update)

	send := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	require.Equal(t, http.StatusOK, send(`{"id":"hits","type":"counter","delta":10}`).Code)
	w := send(`{"id":"hits","type":"counter","delta":5}`)
	require.Equal(t, http.StatusOK, w.Code)

	var resp api.MetricDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "hits", resp.ID)
	require.NotNil(t, resp.Delta)
	assert.Equal(t, int64(15), *resp.Delta)
}

func TestMetricJSONHandler_Get(t *testing.T) {
	tests := []struct {
		name           string
		seed           func(context.Context, *usecase.MetricUseCase) error
		body           string
		wantStatusCode int
		wantJSON       bool
	}{
		{
			name: "get gauge success",
			seed: func(ctx context.Context, uc *usecase.MetricUseCase) error {
				return uc.Save(ctx, entity.NewGaugeMetric("cpu", 0.5))
			},
			body:           `{"id":"cpu","type":"gauge"}`,
			wantStatusCode: http.StatusOK,
			wantJSON:       true,
		},
		{
			name: "get counter success",
			seed: func(ctx context.Context, uc *usecase.MetricUseCase) error {
				return uc.Save(ctx, entity.NewCounterMetric("requests", 10))
			},
			body:           `{"id":"requests","type":"counter"}`,
			wantStatusCode: http.StatusOK,
			wantJSON:       true,
		},
		{
			name:           "not found",
			seed:           func(ctx context.Context, uc *usecase.MetricUseCase) error { return nil },
			body:           `{"id":"missing","type":"gauge"}`,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "invalid json",
			seed:           func(ctx context.Context, uc *usecase.MetricUseCase) error { return nil },
			body:           `{broken`,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newTestUseCase()
			require.NoError(t, tt.seed(context.Background(), uc))

			h := NewMetricJSONHandler(uc)
			r := chi.NewRouter()
			r.Post("/value", h.Get)

			req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(tt.body))
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

func TestMetricJSONHandler_Get_ResponseBody(t *testing.T) {
	uc := newTestUseCase()
	require.NoError(t, uc.Save(context.Background(), entity.NewGaugeMetric("cpu", 42.5)))

	h := NewMetricJSONHandler(uc)
	r := chi.NewRouter()
	r.Post("/value", h.Get)

	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(`{"id":"cpu","type":"gauge"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp api.MetricDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "cpu", resp.ID)
	assert.Equal(t, api.MetricTypeGauge, resp.MType)
	require.NotNil(t, resp.Value)
	assert.Equal(t, 42.5, *resp.Value)
}

func TestMetricJSONHandler_BatchUpdate(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantStatusCode int
	}{
		{
			name:           "success mixed batch",
			body:           `[{"id":"cpu","type":"gauge","value":3.14},{"id":"hits","type":"counter","delta":5}]`,
			wantStatusCode: http.StatusOK,
		},
		{name: "empty batch", body: `[]`, wantStatusCode: http.StatusOK},
		{name: "invalid json", body: `{broken`, wantStatusCode: http.StatusBadRequest},
		{name: "unknown metric type", body: `[{"id":"test","type":"unknown","value":1.0}]`, wantStatusCode: http.StatusBadRequest},
		{name: "gauge without value", body: `[{"id":"cpu","type":"gauge"}]`, wantStatusCode: http.StatusBadRequest},
		{name: "counter without delta", body: `[{"id":"hits","type":"counter"}]`, wantStatusCode: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewMetricJSONHandler(newTestUseCase())
			r := chi.NewRouter()
			r.Post("/updates", h.BatchUpdate)

			req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)
		})
	}
}

func TestMetricJSONHandler_BatchUpdate_CounterAccumulates(t *testing.T) {
	uc := newTestUseCase()
	h := NewMetricJSONHandler(uc)
	r := chi.NewRouter()
	r.Post("/updates", h.BatchUpdate)

	send := func(body string) int {
		req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusOK, send(`[{"id":"hits","type":"counter","delta":10}]`))
	assert.Equal(t, http.StatusOK, send(`[{"id":"hits","type":"counter","delta":5}]`))

	got, err := uc.Get(context.Background(), string(entity.MetricTypeCounter), "hits")
	require.NoError(t, err)
	c, ok := got.(*entity.CounterMetric)
	require.True(t, ok)
	assert.Equal(t, int64(15), c.Value)
}
