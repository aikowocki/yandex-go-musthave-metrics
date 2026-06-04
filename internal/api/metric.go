// Package api содержит DTO-структуры для обмена метриками между агентом и сервером через HTTP API.
package api

// MetricDTO — транспортная структура для передачи метрик в JSON-формате.
// Используется как для отправки (agent → server), так и для получения (server → client).
// Для gauge-метрик заполняется поле Value, для counter — поле Delta.
type MetricDTO struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

const (
	// MetricTypeGauge — тип метрики, хранящей текущее значение (перезаписывается при обновлении).
	MetricTypeGauge = "gauge"
	// MetricTypeCounter — тип метрики-счётчика (значение накапливается при обновлении).
	MetricTypeCounter = "counter"
)
