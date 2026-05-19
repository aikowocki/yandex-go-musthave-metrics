package usecase

import (
	"context"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
)

type MetricRepository interface {
	// Gauges
	GetGauge(ctx context.Context, name string) (*entity.GaugeMetric, error)
	CreateOrUpdateGauge(context.Context, *entity.GaugeMetric) error
	//Counters
	GetCounter(ctx context.Context, name string) (*entity.CounterMetric, error)
	CreateOrUpdateCounter(context.Context, *entity.CounterMetric) error

	GetAll(ctx context.Context) ([]entity.Metric, error)
	SaveBatch(ctx context.Context, metrics []entity.Metric) error
}

type MetricUseCase struct {
	repo MetricRepository
}

func NewMetricUseCase(repo MetricRepository) *MetricUseCase {

	return &MetricUseCase{repo: repo}
}

func (s *MetricUseCase) Get(ctx context.Context, metricType, name string) (entity.Metric, error) {
	switch metricType {
	case string(entity.MetricTypeGauge):
		return s.repo.GetGauge(ctx, name)
	case string(entity.MetricTypeCounter):
		return s.repo.GetCounter(ctx, name)
	}
	return nil, entity.ErrInvalidMetricType
}

func (s *MetricUseCase) Save(ctx context.Context, metric entity.Metric) error {
	switch m := metric.(type) {
	case *entity.GaugeMetric:
		return s.repo.CreateOrUpdateGauge(ctx, m)
	case *entity.CounterMetric:
		return s.repo.CreateOrUpdateCounter(ctx, m)
	}
	return entity.ErrInvalidMetricType
}

func (s *MetricUseCase) GetAll(ctx context.Context) ([]entity.Metric, error) {
	return s.repo.GetAll(ctx)
}

func (s *MetricUseCase) UpdateBatch(ctx context.Context, metrics []entity.Metric) error {
	return s.repo.SaveBatch(ctx, metrics)
}
