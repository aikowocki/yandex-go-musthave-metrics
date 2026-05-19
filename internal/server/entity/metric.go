package entity

import (
	"errors"
	"time"
)

type MetricType string

const (
	MetricTypeCounter MetricType = "counter"
	MetricTypeGauge   MetricType = "gauge"
)

type Metric interface {
	GetType() MetricType
	GetName() string
}

type BaseMetric struct {
	Name      string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func (b *BaseMetric) GetName() string {
	return b.Name
}

type GaugeMetric struct {
	BaseMetric
	Value float64
}

type CounterMetric struct {
	BaseMetric
	Value int64
}

func (g *GaugeMetric) GetType() MetricType   { return MetricTypeGauge }
func (c *CounterMetric) GetType() MetricType { return MetricTypeCounter }

func (g *GaugeMetric) GetValue() float64 { return g.Value }
func (c *CounterMetric) GetValue() int64 { return c.Value }

var (
	ErrInvalidMetricType  = errors.New("invalid metric type")
	ErrInvalidMetricValue = errors.New("invalid metric value")
	ErrEmptyMetricName    = errors.New("empty metric name")
	ErrMetricNotFound     = errors.New("metric not found")
)

func NewGaugeMetric(name string, value float64) *GaugeMetric {
	return &GaugeMetric{
		BaseMetric: BaseMetric{Name: name},
		Value:      value,
	}
}

func NewCounterMetric(name string, value int64) *CounterMetric {
	return &CounterMetric{
		BaseMetric: BaseMetric{Name: name},
		Value:      value,
	}
}
