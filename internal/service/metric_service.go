package service

import (
	"context"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/repository"
)

type MetricService interface {
	Update(ctx context.Context, metric model.Metric) (model.Metric, error)
	Get(ctx context.Context, metricType, name string) (model.Metric, error)
	GetAll(ctx context.Context) ([]model.Metric, error)
}

type metricService struct {
	repo *repository.MetricRepository
}

func NewMetricService(repo *repository.MetricRepository) MetricService {
	return &metricService{repo: repo}
}

func (s *metricService) Update(ctx context.Context, m model.Metric) (model.Metric, error) {
	return s.repo.Save(ctx, m)
}

func (s *metricService) Get(ctx context.Context, metricType, name string) (model.Metric, error) {
	return s.repo.Get(ctx, metricType, name)
}

func (s *metricService) GetAll(ctx context.Context) ([]model.Metric, error) {
	return s.repo.GetAll(ctx)
}
