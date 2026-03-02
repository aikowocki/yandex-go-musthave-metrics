package metric

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/aikowocki/yandex-go-musthave-metrics/internal/model"
	"go.uber.org/zap"
)

type FileBackup struct {
	path     string
	storage  MetricStorage
	interval time.Duration
}

func NewFileBackup(path string, storage MetricStorage, interval time.Duration) *FileBackup {
	return &FileBackup{
		path:     path,
		storage:  storage,
		interval: interval,
	}
}

func (fb *FileBackup) Save() error {
	counters, err := fb.storage.GetAllCounters()
	if err != nil {
		return err
	}
	gauges, err := fb.storage.GetAllGauges()
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
			gauges[dto.ID] = *dto.Value
		case model.MetricTypeCounter:
			counters[dto.ID] = *dto.Delta
		}
	}
	return fb.storage.RestoreBatch(gauges, counters)

}

func (fb *FileBackup) Start() {
	if fb.interval > 0 {
		go func() {
			ticker := time.NewTicker(fb.interval)
			defer ticker.Stop()
			for range ticker.C {
				err := fb.Save()
				if err != nil {
					zap.S().Errorf("backup save error: %v", err)
				}
			}
		}()
	}

}
