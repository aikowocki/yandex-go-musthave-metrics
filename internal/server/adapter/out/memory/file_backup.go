package memory

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/api"
	"go.uber.org/zap"
)

// FileBackup управляет сохранением метрик в файл
type FileBackup struct {
	path     string            // Путь к файлу бэкапа
	storage  BackupableStorage // Хранилище метрик
	interval time.Duration     // Интервал автосохранения (0 = отключено)
}

func NewFileBackup(path string, storage BackupableStorage, interval time.Duration) *FileBackup {
	return &FileBackup{
		path:     path,
		storage:  storage,
		interval: interval,
	}
}

func (fb *FileBackup) Save() error {
	ctx := context.Background()
	counters, err := fb.storage.GetAllCounters(ctx)
	if err != nil {
		return err
	}
	gauges, err := fb.storage.GetAllGauges(ctx)
	if err != nil {
		return err
	}
	var dtos []api.MetricDTO
	for name, value := range counters {
		v := value
		dtos = append(dtos, api.MetricDTO{
			ID:    name,
			MType: api.MetricTypeCounter,
			Delta: &v,
		})
	}

	for name, value := range gauges {
		v := value
		dtos = append(dtos, api.MetricDTO{
			ID:    name,
			MType: api.MetricTypeGauge,
			Value: &v,
		})
	}

	data, err := json.Marshal(dtos)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(fb.path), 0755); err != nil {
		return err
	}
	return os.WriteFile(fb.path, data, 0644)
}

func (fb *FileBackup) Restore() error {
	ctx := context.Background()
	data, err := os.ReadFile(fb.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var dtos []api.MetricDTO
	err = json.Unmarshal(data, &dtos)
	if err != nil {
		return err
	}

	gauges := make(map[string]float64)
	counters := make(map[string]int64)
	for _, dto := range dtos {
		switch dto.MType {
		case api.MetricTypeGauge:
			if dto.Value == nil {
				zap.S().Warnw("skipping gauge with nil value", "id", dto.ID)
				continue
			}
			gauges[dto.ID] = *dto.Value
		case api.MetricTypeCounter:
			if dto.Delta == nil {
				zap.S().Warnw("skipping counter with nil delta", "id", dto.ID)
				continue
			}
			counters[dto.ID] = *dto.Delta
		}
	}
	return fb.storage.RestoreBatch(ctx, gauges, counters)

}

// Start запускает фоновое автосохранение по тикеру при interval > 0.
// Возвращает канал, который закрывается после выхода горутины — то есть
// после выполнения финального Save по отмене ctx. При interval <= 0
// возвращается уже закрытый канал.
func (fb *FileBackup) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	if fb.interval <= 0 {
		close(done)
		return done
	}
	go func() {
		defer close(done)
		ticker := time.NewTicker(fb.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := fb.Save()
				if err != nil {
					zap.S().Errorw("backup save error", zap.Error(err))

				}

			case <-ctx.Done():
				err := fb.Save()
				if err != nil {
					zap.S().Errorw("shutdown backup save error", zap.Error(err))
				}
				return
			}
		}
	}()
	return done
}
