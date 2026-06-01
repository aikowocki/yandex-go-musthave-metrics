// Package usecase содержит бизнес-логику работы с метриками.
package usecase

import (
	"context"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
)

// MetricUseCase реализует бизнес-логику сохранения, получения и пакетного обновления метрик.
type MetricUseCase struct {
	repo port.MetricRepository
}

// NewMetricUseCase создаёт новый экземпляр MetricUseCase с указанным репозиторием.
func NewMetricUseCase(repo port.MetricRepository) *MetricUseCase {
	return &MetricUseCase{repo: repo}
}

// Get возвращает метрику по типу и имени. Возвращает ошибку если тип невалиден или метрика не найдена.
func (s *MetricUseCase) Get(ctx context.Context, metricType, name string) (entity.Metric, error) {
	switch metricType {
	case string(entity.MetricTypeGauge):
		return s.repo.GetGauge(ctx, name)
	case string(entity.MetricTypeCounter):
		return s.repo.GetCounter(ctx, name)
	}
	return nil, entity.ErrInvalidMetricType
}

// Save сохраняет или обновляет метрику. Для counter — прибавляет значение к существующему.
func (s *MetricUseCase) Save(ctx context.Context, metric entity.Metric) error {
	switch m := metric.(type) {
	case *entity.GaugeMetric:
		return s.repo.CreateOrUpdateGauge(ctx, m)
	case *entity.CounterMetric:
		return s.repo.CreateOrUpdateCounter(ctx, m)
	}
	return entity.ErrInvalidMetricType
}

// GetAll возвращает все сохранённые метрики (gauge + counter).
func (s *MetricUseCase) GetAll(ctx context.Context) ([]entity.Metric, error) {
	return s.repo.GetAll(ctx)
}

// UpdateBatch атомарно обновляет пачку метрик за одну операцию.
func (s *MetricUseCase) UpdateBatch(ctx context.Context, metrics []entity.Metric) error {
	return s.repo.SaveBatch(ctx, metrics)
}
