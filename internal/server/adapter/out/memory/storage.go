package memory

import (
	"context"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
	"go.uber.org/zap"
)

type Storage struct {
	metricRepo *MetricRepo
}

func NewMemoryStorage(ctx context.Context, filePath string, restore bool, interval time.Duration) *Storage {
	metricStorage := NewMetricStorage()

	if filePath != "" {
		backup := NewFileBackup(filePath, metricStorage, interval)
		if restore {
			if err := backup.Restore(); err != nil {
				zap.S().Warnw("failed to restore backup", "error", err)
			}
		}
		backup.Start(ctx)

		if interval == 0 {
			// синхронный бекап - оборачиваем хранилище мтерик
			syncStorage := NewSyncBackupStorage(metricStorage, backup)
			return &Storage{metricRepo: NewMetricRepo(syncStorage)}
		}
	}

	return &Storage{metricRepo: NewMetricRepo(metricStorage)}
}

func (s *Storage) MetricRepo() port.MetricRepository {
	return s.metricRepo
}
