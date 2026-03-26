package metric

import (
	"context"
	"maps"
	"sync"
)

type MemoryStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *MemoryStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.gauges[name]
	if !ok {
		return 0, ErrNotFound
	}
	return v, nil
}

func (s *MemoryStorage) UpdateGauge(ctx context.Context, name string, value float64) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
	return s.gauges[name], nil
}

func (s *MemoryStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.counters[name]
	if !ok {
		return 0, ErrNotFound
	}
	return v, nil
}

func (s *MemoryStorage) UpdateCounter(ctx context.Context, name string, value int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
	return s.counters[name], nil
}

func (s *MemoryStorage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.gauges), nil
}

func (s *MemoryStorage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.counters), nil
}

func (s *MemoryStorage) RestoreBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters = counters
	s.gauges = gauges
	return nil
}

func (s *MemoryStorage) UpdateBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error {
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
