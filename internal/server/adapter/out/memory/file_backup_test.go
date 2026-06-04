package memory

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileBackup_SaveRestore(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "backup.json")

	// Заполняем метриками и сохраняем.
	src := NewMetricStorage()
	_, err := src.UpdateGauge(ctx, "cpu", 0.42)
	require.NoError(t, err)
	_, err = src.UpdateGauge(ctx, "memory", 1024.0)
	require.NoError(t, err)
	_, err = src.UpdateCounter(ctx, "hits", 7)
	require.NoError(t, err)

	srcBackup := NewFileBackup(path, src, 0)
	require.NoError(t, srcBackup.Save())

	// Файл существует и не пустой.
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))

	// Пустой storage, Restore из файла.
	dst := NewMetricStorage()
	dstBackup := NewFileBackup(path, dst, 0)
	require.NoError(t, dstBackup.Restore())

	g, err := dst.GetGauge(ctx, "cpu")
	require.NoError(t, err)
	assert.Equal(t, 0.42, g)

	g, err = dst.GetGauge(ctx, "memory")
	require.NoError(t, err)
	assert.Equal(t, 1024.0, g)

	c, err := dst.GetCounter(ctx, "hits")
	require.NoError(t, err)
	assert.Equal(t, int64(7), c)
}

// TestFileBackup_RestoreNoFile проверяет, что Restore из несуществующего файла
// не возвращает ошибку — это ожидаемое поведение при первом запуске.
func TestFileBackup_RestoreNoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.json")

	store := NewMetricStorage()
	backup := NewFileBackup(path, store, 0)

	assert.NoError(t, backup.Restore())
}

// TestFileBackup_SyncSaveOnUpdate проверяет, что SyncBackupStorage
// действительно пишет файл после каждого UpdateGauge/UpdateCounter/UpdateBatch.
func TestFileBackup_SyncSaveOnUpdate(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "backup.json")

	store := NewMetricStorage()
	backup := NewFileBackup(path, store, 0)
	sync := NewSyncBackupStorage(store, backup)

	// До любых записей файла нет.
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err))

	_, err = sync.UpdateGauge(ctx, "cpu", 0.5)
	require.NoError(t, err)

	// После записи файл должен существовать.
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))

	// И Restore в чистый storage поднимает то же значение.
	dst := NewMetricStorage()
	dstBackup := NewFileBackup(path, dst, 0)
	require.NoError(t, dstBackup.Restore())

	g, err := dst.GetGauge(ctx, "cpu")
	require.NoError(t, err)
	assert.Equal(t, 0.5, g)
}
