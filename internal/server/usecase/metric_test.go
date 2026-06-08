package usecase_test

import (
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/adapter/out/memory"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestUseCase() *usecase.MetricUseCase {
	store := memory.NewMetricStorage()
	repo := memory.NewMetricRepo(store)
	return usecase.NewMetricUseCase(repo)
}

func TestMetricUseCase_SaveAndGetGauge(t *testing.T) {
	uc := newTestUseCase()
	ctx := t.Context()

	err := uc.Save(ctx, entity.NewGaugeMetric("cpu", 42.5))
	require.NoError(t, err)

	m, err := uc.Get(ctx, "gauge", "cpu")
	require.NoError(t, err)

	gauge, ok := m.(*entity.GaugeMetric)
	require.True(t, ok)
	assert.Equal(t, 42.5, gauge.Value)
	assert.Equal(t, "cpu", gauge.GetName())
}

func TestMetricUseCase_SaveAndGetCounter(t *testing.T) {
	uc := newTestUseCase()
	ctx := t.Context()

	err := uc.Save(ctx, entity.NewCounterMetric("hits", 10))
	require.NoError(t, err)

	err = uc.Save(ctx, entity.NewCounterMetric("hits", 5))
	require.NoError(t, err)

	m, err := uc.Get(ctx, "counter", "hits")
	require.NoError(t, err)

	counter, ok := m.(*entity.CounterMetric)
	require.True(t, ok)
	assert.Equal(t, int64(15), counter.Value)
}

func TestMetricUseCase_GetNotFound(t *testing.T) {
	uc := newTestUseCase()
	ctx := t.Context()

	_, err := uc.Get(ctx, "gauge", "nonexistent")
	assert.ErrorIs(t, err, entity.ErrMetricNotFound)

	_, err = uc.Get(ctx, "counter", "nonexistent")
	assert.ErrorIs(t, err, entity.ErrMetricNotFound)
}

func TestMetricUseCase_GetInvalidType(t *testing.T) {
	uc := newTestUseCase()
	ctx := t.Context()

	_, err := uc.Get(ctx, "unknown", "cpu")
	assert.ErrorIs(t, err, entity.ErrInvalidMetricType)
}

func TestMetricUseCase_GaugeOverwrites(t *testing.T) {
	uc := newTestUseCase()
	ctx := t.Context()

	require.NoError(t, uc.Save(ctx, entity.NewGaugeMetric("temp", 36.6)))
	require.NoError(t, uc.Save(ctx, entity.NewGaugeMetric("temp", 37.0)))

	m, err := uc.Get(ctx, "gauge", "temp")
	require.NoError(t, err)
	assert.Equal(t, 37.0, m.(*entity.GaugeMetric).Value)
}

func TestMetricUseCase_GetAll(t *testing.T) {
	uc := newTestUseCase()
	ctx := t.Context()

	require.NoError(t, uc.Save(ctx, entity.NewGaugeMetric("cpu", 1.1)))
	require.NoError(t, uc.Save(ctx, entity.NewGaugeMetric("mem", 2.2)))
	require.NoError(t, uc.Save(ctx, entity.NewCounterMetric("hits", 100)))

	metrics, err := uc.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, metrics, 3)
}

func TestMetricUseCase_UpdateBatch(t *testing.T) {
	uc := newTestUseCase()
	ctx := t.Context()

	batch := []entity.Metric{
		entity.NewGaugeMetric("cpu", 55.5),
		entity.NewGaugeMetric("mem", 128.0),
		entity.NewCounterMetric("requests", 10),
		entity.NewCounterMetric("requests", 20),
	}

	err := uc.UpdateBatch(ctx, batch)
	require.NoError(t, err)

	m, err := uc.Get(ctx, "gauge", "cpu")
	require.NoError(t, err)
	assert.Equal(t, 55.5, m.(*entity.GaugeMetric).Value)

	m, err = uc.Get(ctx, "counter", "requests")
	require.NoError(t, err)
	assert.Equal(t, int64(30), m.(*entity.CounterMetric).Value)
}
