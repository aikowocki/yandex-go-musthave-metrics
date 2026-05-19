package port

import (
	"context"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
)

type MetricRepository interface {
	GetGauge(ctx context.Context, name string) (*entity.GaugeMetric, error)
	CreateOrUpdateGauge(ctx context.Context, gauge *entity.GaugeMetric) error
	GetCounter(ctx context.Context, name string) (*entity.CounterMetric, error)
	CreateOrUpdateCounter(ctx context.Context, counter *entity.CounterMetric) error
	GetAll(ctx context.Context) ([]entity.Metric, error)
	SaveBatch(ctx context.Context, metrics []entity.Metric) error
}
