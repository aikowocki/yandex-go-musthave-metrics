package memory

import (
	"context"
	"maps"
	"sync"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
)

type MetricStore struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMetricStorage() *MetricStore {
	return &MetricStore{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *MetricStore) GetGauge(ctx context.Context, name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.gauges[name]
	if !ok {
		return 0, entity.ErrMetricNotFound
	}
	return v, nil
}

func (s *MetricStore) UpdateGauge(ctx context.Context, name string, value float64) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
	return s.gauges[name], nil
}

func (s *MetricStore) GetCounter(ctx context.Context, name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.counters[name]
	if !ok {
		return 0, entity.ErrMetricNotFound
	}
	return v, nil
}

func (s *MetricStore) UpdateCounter(ctx context.Context, name string, value int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
	return s.counters[name], nil
}

func (s *MetricStore) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.gauges), nil
}

func (s *MetricStore) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.counters), nil
}

func (s *MetricStore) RestoreBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters = counters
	s.gauges = gauges
	return nil
}

func (s *MetricStore) UpdateBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(gauges) > 0 {
		maps.Copy(s.gauges, gauges)
	}

	for name, value := range counters {
		s.counters[name] += value
	}
	return nil
}
