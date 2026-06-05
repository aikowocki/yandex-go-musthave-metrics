package memory

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

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
	ctx := context.Background()

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
