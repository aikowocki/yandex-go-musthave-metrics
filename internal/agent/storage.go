package agent

type MetricStorage interface {
	SetGauge(name string, value float64)
	AddCounter(name string, value int64)
	GetGauges() map[string]float64
	GetCounters() map[string]int64
	ResetCounters()
}

type LocalMetrics struct {
	gauges   map[string]float64
	counters map[string]int64
}

func NewLocalStorage() MetricStorage {
	return &LocalMetrics{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *LocalMetrics) SetGauge(name string, value float64) {
	m.gauges[name] = value
}

func (m *LocalMetrics) AddCounter(name string, value int64) {
	m.counters[name] += value
}

// GetGauges Получить все метрики типа gauges
func (m *LocalMetrics) GetGauges() map[string]float64 {
	return m.gauges
}

// GetCounters Получить все метрики типа counter
func (m *LocalMetrics) GetCounters() map[string]int64 {
	return m.counters
}

// ResetCounters обнуляем все счетчики
func (m *LocalMetrics) ResetCounters() {
	m.counters = make(map[string]int64)
}
