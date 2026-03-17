package metric

import (
	"context"
	"maps"
	"sync"
)

type MetricMemoryStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMetricMemoryStorage() *MetricMemoryStorage {
	return &MetricMemoryStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *MetricMemoryStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.gauges[name]
	if !ok {
		return 0, ErrNotFound
	}
	return v, nil
}

func (s *MetricMemoryStorage) UpdateGauge(ctx context.Context, name string, value float64) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
	return s.gauges[name], nil
}

func (s *MetricMemoryStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.counters[name]
	if !ok {
		return 0, ErrNotFound
	}
	return v, nil
}

func (s *MetricMemoryStorage) UpdateCounter(ctx context.Context, name string, value int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
	return s.counters[name], nil
}

func (s *MetricMemoryStorage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.gauges), nil
}

func (s *MetricMemoryStorage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.counters), nil
}

func (s *MetricMemoryStorage) RestoreBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters = counters
	s.gauges = gauges
	return nil
}
