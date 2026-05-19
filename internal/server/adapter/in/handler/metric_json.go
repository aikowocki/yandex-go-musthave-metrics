package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
)

type MetricJSONHandler struct {
	uc MetricUseCase
}

func NewMetricJSONHandler(uc MetricUseCase) *MetricJSONHandler {
	return &MetricJSONHandler{uc: uc}
}

type metricDTO struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

func writeJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: message})
}

func (req *metricDTO) toEntity() (entity.Metric, error) {
	if req.ID == "" {
		return nil, entity.ErrEmptyMetricName
	}
	switch entity.MetricType(req.MType) {
	case entity.MetricTypeGauge:
		if req.Value == nil {
			return nil, entity.ErrInvalidMetricValue
		}
		return entity.NewGaugeMetric(req.ID, *req.Value), nil
	case entity.MetricTypeCounter:
		if req.Delta == nil {
			return nil, entity.ErrInvalidMetricValue
		}
		return entity.NewCounterMetric(req.ID, *req.Delta), nil
	default:
		return nil, entity.ErrInvalidMetricType
	}
}

func toResponse(m entity.Metric) metricDTO {
	resp := metricDTO{ID: m.GetName(), MType: string(m.GetType())}
	switch v := m.(type) {
	case *entity.GaugeMetric:
		resp.Value = &v.Value
	case *entity.CounterMetric:
		resp.Delta = &v.Value
	}
	return resp
}

func (h *MetricJSONHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto metricDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	m, err := dto.toEntity()
	if err != nil {
		h.handleError(err, w, "failed to update metric")
		return
	}

	if err = h.uc.Save(r.Context(), m); err != nil {
		h.handleError(err, w, "failed to update metric")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(m))
}

func (h *MetricJSONHandler) handleError(err error, w http.ResponseWriter, defaultMsg string) {
	switch {
	case errors.Is(err, entity.ErrEmptyMetricName),
		errors.Is(err, entity.ErrMetricNotFound):
		writeJSONError(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, entity.ErrInvalidMetricType),
		errors.Is(err, entity.ErrInvalidMetricValue):
		writeJSONError(w, err.Error(), http.StatusBadRequest)
	default:
		writeJSONError(w, defaultMsg, http.StatusInternalServerError)
	}
}

func (h *MetricJSONHandler) Get(w http.ResponseWriter, r *http.Request) {
	var dto metricDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	m, err := h.uc.Get(r.Context(), dto.MType, dto.ID)
	if err != nil {
		h.handleError(err, w, "failed to get metric")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(m))

}

func (h *MetricJSONHandler) BatchUpdate(w http.ResponseWriter, r *http.Request) {
	var dtos []metricDTO

	if err := json.NewDecoder(r.Body).Decode(&dtos); err != nil {
		writeJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	metrics := make([]entity.Metric, 0, len(dtos))
	for _, dto := range dtos {
		m, err := dto.toEntity()
		if err != nil {
			h.handleError(err, w, "failed to batch update metrics")
			return
		}
		metrics = append(metrics, m)
	}

	if len(metrics) > 0 {
		if err := h.uc.UpdateBatch(r.Context(), metrics); err != nil {
			h.handleError(err, w, "failed to batch update metric")
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
