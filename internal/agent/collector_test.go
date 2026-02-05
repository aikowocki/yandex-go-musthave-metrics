package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectMetrics(t *testing.T) {
	storage := NewLocalStorage()

	CollectMetrics(storage)

	gauges := storage.GetGauges()
	counters := storage.GetCounters()

	var expectedGaugeMetrics = []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "RandomValue",
	}

	var expectedCounters = []string{
		"PollCount",
	}

	for _, metricName := range expectedGaugeMetrics {
		assert.Contains(t, gauges, metricName, "Missing metric: %s", metricName)
	}
	for _, metricName := range expectedCounters {
		assert.Contains(t, counters, metricName, "Missing metric: %s", metricName)
	}
}

func TestCollectMetrics_PollCountIncrement(t *testing.T) {
	storage := NewLocalStorage()

	CollectMetrics(storage)
	CollectMetrics(storage)

	counters := storage.GetCounters()
	assert.Equal(t, int64(2), counters["PollCount"])
}
