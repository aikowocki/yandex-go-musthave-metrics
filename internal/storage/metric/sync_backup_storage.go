package metric

import "go.uber.org/zap"

type SyncBackupStorage struct {
	storage MetricStorage
	backup  *FileBackup
}

func NewSyncBackupStorage(storage MetricStorage, backup *FileBackup) MetricStorage {
	return &SyncBackupStorage{
		storage: storage,
		backup:  backup,
	}
}

func (s *SyncBackupStorage) GetGauge(name string) (float64, error) {
	return s.storage.GetGauge(name)
}

func (s *SyncBackupStorage) UpdateGauge(name string, value float64) (float64, error) {
	value, err := s.storage.UpdateGauge(name, value)
	if err == nil {
		saveErr := s.backup.Save()
		if saveErr != nil {
			zap.S().Errorw("backup update error", "Error", saveErr)
		}
	}
	return value, err
}

func (s *SyncBackupStorage) GetCounter(name string) (int64, error) {
	return s.storage.GetCounter(name)

}
func (s *SyncBackupStorage) UpdateCounter(name string, value int64) (int64, error) {
	value, err := s.storage.UpdateCounter(name, value)
	if err == nil {
		saveErr := s.backup.Save()
		if saveErr != nil {
			zap.S().Errorw("backup update error", "Error", saveErr)
		}
	}
	return value, err
}

func (s *SyncBackupStorage) GetAllGauges() (map[string]float64, error) {
	return s.storage.GetAllGauges()

}
func (s *SyncBackupStorage) GetAllCounters() (map[string]int64, error) {
	return s.storage.GetAllCounters()
}

func (s *SyncBackupStorage) RestoreBatch(gauges map[string]float64, counters map[string]int64) error {
	return s.storage.RestoreBatch(gauges, counters)
}
