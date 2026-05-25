package agent

import (
	"fmt"
	"testing"
)

func BenchmarkLocalStorage_SetGauge(b *testing.B) {
	storage := NewLocalStorage()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.SetGauge("metric", float64(i))
	}
}

func BenchmarkLocalStorage_AddCounter(b *testing.B) {
	storage := NewLocalStorage()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.AddCounter("metric", 1)
	}
}

func BenchmarkLocalStorage_SnapshotCounters(b *testing.B) {
	storage := NewLocalStorage()
	for i := 0; i < 30; i++ {
		storage.AddCounter(fmt.Sprintf("counter_%d", i), int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.SnapshotCounters()
		// восстанавливаем данные после snapshot (он очищает)
		b.StopTimer()
		for j := 0; j < 30; j++ {
			storage.AddCounter(fmt.Sprintf("counter_%d", j), int64(j))
		}
		b.StartTimer()
	}
}

func BenchmarkLocalStorage_SnapshotGauges(b *testing.B) {
	storage := NewLocalStorage()
	for i := 0; i < 30; i++ {
		storage.SetGauge(fmt.Sprintf("gauge_%d", i), float64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		storage.SnapshotGauges()
	}
}
