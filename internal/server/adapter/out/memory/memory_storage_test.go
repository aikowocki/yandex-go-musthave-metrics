package memory

import (
	"path/filepath"
	"testing"
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
