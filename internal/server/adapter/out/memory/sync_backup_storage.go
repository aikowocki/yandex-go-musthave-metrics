package memory

import (
	"context"

	"go.uber.org/zap"
)

type BackupableStorage interface {
	MetricStorage // уже определён в repo_metric.go
	RestoreBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error
}

type SyncBackupStorage struct {
	storage BackupableStorage
	backup  *FileBackup
}

func NewSyncBackupStorage(storage BackupableStorage, backup *FileBackup) MetricStorage {
	return &SyncBackupStorage{
		storage: storage,
		backup:  backup,
	}
}

func (s *SyncBackupStorage) GetGauge(ctx context.Context, name string) (float64, error) {
	return s.storage.GetGauge(ctx, name)
}

func (s *SyncBackupStorage) UpdateGauge(ctx context.Context, name string, value float64) (float64, error) {
	value, err := s.storage.UpdateGauge(ctx, name, value)
	if err == nil {
		saveErr := s.backup.Save()
		if saveErr != nil {
			zap.S().Errorw("backup update error", "Error", saveErr)
		}
	}
	return value, err
}

func (s *SyncBackupStorage) GetCounter(ctx context.Context, name string) (int64, error) {
	return s.storage.GetCounter(ctx, name)

}
func (s *SyncBackupStorage) UpdateCounter(ctx context.Context, name string, value int64) (int64, error) {
	value, err := s.storage.UpdateCounter(ctx, name, value)
	if err == nil {
		saveErr := s.backup.Save()
		if saveErr != nil {
			zap.S().Errorw("backup update error", "Error", saveErr)
		}
	}
	return value, err
}

func (s *SyncBackupStorage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	return s.storage.GetAllGauges(ctx)

}
func (s *SyncBackupStorage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	return s.storage.GetAllCounters(ctx)
}

func (s *SyncBackupStorage) RestoreBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error {
	return s.storage.RestoreBatch(ctx, gauges, counters)
}

func (s *SyncBackupStorage) UpdateBatch(ctx context.Context, gauges map[string]float64, counters map[string]int64) error {
	err := s.storage.UpdateBatch(ctx, gauges, counters)
	if err == nil {
		saveErr := s.backup.Save()
		if saveErr != nil {
			zap.S().Errorw("backup update error", "Error", saveErr)
		}
	}
	return err
}
