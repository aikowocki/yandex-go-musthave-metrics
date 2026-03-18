package model

import (
	"encoding/json"
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
	GetType() MetricType
	GetName() string
	ToDTO() MetricDTO
}

type MetricList []Metric

func (metrics *MetricList) UnmarshalJSON(data []byte) error {
	var dtos []MetricDTO
	if err := json.Unmarshal(data, &dtos); err != nil {
		return err
	}

	for _, dto := range dtos {
		m, err := dto.ToMetric()
		if err != nil {
			return err
		}
		*metrics = append(*metrics, m)
	}
	return nil
}

type MetricDTO struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"` // counter
	Value *float64 `json:"value,omitempty"` // gauge
	Hash  string   `json:"hash,omitempty"`
}

func (dto *MetricDTO) ToMetric() (Metric, error) {
	if dto.ID == "" {
		return nil, ErrEmptyName
	}
	switch MetricType(dto.MType) {
	case MetricTypeGauge:
		if dto.Value == nil {
			return nil, ErrInvalidValue
		}
		return NewGaugeMetric(dto.ID, *dto.Value), nil
	case MetricTypeCounter:
		if dto.Delta == nil {
			return nil, ErrInvalidValue
		}
		return NewCounterMetric(dto.ID, *dto.Delta), nil
	default:
		return nil, ErrInvalidType
	}
}

type BaseMetric struct {
	name string
}

func (b *BaseMetric) GetName() string {
	return b.name
}

type GaugeMetric struct {
	BaseMetric
	value float64
}

type CounterMetric struct {
	BaseMetric
	value int64
}

func (g *GaugeMetric) GetType() MetricType   { return MetricTypeGauge }
func (c *CounterMetric) GetType() MetricType { return MetricTypeCounter }

func (g *GaugeMetric) GetValue() float64 { return g.value }
func (c *CounterMetric) GetValue() int64 { return c.value }

func (g *GaugeMetric) ToDTO() MetricDTO {
	return MetricDTO{
		ID:    g.GetName(),
		MType: string(MetricTypeGauge),
		Value: &g.value,
	}
}
func (c *CounterMetric) ToDTO() MetricDTO {
	return MetricDTO{
		ID:    c.GetName(),
		MType: string(MetricTypeCounter),
		Delta: &c.value,
	}
}

func NewMetric(mType string, name string, value string) (Metric, error) {
	if name == "" {
		return nil, ErrEmptyName
	}
	switch MetricType(mType) {
	case MetricTypeGauge:
		value, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, ErrInvalidValue
		}
		return NewGaugeMetric(name, value), nil
	case MetricTypeCounter:
		value, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, ErrInvalidValue
		}
		return NewCounterMetric(name, value), nil
	default:
		return nil, ErrInvalidType
	}

}

func NewGaugeMetric(name string, value float64) Metric {
	return &GaugeMetric{
		BaseMetric: BaseMetric{name: name},
		value:      value,
	}
}

func NewCounterMetric(name string, value int64) Metric {
	return &CounterMetric{
		BaseMetric: BaseMetric{name: name},
		value:      value,
	}
}
