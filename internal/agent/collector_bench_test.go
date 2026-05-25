package agent

import "testing"

func BenchmarkCollectMetrics(b *testing.B) {
	storage := NewLocalStorage()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CollectMetrics(storage)
	}
}

func BenchmarkCollectBatch(b *testing.B) {
	storage := NewLocalStorage()
	CollectMetrics(storage)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CollectBatch(storage)
		// Восстанавливаем данные, т.к. SnapshotCounters очищает counters
		storage.AddCounter("PollCount", 1)
	}
}
