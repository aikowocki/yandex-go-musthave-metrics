package metric

import (
	"context"
	"errors"
)

type Storage interface {
	GetGauge(ctx context.Context, name string) (float64, error)
	UpdateGauge(ctx context.Context, name string, value float64) (float64, error)

	GetCounter(ctx context.Context, name string) (int64, error)
	UpdateCounter(ctx context.Context, name string, value int64) (int64, error)

	GetAllGauges(ctx context.Context) (map[string]float64, error)
	GetAllCounters(ctx context.Context) (map[string]int64, error)

	UpdateBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error
}

var (
	ErrNotFound = errors.New("not found in storage")
)

// BatchRestorer Опциональный интерфейс для batch восстановления
type BatchRestorer interface {
	RestoreBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error
}

type BackupableStorage interface {
	Storage
	BatchRestorer
}
