package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/api"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
)

// MetricJSONHandler обрабатывает HTTP-запросы для работы с метриками через JSON body.
// Используется для эндпоинтов POST /update, POST /updates, POST /value.
type MetricJSONHandler struct {
	baseMetricHandler
}

// NewMetricJSONHandler создаёт новый обработчик метрик с JSON body.
func NewMetricJSONHandler(uc MetricUseCase, audit port.AuditPublisher) *MetricJSONHandler {
	return &MetricJSONHandler{baseMetricHandler: baseMetricHandler{uc: uc, audit: audit}}
}

func writeJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: message})
}

func toEntity(dto api.MetricDTO) (entity.Metric, error) {
	if dto.ID == "" {
		return nil, entity.ErrEmptyMetricName
	}
	switch entity.MetricType(dto.MType) {
	case entity.MetricTypeGauge:
		if dto.Value == nil {
			return nil, entity.ErrInvalidMetricValue
		}
		return entity.NewGaugeMetric(dto.ID, *dto.Value), nil
	case entity.MetricTypeCounter:
		if dto.Delta == nil {
			return nil, entity.ErrInvalidMetricValue
		}
		return entity.NewCounterMetric(dto.ID, *dto.Delta), nil
	default:
		return nil, entity.ErrInvalidMetricType
	}
}

func toResponse(m entity.Metric) api.MetricDTO {
	resp := api.MetricDTO{ID: m.GetName(), MType: string(m.GetType())}
	switch v := m.(type) {
	case *entity.GaugeMetric:
		resp.Value = &v.Value
	case *entity.CounterMetric:
		resp.Delta = &v.Value
	}
	return resp
}

func (h *MetricJSONHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto api.MetricDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		writeJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	m, err := toEntity(dto)
	if err != nil {
		h.handleError(err, w, "failed to update metric")
		return
	}

	if err = h.uc.Save(r.Context(), m); err != nil {
		h.handleError(err, w, "failed to update metric")
		return
	}

	h.publishAudit(r, []string{m.GetName()})

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
	var dto api.MetricDTO
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
	var dtos []api.MetricDTO

	if err := json.NewDecoder(r.Body).Decode(&dtos); err != nil {
		writeJSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	metrics := make([]entity.Metric, 0, len(dtos))
	for _, dto := range dtos {
		m, err := toEntity(dto)
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
		names := make([]string, len(metrics))
		for i, m := range metrics {
			names[i] = m.GetName()
		}
		h.publishAudit(r, names)
	}

	w.WriteHeader(http.StatusOK)
}
