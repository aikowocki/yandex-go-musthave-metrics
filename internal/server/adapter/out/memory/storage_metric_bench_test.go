package memory

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkMetricStore_UpdateGauge(b *testing.B) {
	store := NewMetricStorage()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.UpdateGauge(ctx, "cpu", float64(i))
	}
}

func BenchmarkMetricStore_GetGauge(b *testing.B) {
	store := NewMetricStorage()
	ctx := context.Background()
	_, _ = store.UpdateGauge(ctx, "cpu", 3.14)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetGauge(ctx, "cpu")
	}
}

func BenchmarkMetricStore_UpdateCounter(b *testing.B) {
	store := NewMetricStorage()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.UpdateCounter(ctx, "hits", 1)
	}
}

func BenchmarkMetricStore_GetCounter(b *testing.B) {
	store := NewMetricStorage()
	ctx := context.Background()
	_, _ = store.UpdateCounter(ctx, "hits", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetCounter(ctx, "hits")
	}
}

func BenchmarkMetricStore_UpdateBatch(b *testing.B) {
	store := NewMetricStorage()
	ctx := context.Background()

	gauges := make(map[string]float64, 30)
	counters := make(map[string]int64, 10)
	for i := 0; i < 30; i++ {
		gauges[fmt.Sprintf("gauge_%d", i)] = float64(i)
	}
	for i := 0; i < 10; i++ {
		counters[fmt.Sprintf("counter_%d", i)] = int64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.UpdateBatch(ctx, gauges, counters)
	}
}

func BenchmarkMetricStore_GetAllGauges(b *testing.B) {
	store := NewMetricStorage()
	ctx := context.Background()
	for i := 0; i < 30; i++ {
		_, _ = store.UpdateGauge(ctx, fmt.Sprintf("gauge_%d", i), float64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetAllGauges(ctx)
	}
}

func BenchmarkMetricStore_GetAllCounters(b *testing.B) {
	store := NewMetricStorage()
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		_, _ = store.UpdateCounter(ctx, fmt.Sprintf("counter_%d", i), int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetAllCounters(ctx)
	}
}
