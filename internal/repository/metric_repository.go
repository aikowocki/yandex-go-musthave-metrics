package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/storage/metric"
)

var ErrUnknownType = errors.New("unknown metric type")

type MetricRepository struct {
	storage metric.Storage
}

func NewMetricRepository(storage metric.Storage) *MetricRepository {
	return &MetricRepository{storage: storage}
}

// Get Получить одну метрику
func (r *MetricRepository) Get(ctx context.Context, metricType, name string) (model.Metric, error) {
	switch metricType {
	case string(model.MetricTypeCounter):
		v, err := r.storage.GetCounter(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("get counter %q: %w", name, err)
		}
		return model.NewCounterMetric(name, v), nil

	case string(model.MetricTypeGauge):
		v, err := r.storage.GetGauge(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("get gauge %q: %w", name, err)
		}
		return model.NewGaugeMetric(name, v), nil

	default:
		return nil, ErrUnknownType
	}
}

// GetAll Получить все метрики
func (r *MetricRepository) GetAll(ctx context.Context) ([]model.Metric, error) {
	var result []model.Metric

	gauges, err := r.storage.GetAllGauges(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all gauges: %w", err)
	}
	for name, v := range gauges {
		result = append(result, model.NewGaugeMetric(name, v))
	}

	counters, err := r.storage.GetAllCounters(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all counters: %w", err)
	}
	for name, v := range counters {
		result = append(result, model.NewCounterMetric(name, v))
	}

	return result, nil
}

// Save Сохранить или обновить метрику
func (r *MetricRepository) Save(ctx context.Context, m model.Metric) (model.Metric, error) {
	switch m := m.(type) {
	case *model.CounterMetric:
		newVal, err := r.storage.UpdateCounter(ctx, m.GetName(), m.GetValue())
		if err != nil {
			return nil, fmt.Errorf("save counter %q: %w", m.GetName(), err)
		}
		return model.NewCounterMetric(m.GetName(), newVal), nil
	case *model.GaugeMetric:
		newVal, err := r.storage.UpdateGauge(ctx, m.GetName(), m.GetValue())
		if err != nil {
			return nil, fmt.Errorf("save gauge %q: %w", m.GetName(), err)
		}
		return model.NewGaugeMetric(m.GetName(), newVal), nil

	default:
		return nil, fmt.Errorf("unknown metric type %T: %w", m, ErrUnknownType)
	}
}

func (r *MetricRepository) SaveBatch(ctx context.Context, metrics model.MetricList) error {
	gauges := make(map[string]float64)
	counters := make(map[string]int64)

	for _, m := range metrics {
		switch m := m.(type) {
		case *model.CounterMetric:
			counters[m.GetName()] += m.GetValue()
		case *model.GaugeMetric:
			gauges[m.GetName()] = m.GetValue()
		default:
			return fmt.Errorf("unknown metric type %T: %w", m, ErrUnknownType)
		}
	}
	return r.storage.UpdateBatch(ctx, gauges, counters)
}
