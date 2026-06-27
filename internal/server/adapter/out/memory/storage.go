package memory

import (
	"context"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/server/port"
	"go.uber.org/zap"
)

type Storage struct {
	metricRepo *MetricRepo
	backupDone <-chan struct{}
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
		backupDone := backup.Start(ctx)

		if interval == 0 {
			// синхронный бекап - оборачиваем хранилище мтерик
			syncStorage := NewSyncBackupStorage(metricStorage, backup)
			return &Storage{metricRepo: NewMetricRepo(syncStorage), backupDone: backupDone}
		}

		return &Storage{metricRepo: NewMetricRepo(metricStorage), backupDone: backupDone}
	}

	return &Storage{metricRepo: NewMetricRepo(metricStorage)}
}

func (s *Storage) MetricRepo() port.MetricRepository {
	return s.metricRepo
}

// WaitBackup блокируется до завершения фоновой горутины бэкапа (включая её
// финальный Save по отмене ctx) либо до истечения переданного ctx — что
// наступит раньше. Гарантирует, что несохранённые метрики попадут в файл до
// выхода из процесса. Если фонового бэкапа нет, возвращается немедленно.
func (s *Storage) WaitBackup(ctx context.Context) {
	if s.backupDone == nil {
		return
	}
	select {
	case <-s.backupDone:
	case <-ctx.Done():
	}
}
