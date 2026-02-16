package model

import (
	"errors"
	"strconv"
)

var (
	ErrInvalidType  = errors.New("invalid metric type")
	ErrInvalidValue = errors.New("invalid metric value")
	ErrEmptyName    = errors.New("empty metric name")
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
		return nil, ErrEmptyName
	}
	switch MetricType(Type) {
	case MetricTypeGauge:
		value, err := strconv.ParseFloat(Value, 64)
		if err != nil {
			return nil, ErrInvalidValue
		}
		return &GaugeMetric{
			Name:  Name,
			Value: value,
		}, nil
	case MetricTypeCounter:
		value, err := strconv.ParseInt(Value, 10, 64)
		if err != nil {
			return nil, ErrInvalidValue
		}
		return &CounterMetric{
			Name:  Name,
			Value: value,
		}, nil
	default:
		return nil, ErrInvalidType
	}

}
