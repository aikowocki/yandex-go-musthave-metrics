package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
	"github.com/go-chi/chi/v5"
)

type MetricUseCase interface {
	Save(ctx context.Context, metric entity.Metric) error
	Get(ctx context.Context, metricType string, name string) (entity.Metric, error)
	GetAll(ctx context.Context) ([]entity.Metric, error)
	UpdateBatch(ctx context.Context, metrics []entity.Metric) error
}

type baseMetricHandler struct {
	uc    MetricUseCase
	audit port.AuditPublisher
}

func (h *baseMetricHandler) publishAudit(r *http.Request, names []string) {
	if h.audit == nil || len(names) == 0 {
		return
	}
	h.audit.Publish(entity.AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   names,
		IPAddress: clientIP(r),
	})
}

// MetricHandler обрабатывает HTTP-запросы для работы с метриками через URL-параметры.
// Используется для эндпоинтов вида /update/{type}/{name}/{value} и /value/{type}/{name}.
type MetricHandler struct {
	baseMetricHandler
}

// NewMetricHandler создаёт новый обработчик метрик с URL-параметрами.
func NewMetricHandler(uc MetricUseCase, audit port.AuditPublisher) *MetricHandler {
	return &MetricHandler{baseMetricHandler: baseMetricHandler{uc: uc, audit: audit}}
}

func (h *MetricHandler) handleError(err error, w http.ResponseWriter) {
	switch {
	case isMetricNotFound(err):
		http.Error(w, err.Error(), http.StatusNotFound)
	case isMetricInvalid(err):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func isMetricNotFound(err error) bool {
	return errors.Is(err, entity.ErrEmptyMetricName) || errors.Is(err, entity.ErrMetricNotFound)
}

func isMetricInvalid(err error) bool {
	return errors.Is(err, entity.ErrInvalidMetricType) || errors.Is(err, entity.ErrInvalidMetricValue)
}

func (h *MetricHandler) Update(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	rawValue := chi.URLParam(r, "value")

	if name == "" {
		http.Error(w, entity.ErrEmptyMetricName.Error(), http.StatusNotFound)
		return
	}

	var m entity.Metric
	switch entity.MetricType(mType) {
	case entity.MetricTypeGauge:
		v, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			http.Error(w, entity.ErrInvalidMetricValue.Error(), http.StatusBadRequest)
			return
		}
		m = entity.NewGaugeMetric(name, v)
	case entity.MetricTypeCounter:
		v, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			http.Error(w, entity.ErrInvalidMetricValue.Error(), http.StatusBadRequest)
			return
		}
		m = entity.NewCounterMetric(name, v)
	default:
		http.Error(w, entity.ErrInvalidMetricType.Error(), http.StatusBadRequest)
		return
	}

	if err := h.uc.Save(r.Context(), m); err != nil {
		h.handleError(err, w)
		return
	}

	h.publishAudit(r, []string{m.GetName()})
	w.WriteHeader(http.StatusOK)
}

func (h *MetricHandler) Get(w http.ResponseWriter, r *http.Request) {
	m, err := h.uc.Get(r.Context(), chi.URLParam(r, "type"), chi.URLParam(r, "name"))
	if err != nil {
		h.handleError(err, w)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	switch v := m.(type) {
	case *entity.GaugeMetric:
		_, _ = fmt.Fprint(w, strconv.FormatFloat(v.Value, 'f', -1, 64))
	case *entity.CounterMetric:
		_, _ = fmt.Fprint(w, strconv.FormatInt(v.Value, 10))
	}
}

func (h *MetricHandler) List(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.uc.GetAll(r.Context())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprint(w, "<html><body><h1>Metrics</h1><ul>")
	for _, m := range metrics {
		switch v := m.(type) {
		case *entity.GaugeMetric:
			_, _ = fmt.Fprintf(w, "<li>%s (gauge): %s</li>", v.GetName(), strconv.FormatFloat(v.Value, 'f', -1, 64))
		case *entity.CounterMetric:
			_, _ = fmt.Fprintf(w, "<li>%s (counter): %d</li>", v.GetName(), v.Value)
		}
	}
	_, _ = fmt.Fprint(w, "</ul></body></html>")
}
