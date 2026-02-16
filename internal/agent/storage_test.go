package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalMetrics_SetGauge(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value float64
	}{
		{
			name:  "set new gauge",
			key:   "test",
			value: 3.14,
		},
		{
			name:  "overwrite gauge",
			key:   "test",
			value: 5.5,
		},
	}

	storage := NewLocalStorage()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage.SetGauge(tt.key, tt.value)
			v, ok := storage.GetGauge(tt.key)
			require.True(t, ok, "gauge '%s' should exist", tt.key)
			assert.Equal(t, tt.value, v)
		})
	}
}

func TestLocalMetrics_AddCounter(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value int64
		want  int64
	}{
		{
			name:  "add new counter",
			key:   "test",
			value: 5,
			want:  5,
		},
		{
			name:  "increment counter",
			key:   "test",
			value: 3,
			want:  8,
		},
	}

	storage := NewLocalStorage()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage.AddCounter(tt.key, tt.value)
			v, ok := storage.GetCounter(tt.key)
			require.True(t, ok, "counter '%s' should exist", tt.key)
			assert.Equal(t, tt.want, v)
		})
	}
}

func TestLocalMetrics_SnapshotCounters(t *testing.T) {
	storage := NewLocalStorage()

	storage.AddCounter("test", 10)
	storage.AddCounter("test2", 5)

	snapshot := storage.SnapshotCounters()
	assert.Equal(t, int64(10), snapshot["test"])
	assert.Equal(t, int64(5), snapshot["test2"])

	empty := storage.SnapshotCounters()
	assert.Empty(t, empty)

	storage.AddCounter("new", 1)
	snapshot["new"] = 999

	v, ok := storage.GetCounter("new")
	require.True(t, ok)
	assert.Equal(t, int64(1), v)

}
