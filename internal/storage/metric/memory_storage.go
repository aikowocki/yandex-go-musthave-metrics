package metric

import (
	"errors"
	"maps"
	"sync"
)

var (
	ErrNotFound = errors.New("not found in storage")
)

type MetricMemoryStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMetricMemoryStorage() MetricStorage {
	return &MetricMemoryStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *MetricMemoryStorage) GetGauge(name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.gauges[name]
	if !ok {
		return 0, ErrNotFound
	}
	return v, nil
}

func (s *MetricMemoryStorage) UpdateGauge(name string, value float64) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
	return s.gauges[name], nil
}

func (s *MetricMemoryStorage) GetCounter(name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.counters[name]
	if !ok {
		return 0, ErrNotFound
	}
	return v, nil
}

func (s *MetricMemoryStorage) UpdateCounter(name string, value int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
	return s.counters[name], nil
}

func (s *MetricMemoryStorage) GetAllGauges() (map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.gauges), nil
}

func (s *MetricMemoryStorage) GetAllCounters() (map[string]int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.counters), nil
}

func (s *MetricMemoryStorage) RestoreBatch(gauges map[string]float64, counters map[string]int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters = counters
	s.gauges = gauges
	return nil
}
