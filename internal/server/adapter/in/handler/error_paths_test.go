package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// fakeUseCase — управляемый мок MetricUseCase: каждая операция возвращает
// заданную ошибку, что позволяет проверить маппинг ошибок на HTTP-коды.
type fakeUseCase struct {
	saveErr   error
	getMetric entity.Metric
	getErr    error
	getAllErr error
	batchErr  error
}

func (f fakeUseCase) Save(context.Context, entity.Metric) error { return f.saveErr }
func (f fakeUseCase) Get(context.Context, string, string) (entity.Metric, error) {
	return f.getMetric, f.getErr
}
func (f fakeUseCase) GetAll(context.Context) ([]entity.Metric, error)    { return nil, f.getAllErr }
func (f fakeUseCase) UpdateBatch(context.Context, []entity.Metric) error { return f.batchErr }

// ---------- MetricHandler ----------

func TestMetricHandler_Update_UseCaseErrors(t *testing.T) {
	tests := []struct {
		name       string
		saveErr    error
		wantStatus int
	}{
		{name: "internal error → 500", saveErr: errors.New("db down"), wantStatus: http.StatusInternalServerError},
		{name: "not found → 404", saveErr: entity.ErrMetricNotFound, wantStatus: http.StatusNotFound},
		{name: "invalid value → 400", saveErr: entity.ErrInvalidMetricValue, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewMetricHandler(fakeUseCase{saveErr: tt.saveErr}, nil)
			r := chi.NewRouter()
			r.Post("/update/{type}/{name}/{value}", h.Update)

			req := httptest.NewRequest(http.MethodPost, "/update/gauge/cpu/1.5", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestMetricHandler_Get_UseCaseErrors(t *testing.T) {
	tests := []struct {
		name       string
		getErr     error
		wantStatus int
	}{
		{name: "not found → 404", getErr: entity.ErrMetricNotFound, wantStatus: http.StatusNotFound},
		{name: "invalid type → 400", getErr: entity.ErrInvalidMetricType, wantStatus: http.StatusBadRequest},
		{name: "internal → 500", getErr: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewMetricHandler(fakeUseCase{getErr: tt.getErr}, nil)
			r := chi.NewRouter()
			r.Get("/value/{type}/{name}", h.Get)

			req := httptest.NewRequest(http.MethodGet, "/value/gauge/cpu", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestMetricHandler_List_Error(t *testing.T) {
	h := NewMetricHandler(fakeUseCase{getAllErr: errors.New("storage error")}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.List(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- MetricJSONHandler ----------

func TestMetricJSONHandler_Update_Errors(t *testing.T) {
	t.Run("invalid JSON → 400", func(t *testing.T) {
		h := NewMetricJSONHandler(fakeUseCase{}, nil)
		req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("{bad json"))
		w := httptest.NewRecorder()
		h.Update(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid metric type → 400", func(t *testing.T) {
		h := NewMetricJSONHandler(fakeUseCase{}, nil)
		req := httptest.NewRequest(http.MethodPost, "/update",
			strings.NewReader(`{"id":"x","type":"weird"}`))
		w := httptest.NewRecorder()
		h.Update(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("usecase save error → 500", func(t *testing.T) {
		h := NewMetricJSONHandler(fakeUseCase{saveErr: errors.New("db down")}, nil)
		req := httptest.NewRequest(http.MethodPost, "/update",
			strings.NewReader(`{"id":"cpu","type":"gauge","value":1.5}`))
		w := httptest.NewRecorder()
		h.Update(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestMetricJSONHandler_Get_Errors(t *testing.T) {
	t.Run("invalid JSON → 400", func(t *testing.T) {
		h := NewMetricJSONHandler(fakeUseCase{}, nil)
		req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader("not json"))
		w := httptest.NewRecorder()
		h.Get(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found → 404", func(t *testing.T) {
		h := NewMetricJSONHandler(fakeUseCase{getErr: entity.ErrMetricNotFound}, nil)
		req := httptest.NewRequest(http.MethodPost, "/value",
			strings.NewReader(`{"id":"cpu","type":"gauge"}`))
		w := httptest.NewRecorder()
		h.Get(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestMetricJSONHandler_BatchUpdate_Errors(t *testing.T) {
	t.Run("invalid JSON → 400", func(t *testing.T) {
		h := NewMetricJSONHandler(fakeUseCase{}, nil)
		req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader("["))
		w := httptest.NewRecorder()
		h.BatchUpdate(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid metric in batch → 400", func(t *testing.T) {
		h := NewMetricJSONHandler(fakeUseCase{}, nil)
		req := httptest.NewRequest(http.MethodPost, "/updates",
			strings.NewReader(`[{"id":"x","type":"weird"}]`))
		w := httptest.NewRecorder()
		h.BatchUpdate(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("usecase batch error → 500", func(t *testing.T) {
		h := NewMetricJSONHandler(fakeUseCase{batchErr: errors.New("db down")}, nil)
		req := httptest.NewRequest(http.MethodPost, "/updates",
			strings.NewReader(`[{"id":"cpu","type":"gauge","value":1.5}]`))
		w := httptest.NewRecorder()
		h.BatchUpdate(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("empty batch → 200", func(t *testing.T) {
		h := NewMetricJSONHandler(fakeUseCase{}, nil)
		req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(`[]`))
		w := httptest.NewRecorder()
		h.BatchUpdate(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
