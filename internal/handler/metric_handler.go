package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/service"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/storage/metric"
	"github.com/go-chi/chi/v5"
)

type MetricHandler struct {
	service service.MetricService
}

func NewMetricHandler(service service.MetricService) *MetricHandler {
	return &MetricHandler{service: service}
}

func (h *MetricHandler) Update(w http.ResponseWriter, r *http.Request) {
	m, err := model.NewMetric(
		chi.URLParam(r, "type"),
		chi.URLParam(r, "name"),
		chi.URLParam(r, "value"))

	if err != nil {
		h.handleUpdateError(err, w)
		return
	}

	ctx := r.Context()
	if _, err := h.service.Update(ctx, m); err != nil {
		h.handleUpdateError(err, w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *MetricHandler) UpdateJSON(w http.ResponseWriter, r *http.Request) {
	var mr model.MetricDTO
	if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	m, err := mr.ToMetric()

	if err != nil {
		h.handleUpdateError(err, w)
		return
	}

	ctx := r.Context()
	if m, err = h.service.Update(ctx, m); err != nil {
		h.handleUpdateError(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.ToDTO())
}

func (h *MetricHandler) handleUpdateError(err error, w http.ResponseWriter) {
	switch {
	case errors.Is(err, model.ErrEmptyName):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, model.ErrInvalidType),
		errors.Is(err, model.ErrInvalidValue):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "failed to update metric", http.StatusInternalServerError)
	}
}

func (h *MetricHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	m, err := h.service.Get(ctx, chi.URLParam(r, "type"), chi.URLParam(r, "name"))
	if err != nil {
		if errors.Is(err, metric.ErrNotFound) {
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get metric", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	switch m := m.(type) {
	case *model.GaugeMetric:
		fmt.Fprintf(w, "%s", strconv.FormatFloat(m.GetValue(), 'f', -1, 64))
	case *model.CounterMetric:
		fmt.Fprintf(w, "%d", m.GetValue())
	}
}

func (h *MetricHandler) GetJSON(w http.ResponseWriter, r *http.Request) {
	var mr model.MetricDTO
	if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	m, err := h.service.Get(ctx, mr.MType, mr.ID)
	if err != nil {
		h.handleGetError(err, w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.ToDTO())

}

func (h *MetricHandler) handleGetError(err error, w http.ResponseWriter) {
	switch {
	case errors.Is(err, metric.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, "failed to get metric", http.StatusInternalServerError)
	}
}

func (h *MetricHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metrics, err := h.service.GetAll(ctx)
	if err != nil {
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><body><h1>Metrics</h1><ul>")
	for _, m := range metrics {
		switch m := m.(type) {
		case *model.GaugeMetric:
			fmt.Fprintf(w, "<li>%s (gauge): %s</li>", m.GetName(), strconv.FormatFloat(m.GetValue(), 'f', -1, 64))
		case *model.CounterMetric:
			fmt.Fprintf(w, "<li>%s (counter): %d</li>", m.GetName(), m.GetValue())
		}
	}
	fmt.Fprintf(w, "</ul></body></html>")
}
