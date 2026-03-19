package metric

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

// FileBackup управляет переодическим сохранением метрик в файл
type FileBackup struct {
	path     string            // путь к файлу бэкапа
	storage  BackupableStorage // хранилище метрик
	interval time.Duration     // интервал автосохранения (0 = отключено)
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
	var dtos []model.MetricDTO
	for name, value := range counters {
		dtos = append(dtos, model.MetricDTO{
			ID:    name,
			MType: string(model.MetricTypeCounter),
			Delta: &value,
		})
	}

	for name, value := range gauges {
		dtos = append(dtos, model.MetricDTO{
			ID:    name,
			MType: string(model.MetricTypeGauge),
			Value: &value,
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

	var dtos []model.MetricDTO

	err = json.Unmarshal(data, &dtos)
	if err != nil {
		return err
	}

	gauges := make(map[string]float64)
	counters := make(map[string]int64)
	for _, dto := range dtos {
		switch model.MetricType(dto.MType) {
		case model.MetricTypeGauge:
			if dto.Value == nil {
				zap.S().Warnw("skipping gauge with nil value", "id", dto.ID)
				continue
			}
			gauges[dto.ID] = *dto.Value
		case model.MetricTypeCounter:
			if dto.Delta == nil {
				zap.S().Warnw("skipping counter with nil delta", "id", dto.ID)
				continue
			}
			counters[dto.ID] = *dto.Delta
		}
	}
	return fb.storage.RestoreBatch(ctx, gauges, counters)

}

func (fb *FileBackup) Start(ctx context.Context) {
	if fb.interval > 0 {
		go func() {
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
	}
}
