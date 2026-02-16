package service

import (
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/repository"
)

type MetricService interface {
	Update(metricType, name, value string) error
	Get(metricType, name string) (model.Metric, error)
	GetAll() ([]model.Metric, error)
}

type metricService struct {
	repo *repository.MetricRepository
}

func NewMetricService(repo *repository.MetricRepository) MetricService {
	return &metricService{repo: repo}
}

func (s *metricService) Update(metricType, name, value string) error {
	m, err := model.NewMetric(metricType, name, value)

	if err != nil {
		return err
	}

	return s.repo.Save(m)
}

func (s *metricService) Get(metricType, name string) (model.Metric, error) {
	return s.repo.Get(metricType, name)
}

func (s *metricService) GetAll() ([]model.Metric, error) {
	return s.repo.GetAll()
}
