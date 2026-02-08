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

func (s *MetricMemoryStorage) UpdateGauge(name string, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
	return nil
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

func (s *MetricMemoryStorage) UpdateCounter(name string, value int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
	return nil
}

func (s *MetricMemoryStorage) GetAllGauges() (map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copyMap := make(map[string]float64, len(s.gauges))
	maps.Copy(copyMap, s.gauges)
	return copyMap, nil
}

func (s *MetricMemoryStorage) GetAllCounters() (map[string]int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copyMap := make(map[string]int64, len(s.counters))
	maps.Copy(copyMap, s.counters)
	return copyMap, nil
}
