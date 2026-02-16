package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectMetrics(t *testing.T) {
	storage := NewLocalStorage()

	CollectMetrics(storage)

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
		_, ok := storage.GetGauge(metricName)
		assert.True(t, ok, "Missing gauge metric: %s", metricName)
	}
	for _, metricName := range expectedCounters {
		_, ok := storage.GetCounter(metricName)
		assert.True(t, ok, "Missing counter metric: %s", metricName)
	}
}

func TestCollectMetrics_PollCountIncrement(t *testing.T) {
	storage := NewLocalStorage()

	CollectMetrics(storage)
	CollectMetrics(storage)

	v, ok := storage.GetCounter("PollCount")
	require.True(t, ok)
	assert.Equal(t, int64(2), v)
}
