package integration

import (
	"context"
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMetricRepo(t *testing.T, storage port.PGStorage) {
	t.Helper()
	repo := storage.MetricRepo()
	ctx := context.Background()

	t.Run("CreateOrUpdateGauge", func(t *testing.T) {
		gauge := entity.NewGaugeMetric("cpu", 0.5)
		err := repo.CreateOrUpdateGauge(ctx, gauge)
		require.NoError(t, err)

		found, err := repo.GetGauge(ctx, "cpu")
		require.NoError(t, err)
		assert.Equal(t, 0.5, found.Value)

		// upsert — перезаписывает
		gauge2 := entity.NewGaugeMetric("cpu", 0.8)
		err = repo.CreateOrUpdateGauge(ctx, gauge2)
		require.NoError(t, err)

		found, err = repo.GetGauge(ctx, "cpu")
		require.NoError(t, err)
		assert.Equal(t, 0.8, found.Value)
	})

	t.Run("CreateOrUpdateCounter_Accumulates", func(t *testing.T) {
		counter := entity.NewCounterMetric("requests", 10)
		err := repo.CreateOrUpdateCounter(ctx, counter)
		require.NoError(t, err)

		found, err := repo.GetCounter(ctx, "requests")
		require.NoError(t, err)
		assert.Equal(t, int64(10), found.Value)

		// накапливается
		counter2 := entity.NewCounterMetric("requests", 5)
		err = repo.CreateOrUpdateCounter(ctx, counter2)
		require.NoError(t, err)
		assert.Equal(t, int64(15), counter2.Value) // RETURNING value

		found, err = repo.GetCounter(ctx, "requests")
		require.NoError(t, err)
		assert.Equal(t, int64(15), found.Value)
	})

	t.Run("GetGauge_NotFound", func(t *testing.T) {
		_, err := repo.GetGauge(ctx, "nonexistent")
		assert.ErrorIs(t, err, entity.ErrMetricNotFound)
	})

	t.Run("GetCounter_NotFound", func(t *testing.T) {
		_, err := repo.GetCounter(ctx, "nonexistent")
		assert.ErrorIs(t, err, entity.ErrMetricNotFound)
	})

	t.Run("GetAll", func(t *testing.T) {
		metrics, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(metrics), 2) // cpu + requests из предыдущих тестов
	})

	t.Run("SaveBatch", func(t *testing.T) {
		metrics := []entity.Metric{
			entity.NewGaugeMetric("batch_cpu", 1.5),
			entity.NewGaugeMetric("batch_mem", 256.0),
			entity.NewCounterMetric("batch_hits", 10),
		}
		err := repo.SaveBatch(ctx, metrics)
		require.NoError(t, err)

		g, err := repo.GetGauge(ctx, "batch_cpu")
		require.NoError(t, err)
		assert.Equal(t, 1.5, g.Value)

		c, err := repo.GetCounter(ctx, "batch_hits")
		require.NoError(t, err)
		assert.Equal(t, int64(10), c.Value)
	})
}

func TestMetricRepo(t *testing.T) {
	storage := setupPGXStorage(t)
	testMetricRepo(t, storage)
}
