package entity_test

import (
	"testing"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/entity"
	"github.com/stretchr/testify/assert"
)

func TestNewGaugeMetric(t *testing.T) {
	g := entity.NewGaugeMetric("cpu", 3.14)

	assert.Equal(t, "cpu", g.GetName())
	assert.Equal(t, entity.MetricTypeGauge, g.GetType())
	assert.Equal(t, 3.14, g.GetValue())
}

func TestNewCounterMetric(t *testing.T) {
	c := entity.NewCounterMetric("hits", 10)

	assert.Equal(t, "hits", c.GetName())
	assert.Equal(t, entity.MetricTypeCounter, c.GetType())
	assert.Equal(t, int64(10), c.GetValue())
}

// TestMetric_InterfaceSatisfaction фиксирует, что оба типа метрик
// реализуют интерфейс entity.Metric, и тип определяется корректно.
func TestMetric_InterfaceSatisfaction(t *testing.T) {
	tests := []struct {
		name     string
		metric   entity.Metric
		wantType entity.MetricType
		wantName string
	}{
		{
			name:     "gauge satisfies Metric",
			metric:   entity.NewGaugeMetric("g", 1.5),
			wantType: entity.MetricTypeGauge,
			wantName: "g",
		},
		{
			name:     "counter satisfies Metric",
			metric:   entity.NewCounterMetric("c", 7),
			wantType: entity.MetricTypeCounter,
			wantName: "c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantType, tt.metric.GetType())
			assert.Equal(t, tt.wantName, tt.metric.GetName())
		})
	}
}

func TestMetricType_Values(t *testing.T) {
	assert.Equal(t, "gauge", string(entity.MetricTypeGauge))
	assert.Equal(t, "counter", string(entity.MetricTypeCounter))
}

// TestSentinelErrors проверяет, что доменные ошибки различимы между собой
// и несут корректные сообщения — на них завязана маршрутизация HTTP-статусов.
func TestSentinelErrors(t *testing.T) {
	errs := []error{
		entity.ErrInvalidMetricType,
		entity.ErrInvalidMetricValue,
		entity.ErrEmptyMetricName,
		entity.ErrMetricNotFound,
	}

	for i := range errs {
		assert.NotEmpty(t, errs[i].Error())
		for j := range errs {
			if i != j {
				assert.NotErrorIs(t, errs[i], errs[j], "sentinel errors must be distinct")
			}
		}
	}
}
