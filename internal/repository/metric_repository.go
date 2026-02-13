package repository

import (
	"errors"
	"fmt"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/storage/metric"
)

var ErrUnknownType = errors.New("unknown metric type")

type MetricRepository struct {
	storage metric.MetricStorage
}

func NewMetricRepository(storage metric.MetricStorage) *MetricRepository {
	return &MetricRepository{storage: storage}
}

// Get Получить одну метрику
func (r *MetricRepository) Get(metricType, name string) (model.Metric, error) {
	switch metricType {
	case string(model.MetricTypeCounter):
		v, err := r.storage.GetCounter(name)
		if err != nil {
			return nil, fmt.Errorf("get counter %q: %w", name, err)
		}
		return &model.CounterMetric{
			Name:  name,
			Value: v,
		}, nil

	case string(model.MetricTypeGauge):
		v, err := r.storage.GetGauge(name)
		if err != nil {
			return nil, fmt.Errorf("get gauge %q: %w", name, err)
		}
		return &model.GaugeMetric{
			Name:  name,
			Value: v,
		}, nil

	default:
		return nil, ErrUnknownType
	}
}

// GetAll Получить все метрики
func (r *MetricRepository) GetAll() ([]model.Metric, error) {
	var result []model.Metric

	gauges, err := r.storage.GetAllGauges()
	if err != nil {
		return nil, fmt.Errorf("get all gauges: %w", err)
	}
	for name, v := range gauges {
		result = append(result, &model.GaugeMetric{
			Name:  name,
			Value: v,
		})
	}

	counters, err := r.storage.GetAllCounters()
	if err != nil {
		return nil, fmt.Errorf("get all counters: %w", err)
	}
	for name, v := range counters {
		result = append(result, &model.CounterMetric{
			Name:  name,
			Value: v,
		})
	}

	return result, nil
}

// Save Сохранить или обновить метрику
func (r *MetricRepository) Save(m model.Metric) error {
	switch m := m.(type) {
	case *model.CounterMetric:
		if err := r.storage.UpdateCounter(m.Name, m.Value); err != nil {
			return fmt.Errorf("save counter %q: %w", m.Name, err)
		}
		return nil
	case *model.GaugeMetric:
		if err := r.storage.UpdateGauge(m.Name, m.Value); err != nil {
			return fmt.Errorf("save gauge %q: %w", m.Name, err)
		}
		return nil
	default:
		return ErrUnknownType
	}
}
