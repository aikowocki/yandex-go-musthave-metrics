package memory

import (
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/stretchr/testify/assert"
)

func runStorageTests(t *testing.T, storage MetricStorage) {
	t.Helper()
	ctx := t.Context()

	t.Run("UpdateGauge_Upsert", func(t *testing.T) {
		_, err := storage.UpdateGauge(ctx, "cpu", 0.5)
		assert.NoError(t, err)

		value, err := storage.GetGauge(ctx, "cpu")
		assert.NoError(t, err)
		assert.Equal(t, 0.5, value)

		// upsert — перезаписывает
		_, err = storage.UpdateGauge(ctx, "cpu", 0.8)
		assert.NoError(t, err)

		value, err = storage.GetGauge(ctx, "cpu")
		assert.NoError(t, err)
		assert.Equal(t, 0.8, value)
	})

	t.Run("UpdateCounter_Accumulates", func(t *testing.T) {
		v, err := storage.UpdateCounter(ctx, "requests", 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(10), v)

		// накапливается
		v, err = storage.UpdateCounter(ctx, "requests", 5)
		assert.NoError(t, err)
		assert.Equal(t, int64(15), v)

		value, err := storage.GetCounter(ctx, "requests")
		assert.NoError(t, err)
		assert.Equal(t, int64(15), value)
	})

	t.Run("GetNotFound", func(t *testing.T) {
		_, err := storage.GetGauge(ctx, "nonexistent_gauge")
		assert.ErrorIs(t, err, entity.ErrMetricNotFound)

		_, err = storage.GetCounter(ctx, "nonexistent_counter")
		assert.ErrorIs(t, err, entity.ErrMetricNotFound)
	})

	t.Run("UpdateBatch", func(t *testing.T) {
		// первый батч — mixed gauges и counters
		err := storage.UpdateBatch(ctx,
			map[string]float64{"batch_cpu": 1.5, "batch_mem": 256.0},
			map[string]int64{"batch_hits": 10},
		)
		assert.NoError(t, err)

		v, err := storage.GetGauge(ctx, "batch_cpu")
		assert.NoError(t, err)
		assert.Equal(t, 1.5, v)

		c, err := storage.GetCounter(ctx, "batch_hits")
		assert.NoError(t, err)
		assert.Equal(t, int64(10), c)

		// второй батч — gauge upsert, counter накапливается
		err = storage.UpdateBatch(ctx,
			map[string]float64{"batch_cpu": 9.9},
			map[string]int64{"batch_hits": 5},
		)
		assert.NoError(t, err)

		v, err = storage.GetGauge(ctx, "batch_cpu")
		assert.NoError(t, err)
		assert.Equal(t, 9.9, v)

		c, err = storage.GetCounter(ctx, "batch_hits")
		assert.NoError(t, err)
		assert.Equal(t, int64(15), c)

		// пустой батч не ломает ничего
		err = storage.UpdateBatch(ctx, nil, nil)
		assert.NoError(t, err)
	})

	t.Run("GetAll", func(t *testing.T) {
		_, err := storage.UpdateGauge(ctx, "memory", 128.0)
		assert.NoError(t, err)
		_, err = storage.UpdateCounter(ctx, "hits", 3)
		assert.NoError(t, err)

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
