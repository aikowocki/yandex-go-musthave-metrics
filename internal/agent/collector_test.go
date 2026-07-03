package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectMetrics(t *testing.T) {
	storage := NewLocalStorage()

	CollectMetrics(storage)

	gaugeMetrics := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "RandomValue",
	}

	for _, name := range gaugeMetrics {
		value, ok := storage.GetGauge(name)
		assert.True(t, ok, "gauge %s should exist", name)
		assert.GreaterOrEqual(t, value, 0.0, "gauge %s should be >= 0", name)
	}

	// Sys = сумма системной памяти, на работающем процессе всегда > 0
	sys, _ := storage.GetGauge("Sys")
	assert.Greater(t, sys, 0.0, "Sys should be > 0 for a running process")

	// Проверяем что PollCount увеличился
	pollCount, ok := storage.GetCounter("PollCount")
	assert.True(t, ok, "PollCount should exist")
	assert.Equal(t, int64(1), pollCount, "PollCount should be 1")

	// Повторный сбор — PollCount должен накопиться
	CollectMetrics(storage)
	pollCount, ok = storage.GetCounter("PollCount")
	assert.True(t, ok)
	assert.Equal(t, int64(2), pollCount, "PollCount should be 2")
}

func TestCollectSystemMetrics(t *testing.T) {
	storage := NewLocalStorage()

	// На darwin/linux gopsutil всегда отдаёт память — проверяем без if ok,
	// чтобы тест реально падал при поломке сбора.
	CollectSystemMetrics(storage)

	totalMem, ok := storage.GetGauge("TotalMemory")
	require.True(t, ok, "TotalMemory must be collected")
	assert.Greater(t, totalMem, 0.0, "TotalMemory should be > 0")

	freeMem, ok := storage.GetGauge("FreeMemory")
	require.True(t, ok, "FreeMemory must be collected")
	assert.Greater(t, freeMem, 0.0, "FreeMemory should be > 0")
	assert.LessOrEqual(t, freeMem, totalMem, "FreeMemory should not exceed TotalMemory")

	// Должна быть хотя бы одна CPU-метрика (минимум одно ядро).
	cpu1, ok := storage.GetGauge("CPUUtilization1")
	require.True(t, ok, "at least CPUUtilization1 must be collected")
	assert.GreaterOrEqual(t, cpu1, 0.0)
	assert.LessOrEqual(t, cpu1, 100.0)
}
