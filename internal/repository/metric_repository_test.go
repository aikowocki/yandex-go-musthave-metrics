package repository

import (
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"github.com/aikowocki/yandex-go-musthave-metrics/internal/storage/metric"
	"github.com/stretchr/testify/assert"
)

func TestRepository_Save(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := NewMetricRepository(stor)

	// Сохраняем gauge
	gauge := model.NewGaugeMetric("cpu", 0.5)
	_, err := repo.Save(gauge)
	assert.NoError(t, err)

	// Проверяем
	m, err := repo.Get(string(model.MetricTypeGauge), "cpu")
	assert.NoError(t, err)
	assert.Equal(t, 0.5, m.(*model.GaugeMetric).GetValue())
}

func TestRepository_CounterSum(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := NewMetricRepository(stor)

	// Сохраняем counter дважды
	repo.Save(model.NewCounterMetric("requests", 10))
	repo.Save(model.NewCounterMetric("requests", 5))

	// Проверяем сумму
	m, err := repo.Get(string(model.MetricTypeCounter), "requests")
	assert.NoError(t, err)
	assert.Equal(t, int64(15), m.(*model.CounterMetric).GetValue())
}

func TestRepository_GetAll(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := NewMetricRepository(stor)

	// Сохраняем несколько метрик
	repo.Save(model.NewGaugeMetric("cpu", 0.5))
	repo.Save(model.NewGaugeMetric("memory", 128.0))
	repo.Save(model.NewCounterMetric("requests", 10))
	repo.Save(model.NewCounterMetric("errors", 2))

	// Получаем все метрики
	metrics, err := repo.GetAll()
	assert.NoError(t, err)
	assert.Len(t, metrics, 4)

	// Проверяем что все метрики есть
	names := make(map[string]bool)
	for _, m := range metrics {
		switch m := m.(type) {
		case *model.GaugeMetric:
			names[m.GetName()] = true
			if m.GetName() == "cpu" {
				assert.Equal(t, 0.5, m.GetValue())
			}
			if m.GetName() == "memory" {
				assert.Equal(t, 128.0, m.GetValue())
			}
		case *model.CounterMetric:
			names[m.GetName()] = true
			if m.GetName() == "requests" {
				assert.Equal(t, int64(10), m.GetValue())
			}
			if m.GetName() == "errors" {
				assert.Equal(t, int64(2), m.GetValue())
			}
		}
	}

	assert.True(t, names["cpu"])
	assert.True(t, names["memory"])
	assert.True(t, names["requests"])
	assert.True(t, names["errors"])
}

func TestRepository_GetAll_Empty(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := NewMetricRepository(stor)

	// Получаем все метрики из пустого storage
	metrics, err := repo.GetAll()
	assert.NoError(t, err)
	assert.Empty(t, metrics)
}

func TestRepository_Get_NotFound(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := NewMetricRepository(stor)

	// Пытаемся получить несуществующую метрику
	_, err := repo.Get(string(model.MetricTypeGauge), "nonexistent")
	assert.Error(t, err)
	assert.ErrorIs(t, err, metric.ErrNotFound)
}

func TestRepository_Get_InvalidType(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := NewMetricRepository(stor)

	// Пытаемся получить с неправильным типом
	_, err := repo.Get("invalid", "test")
	assert.Error(t, err)
	assert.Equal(t, ErrUnknownType, err)
}

func TestRepository_Save_ReturnsUpdatedValue(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := NewMetricRepository(stor)

	// Gauge — Save возвращает сохранённое значение
	saved, err := repo.Save(model.NewGaugeMetric("cpu", 0.5))
	assert.NoError(t, err)
	assert.Equal(t, "cpu", saved.GetName())
	assert.Equal(t, 0.5, saved.(*model.GaugeMetric).GetValue())

	// Counter — Save возвращает накопленную сумму
	saved, err = repo.Save(model.NewCounterMetric("hits", 10))
	assert.NoError(t, err)
	assert.Equal(t, int64(10), saved.(*model.CounterMetric).GetValue())

	saved, err = repo.Save(model.NewCounterMetric("hits", 5))
	assert.NoError(t, err)
	assert.Equal(t, int64(15), saved.(*model.CounterMetric).GetValue())
}

func TestRepository_Save_DoesNotMutateInput(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := NewMetricRepository(stor)

	// Сохраняем counter с delta 10
	repo.Save(model.NewCounterMetric("hits", 10))

	// Создаём новый объект с delta 5
	input := model.NewCounterMetric("hits", 5)

	// Save должен вернуть сумму 15, но input должен остаться 5
	saved, _ := repo.Save(input)
	assert.Equal(t, int64(15), saved.(*model.CounterMetric).GetValue())
	assert.Equal(t, int64(5), input.(*model.CounterMetric).GetValue())
}
