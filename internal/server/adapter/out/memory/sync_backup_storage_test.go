package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSyncBackupStorage_RestoreBatch проверяет ветку восстановления, которой
// нет в общем контракте (RestoreBatch не входит в MetricStorage).
func TestSyncBackupStorage_RestoreBatch(t *testing.T) {
	store := NewMetricStorage()
	backup := NewFileBackup(filepath.Join(t.TempDir(), "backup.json"), store, 0)
	storage := NewSyncBackupStorage(store, backup)

	// NewSyncBackupStorage возвращает MetricStorage; RestoreBatch — расширение.
	restorer, ok := storage.(interface {
		RestoreBatch(context.Context, map[string]float64, map[string]int64) error
	})
	require.True(t, ok, "SyncBackupStorage must expose RestoreBatch")

	ctx := context.Background()
	err := restorer.RestoreBatch(ctx,
		map[string]float64{"cpu": 0.7},
		map[string]int64{"hits": 42},
	)
	require.NoError(t, err)

	// Восстановленные значения должны читаться обратно.
	g, err := storage.GetGauge(ctx, "cpu")
	require.NoError(t, err)
	assert.Equal(t, 0.7, g)

	c, err := storage.GetCounter(ctx, "hits")
	require.NoError(t, err)
	assert.Equal(t, int64(42), c)
}

// TestSyncBackupStorage_BackupFailureDoesNotFailUpdate проверяет, что ошибка
// записи бэкапа логируется, но не приводит к ошибке самой операции обновления —
// данные в памяти должны обновиться независимо от судьбы файла.
func TestSyncBackupStorage_BackupFailureDoesNotFailUpdate(t *testing.T) {
	ctx := context.Background()

	// Путь к бэкапу «внутри» обычного файла → Save всегда вернёт ошибку MkdirAll.
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	badPath := filepath.Join(blocker, "backup.json")

	store := NewMetricStorage()
	backup := NewFileBackup(badPath, store, 0)
	storage := NewSyncBackupStorage(store, backup)

	// UpdateGauge: значение записано, ошибки нет.
	g, err := storage.UpdateGauge(ctx, "cpu", 0.5)
	require.NoError(t, err)
	assert.Equal(t, 0.5, g)

	// UpdateCounter: накопление работает, ошибки нет.
	c, err := storage.UpdateCounter(ctx, "hits", 10)
	require.NoError(t, err)
	assert.Equal(t, int64(10), c)

	// UpdateBatch: тоже не падает при ошибке бэкапа.
	err = storage.UpdateBatch(ctx,
		map[string]float64{"mem": 1.0},
		map[string]int64{"req": 3},
	)
	require.NoError(t, err)

	// Данные реально в памяти.
	got, err := storage.GetGauge(ctx, "cpu")
	require.NoError(t, err)
	assert.Equal(t, 0.5, got)
}
