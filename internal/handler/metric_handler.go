package handler

import (
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

	if err := h.service.Update(
		chi.URLParam(r, "type"),
		chi.URLParam(r, "name"),
		chi.URLParam(r, "value"),
	); err != nil {
		switch {
		case errors.Is(err, model.ErrEmptyName):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, model.ErrInvalidType),
			errors.Is(err, model.ErrInvalidValue):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "failed to update metric", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *MetricHandler) Get(w http.ResponseWriter, r *http.Request) {
	m, err := h.service.Get(chi.URLParam(r, "type"), chi.URLParam(r, "name"))
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
		fmt.Fprintf(w, "%s", strconv.FormatFloat(m.Value, 'f', -1, 64))
	case *model.CounterMetric:
		fmt.Fprintf(w, "%d", m.Value)
	}
}

func (h *MetricHandler) List(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.service.GetAll()
	if err != nil {
		http.Error(w, "failed to get metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><body><h1>Metrics</h1><ul>")
	for _, m := range metrics {
		switch m := m.(type) {
		case *model.GaugeMetric:
			fmt.Fprintf(w, "<li>%s (gauge): %s</li>", m.Name, strconv.FormatFloat(m.Value, 'f', -1, 64))
		case *model.CounterMetric:
			fmt.Fprintf(w, "<li>%s (counter): %d</li>", m.Name, m.Value)
		}
	}
	fmt.Fprintf(w, "</ul></body></html>")
}
