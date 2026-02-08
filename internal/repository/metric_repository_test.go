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
	gauge := &model.GaugeMetric{Name: "cpu", Value: 0.5}
	err := repo.Save(gauge)
	assert.NoError(t, err)

	// Проверяем
	m, err := repo.Get(string(model.MetricTypeGauge), "cpu")
	assert.NoError(t, err)
	assert.IsType(t, &model.GaugeMetric{}, m)
	assert.Equal(t, 0.5, m.(*model.GaugeMetric).Value)
}

func TestRepository_CounterSum(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := NewMetricRepository(stor)

	// Сохраняем counter дважды
	counter1 := &model.CounterMetric{Name: "requests", Value: 10}
	repo.Save(counter1)

	counter2 := &model.CounterMetric{Name: "requests", Value: 5}
	repo.Save(counter2)

	// Проверяем сумму
	m, err := repo.Get(string(model.MetricTypeCounter), "requests")
	assert.NoError(t, err)
	assert.Equal(t, int64(15), m.(*model.CounterMetric).Value)
}

func TestRepository_GetAll(t *testing.T) {
	stor := metric.NewMetricMemoryStorage()
	repo := NewMetricRepository(stor)

	// Сохраняем несколько метрик
	repo.Save(&model.GaugeMetric{Name: "cpu", Value: 0.5})
	repo.Save(&model.GaugeMetric{Name: "memory", Value: 128.0})
	repo.Save(&model.CounterMetric{Name: "requests", Value: 10})
	repo.Save(&model.CounterMetric{Name: "errors", Value: 2})

	// Получаем все метрики
	metrics, err := repo.GetAll()
	assert.NoError(t, err)
	assert.Len(t, metrics, 4)

	// Проверяем что все метрики есть
	names := make(map[string]bool)
	for _, m := range metrics {
		switch m := m.(type) {
		case *model.GaugeMetric:
			names[m.Name] = true
			if m.Name == "cpu" {
				assert.Equal(t, 0.5, m.Value)
			}
			if m.Name == "memory" {
				assert.Equal(t, 128.0, m.Value)
			}
		case *model.CounterMetric:
			names[m.Name] = true
			if m.Name == "requests" {
				assert.Equal(t, int64(10), m.Value)
			}
			if m.Name == "errors" {
				assert.Equal(t, int64(2), m.Value)
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
