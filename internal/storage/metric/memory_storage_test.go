// internal/storage/metric/memory_storage_test.go
package metric

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemStorage_UpdateGauge(t *testing.T) {
	storage := NewMetricMemoryStorage()
	ctx := context.Background()

	// Сохраняем
	_, err := storage.UpdateGauge(ctx, "cpu", 0.5)
	assert.NoError(t, err)

	// Проверяем
	value, err := storage.GetGauge(ctx, "cpu")
	assert.NoError(t, err)
	assert.Equal(t, 0.5, value)

	// Обновляем (должно заменить)
	_, err = storage.UpdateGauge(ctx, "cpu", 0.8)
	assert.NoError(t, err)

	value, err = storage.GetGauge(ctx, "cpu")
	assert.NoError(t, err)
	assert.Equal(t, 0.8, value)
}

func TestMemStorage_UpdateCounter(t *testing.T) {
	storage := NewMetricMemoryStorage()
	ctx := context.Background()

	// Первое значение
	_, err := storage.UpdateCounter(ctx, "requests", 10)
	assert.NoError(t, err)

	value, err := storage.GetCounter(ctx, "requests")
	assert.NoError(t, err)
	assert.Equal(t, int64(10), value)

	// Второе значение (должно суммироваться)
	_, err = storage.UpdateCounter(ctx, "requests", 5)
	assert.NoError(t, err)

	value, err = storage.GetCounter(ctx, "requests")
	assert.NoError(t, err)
	assert.Equal(t, int64(15), value)
}

func TestMemStorage_GetNotFound(t *testing.T) {
	storage := NewMetricMemoryStorage()
	ctx := context.Background()

	_, err := storage.GetGauge(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)

	_, err = storage.GetCounter(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMemStorage_GetAll(t *testing.T) {
	storage := NewMetricMemoryStorage()
	ctx := context.Background()

	storage.UpdateGauge(ctx, "cpu", 0.5)
	storage.UpdateGauge(ctx, "memory", 128.0)
	storage.UpdateCounter(ctx, "requests", 10)

	gauges, err := storage.GetAllGauges(ctx)
	assert.NoError(t, err)
	assert.Len(t, gauges, 2)
	assert.Equal(t, 0.5, gauges["cpu"])
	assert.Equal(t, 128.0, gauges["memory"])

	counters, err := storage.GetAllCounters(ctx)
	assert.NoError(t, err)
	assert.Len(t, counters, 1)
	assert.Equal(t, int64(10), counters["requests"])
}
