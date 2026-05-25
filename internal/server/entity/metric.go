// Package entity содержит доменные модели метрик и связанные с ними ошибки.
package entity

import (
	"errors"
	"time"
)

// MetricType определяет тип метрики (gauge или counter).
type MetricType string

const (
	// MetricTypeCounter — счётчик, значение которого накапливается.
	MetricTypeCounter MetricType = "counter"
	// MetricTypeGauge — показатель, значение которого перезаписывается.
	MetricTypeGauge MetricType = "gauge"
)

// Metric — общий интерфейс для всех типов метрик.
type Metric interface {
	GetType() MetricType
	GetName() string
}

// BaseMetric содержит общие поля для всех метрик: имя и временные метки.
type BaseMetric struct {
	Name      string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

// GetName возвращает имя метрики.
func (b *BaseMetric) GetName() string {
	return b.Name
}

// GaugeMetric — метрика типа gauge. Хранит текущее значение как float64.
type GaugeMetric struct {
	BaseMetric
	Value float64
}

// CounterMetric — метрика типа counter. Хранит накопленное значение как int64.
type CounterMetric struct {
	BaseMetric
	Value int64
}

// GetType возвращает тип метрики.
func (g *GaugeMetric) GetType() MetricType { return MetricTypeGauge }

// GetType возвращает тип метрики.
func (c *CounterMetric) GetType() MetricType { return MetricTypeCounter }

// GetValue возвращает текущее значение gauge-метрики.
func (g *GaugeMetric) GetValue() float64 { return g.Value }

// GetValue возвращает накопленное значение counter-метрики.
func (c *CounterMetric) GetValue() int64 { return c.Value }

// Ошибки валидации и поиска метрик.
var (
	ErrInvalidMetricType  = errors.New("invalid metric type")
	ErrInvalidMetricValue = errors.New("invalid metric value")
	ErrEmptyMetricName    = errors.New("empty metric name")
	ErrMetricNotFound     = errors.New("metric not found")
)

// NewGaugeMetric создаёт новую gauge-метрику с указанным именем и значением.
func NewGaugeMetric(name string, value float64) *GaugeMetric {
	return &GaugeMetric{
		BaseMetric: BaseMetric{Name: name},
		Value:      value,
	}
}

// NewCounterMetric создаёт новую counter-метрику с указанным именем и значением.
func NewCounterMetric(name string, value int64) *CounterMetric {
	return &CounterMetric{
		BaseMetric: BaseMetric{Name: name},
		Value:      value,
	}
}
