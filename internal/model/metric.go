package model

import (
	"strconv"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/httperror"
)

type MetricType string

const (
	MetricTypeCounter MetricType = "counter"
	MetricTypeGauge   MetricType = "gauge"
)

type Metric interface {
	getType() MetricType
}

type GaugeMetric struct {
	Name  string
	Value float64
}

type CounterMetric struct {
	Name  string
	Value int64
}

func (g *GaugeMetric) getType() MetricType   { return MetricTypeGauge }
func (c *CounterMetric) getType() MetricType { return MetricTypeCounter }

func NewMetric(Type string, Name string, Value string) (Metric, error) {
	if Name == "" {
		return nil, httperror.NotFound("metric name required")
	}
	switch MetricType(Type) {
	case MetricTypeGauge:
		value, err := strconv.ParseFloat(Value, 64)
		if err != nil {
			return nil, httperror.BadRequest("invalid gauge value")
		}
		return &GaugeMetric{
			Name:  Name,
			Value: value,
		}, nil
	case MetricTypeCounter:
		value, err := strconv.ParseInt(Value, 10, 64)
		if err != nil {
			return nil, httperror.BadRequest("invalid counter value")
		}
		return &CounterMetric{
			Name:  Name,
			Value: value,
		}, nil
	default:
		return nil, httperror.BadRequest("invalid metric type")
	}

}
