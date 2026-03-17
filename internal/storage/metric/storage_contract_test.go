package metric

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func runStorageTests(t *testing.T, storage Storage) {
	t.Helper()
	ctx := context.Background()

	t.Run("UpdateGauge_Upsert", func(t *testing.T) {
		_, err := storage.UpdateGauge(ctx, "cpu", 0.5)
		assert.NoError(t, err)

		value, err := storage.GetGauge(ctx, "cpu")
		assert.NoError(t, err)
		assert.Equal(t, 0.5, value)

		_, err = storage.UpdateGauge(ctx, "cpu", 0.8)
		assert.NoError(t, err)

		value, err = storage.GetGauge(ctx, "cpu")
		assert.NoError(t, err)
		assert.Equal(t, 0.8, value)
	})

	t.Run("UpdateCounter_Accumulates", func(t *testing.T) {
		_, err := storage.UpdateCounter(ctx, "requests", 10)
		assert.NoError(t, err)

		value, err := storage.GetCounter(ctx, "requests")
		assert.NoError(t, err)
		assert.Equal(t, int64(10), value)

		_, err = storage.UpdateCounter(ctx, "requests", 5)
		assert.NoError(t, err)

		value, err = storage.GetCounter(ctx, "requests")
		assert.NoError(t, err)
		assert.Equal(t, int64(15), value)
	})

	t.Run("GetNotFound", func(t *testing.T) {
		_, err := storage.GetGauge(ctx, "nonexistent_gauge")
		assert.ErrorIs(t, err, ErrNotFound)

		_, err = storage.GetCounter(ctx, "nonexistent_counter")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("GetAll", func(t *testing.T) {
		storage.UpdateGauge(ctx, "memory", 128.0)
		storage.UpdateCounter(ctx, "hits", 3)

		gauges, err := storage.GetAllGauges(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(gauges), 1)
		assert.Equal(t, 128.0, gauges["memory"])

		counters, err := storage.GetAllCounters(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(counters), 1)
		assert.Equal(t, int64(3), counters["hits"])
	})
}
