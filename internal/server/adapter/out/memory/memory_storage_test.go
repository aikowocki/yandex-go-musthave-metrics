package memory

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricStore(t *testing.T) {
	runStorageTests(t, NewMetricStorage())
}

func TestSyncBackupStorage(t *testing.T) {
	store := NewMetricStorage()
	backup := NewFileBackup(filepath.Join(t.TempDir(), "backup.json"), store, 0)
	storage := NewSyncBackupStorage(store, backup)

	runStorageTests(t, storage)
}

// TestMetricStore_ConcurrentAccess проверяет потокобезопасность MetricStore
// под одновременной нагрузкой. Запускать с -race для обнаружения гонок;
func TestMetricStore_ConcurrentAccess(t *testing.T) {
	store := NewMetricStorage()
	ctx := t.Context()

	const (
		workers     = 50
		opsPerGoro  = 100
		counterName = "hits"
	)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < opsPerGoro; j++ {
				_, _ = store.UpdateCounter(ctx, counterName, 1)
				_, _ = store.UpdateGauge(ctx, "cpu", float64(n))
				_, _ = store.GetCounter(ctx, counterName)
				_, _ = store.GetGauge(ctx, "cpu")
				_, _ = store.GetAllGauges(ctx)
			}
		}(i)
	}
	wg.Wait()

	v, err := store.GetCounter(ctx, counterName)
	require.NoError(t, err)
	assert.Equal(t, int64(workers*opsPerGoro), v)
}

// TestNewMemoryStorage_RestoreFromFile проверяет сборку хранилища с восстановлением
// из существующего файла бэкапа (синхронный режим, interval == 0).
func TestNewMemoryStorage_RestoreFromFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "backup.json")

	// Готовим файл бэкапа с метриками.
	seed := NewMetricStorage()
	_, err := seed.UpdateGauge(ctx, "cpu", 0.9)
	require.NoError(t, err)
	_, err = seed.UpdateCounter(ctx, "hits", 5)
	require.NoError(t, err)
	require.NoError(t, NewFileBackup(path, seed, 0).Save())

	// Поднимаем storage с restore=true, синхронный бэкап (interval=0).
	storage := NewMemoryStorage(ctx, path, true, 0)
	repo := storage.MetricRepo()

	g, err := repo.GetGauge(ctx, "cpu")
	require.NoError(t, err)
	assert.Equal(t, 0.9, g.Value)

	c, err := repo.GetCounter(ctx, "hits")
	require.NoError(t, err)
	assert.Equal(t, int64(5), c.Value)
}

// TestNewMemoryStorage_AsyncBackup проверяет ветку с фоновым бэкапом (interval > 0):
// WaitBackup должен завершиться после отмены контекста.
func TestNewMemoryStorage_AsyncBackup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	path := filepath.Join(t.TempDir(), "async.json")

	storage := NewMemoryStorage(ctx, path, false, 10*time.Millisecond)
	_, err := storage.MetricRepo().GetGauge(ctx, "missing")
	assert.Error(t, err) // storage живой, метрики нет

	// Отменяем — фоновая горутина должна завершиться, WaitBackup разблокируется.
	cancel()
	done := make(chan struct{})
	go func() {
		storage.WaitBackup(context.Background())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("WaitBackup did not return after context cancel")
	}
}
