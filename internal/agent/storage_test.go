package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalMetrics_SetGauge(t *testing.T) {
	storage := NewLocalStorage()

	storage.SetGauge("test", 3.14)
	assert.Equal(t, 3.14, storage.GetGauges()["test"])

	storage.SetGauge("test", 5.5)
	assert.Equal(t, 5.5, storage.GetGauges()["test"])
}

func TestLocalMetrics_AddCounter(t *testing.T) {
	storage := NewLocalStorage()

	storage.AddCounter("test", 5)
	assert.Equal(t, int64(5), storage.GetCounters()["test"])

	// Сложение
	storage.AddCounter("test", 3)
	assert.Equal(t, int64(8), storage.GetCounters()["test"])
}

func TestLocalMetrics_ResetCounters(t *testing.T) {
	storage := NewLocalStorage()

	storage.AddCounter("test", 10)
	storage.ResetCounters()

	assert.Empty(t, storage.GetCounters())
}
