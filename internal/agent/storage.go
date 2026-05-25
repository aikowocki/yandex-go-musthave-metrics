package agent

import (
	"maps"
	"sync"
)

// MetricStorage — интерфейс локального хранилища метрик агента.
// Все методы потокобезопасны и могут вызываться из нескольких горутин одновременно.
type MetricStorage interface {
	SetGauge(name string, value float64)
	GetGauge(name string) (float64, bool)
	AddCounter(name string, value int64)
	GetCounter(name string) (int64, bool)
	ForEachGauge(fn func(name string, value float64))
	SnapshotGauges() map[string]float64
	SnapshotCounters() map[string]int64
}

// LocalMetrics — потокобезопасная реализация MetricStorage на основе map с sync.RWMutex.
type LocalMetrics struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewLocalStorage создаёт новое пустое хранилище метрик.
func NewLocalStorage() MetricStorage {
	return &LocalMetrics{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *LocalMetrics) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *LocalMetrics) GetGauge(name string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.gauges[name]
	return v, ok
}

func (m *LocalMetrics) AddCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *LocalMetrics) GetCounter(name string) (int64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.counters[name]
	return v, ok
}

func (m *LocalMetrics) ForEachGauge(fn func(name string, value float64)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for name, value := range m.gauges {
		fn(name, value)
	}
}

// SnapshotGauges возвращает копию gauges
func (m *LocalMetrics) SnapshotGauges() map[string]float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Clone(m.gauges)
}

// SnapshotCounters возвращает копию counters и поcле очищает их
func (m *LocalMetrics) SnapshotCounters() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	defer func() { m.counters = make(map[string]int64) }()

	return maps.Clone(m.counters)
}
