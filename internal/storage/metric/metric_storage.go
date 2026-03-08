package metric

type MetricStorage interface {
	GetGauge(name string) (float64, error)
	UpdateGauge(name string, value float64) (float64, error)

	GetCounter(name string) (int64, error)
	UpdateCounter(name string, value int64) (int64, error)

	GetAllGauges() (map[string]float64, error)
	GetAllCounters() (map[string]int64, error)

	RestoreBatch(gauges map[string]float64, counters map[string]int64) error
}
