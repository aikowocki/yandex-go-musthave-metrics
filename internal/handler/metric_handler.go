package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/repository"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/storage/metric"
	"github.com/go-chi/chi/v5"
)

type MetricHandler struct {
	repo *repository.MetricRepository
}

func NewMetricHandler(repo *repository.MetricRepository) *MetricHandler {
	return &MetricHandler{repo: repo}
}

func (h *MetricHandler) Update(w http.ResponseWriter, r *http.Request) {
	m, err := model.NewMetric(
		chi.URLParam(r, "type"),
		chi.URLParam(r, "name"),
		chi.URLParam(r, "value"),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Save(m); err != nil {
		http.Error(w, "failed to save metric", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *MetricHandler) Get(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	m, err := h.repo.Get(metricType, name)
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
	metrics, err := h.repo.GetAll()
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
