package handler

import (
	"strconv"
	"strings"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/httperror"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
)

type Metric interface {
	Type() model.MetricType
}

type GaugeMetric struct {
	Name  string
	Value float64
}

type CounterMetric struct {
	Name  string
	Value int64
}

func (g *GaugeMetric) Type() model.MetricType   { return model.Gauge }
func (c *CounterMetric) Type() model.MetricType { return model.Counter }

type MetricRequest struct {
	Type  string
	Name  string
	Value string
}

func ParseAndValidate(path string) (Metric, *httperror.HTTPError) {
	metric, err := parsePath(path).validate()
	if err != nil {
		return nil, err
	}
	return metric, nil
}

func parsePath(path string) *MetricRequest {
	parts := strings.Split(path, "/")

	return &MetricRequest{
		Type:  getOrEmpty(parts, 2),
		Name:  getOrEmpty(parts, 3),
		Value: getOrEmpty(parts, 4),
	}
}

func getOrEmpty(parts []string, i int) string {
	if i >= len(parts) {
		return ""
	}
	return parts[i]
}

func (r *MetricRequest) validate() (Metric, *httperror.HTTPError) {
	if r.Name == "" {
		return nil, httperror.NotFound("metric name required")
	}
	var metric Metric

	switch model.MetricType(r.Type) {
	case model.Gauge:
		value, err := strconv.ParseFloat(r.Value, 64)
		if err != nil {
			return nil, httperror.BadRequest("invalid gauge value")
		}
		metric = &GaugeMetric{
			Name:  r.Name,
			Value: value,
		}
	case model.Counter:
		value, err := strconv.ParseInt(r.Value, 10, 64)
		if err != nil {
			return nil, httperror.BadRequest("invalid counter value")
		}
		metric = &CounterMetric{
			Name:  r.Name,
			Value: value,
		}
	default:
		return nil, httperror.BadRequest("invalid metric type")
	}

	return metric, nil
}
