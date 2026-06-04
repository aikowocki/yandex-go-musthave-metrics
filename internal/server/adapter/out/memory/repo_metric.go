package memory

import (
	"context"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
)

type MetricStorage interface {
	GetGauge(ctx context.Context, name string) (float64, error)
	UpdateGauge(ctx context.Context, name string, value float64) (float64, error)

	GetCounter(ctx context.Context, name string) (int64, error)
	UpdateCounter(ctx context.Context, name string, value int64) (int64, error)

	GetAllGauges(ctx context.Context) (map[string]float64, error)
	GetAllCounters(ctx context.Context) (map[string]int64, error)

	UpdateBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error
}

type MetricRepo struct {
	storage MetricStorage
}

func NewMetricRepo(storage MetricStorage) *MetricRepo {
	return &MetricRepo{storage: storage}
}

func (r *MetricRepo) GetGauge(ctx context.Context, name string) (*entity.GaugeMetric, error) {
	value, err := r.storage.GetGauge(ctx, name)
	if err != nil {
		return nil, err
	}
	return &entity.GaugeMetric{
		BaseMetric: entity.BaseMetric{Name: name},
		Value:      value,
	}, nil
}

func (r *MetricRepo) GetCounter(ctx context.Context, name string) (*entity.CounterMetric, error) {
	value, err := r.storage.GetCounter(ctx, name)
	if err != nil {
		return nil, err
	}
	return &entity.CounterMetric{
		BaseMetric: entity.BaseMetric{Name: name},
		Value:      value,
	}, nil
}

func (r *MetricRepo) CreateOrUpdateGauge(ctx context.Context, gauge *entity.GaugeMetric) error {
	_, err := r.storage.UpdateGauge(ctx, gauge.Name, gauge.Value)
	return err
}

func (r *MetricRepo) CreateOrUpdateCounter(ctx context.Context, counter *entity.CounterMetric) error {
	v, err := r.storage.UpdateCounter(ctx, counter.Name, counter.Value)
	if err != nil {
		return err
	}
	counter.Value = v
	return nil
}

func (r *MetricRepo) GetAll(ctx context.Context) ([]entity.Metric, error) {
	gauges, err := r.storage.GetAllGauges(ctx)
	if err != nil {
		return nil, err
	}
	counters, err := r.storage.GetAllCounters(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]entity.Metric, 0, len(gauges)+len(counters))
	for name, value := range gauges {
		result = append(result, &entity.GaugeMetric{
			BaseMetric: entity.BaseMetric{Name: name},
			Value:      value,
		})
	}
	for name, value := range counters {
		result = append(result, &entity.CounterMetric{
			BaseMetric: entity.BaseMetric{Name: name},
			Value:      value,
		})
	}
	return result, nil
}

func (r *MetricRepo) SaveBatch(ctx context.Context, metrics []entity.Metric) error {
	gauges := make(map[string]float64)
	counters := make(map[string]int64)

	for _, m := range metrics {
		switch v := m.(type) {
		case *entity.GaugeMetric:
			gauges[v.Name] = v.Value
		case *entity.CounterMetric:
			counters[v.Name] += v.Value
		}
	}
	return r.storage.UpdateBatch(ctx, gauges, counters)
}
