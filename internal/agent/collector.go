package agent

import (
	"fmt"
	"math/rand"
	"runtime"
)

func getRuntimeMetrics() map[string]any {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return map[string]any{
		"Alloc":         memStats.Alloc,
		"BuckHashSys":   memStats.BuckHashSys,
		"Frees":         memStats.Frees,
		"GCCPUFraction": memStats.GCCPUFraction,
		"GCSys":         memStats.GCSys,
		"HeapAlloc":     memStats.HeapAlloc,
		"HeapIdle":      memStats.HeapIdle,
		"HeapInuse":     memStats.HeapInuse,
		"HeapObjects":   memStats.HeapObjects,
		"HeapReleased":  memStats.HeapReleased,
		"HeapSys":       memStats.HeapSys,
		"LastGC":        memStats.LastGC,
		"Lookups":       memStats.Lookups,
		"MCacheInuse":   memStats.MCacheInuse,
		"MCacheSys":     memStats.MCacheSys,
		"MSpanInuse":    memStats.MSpanInuse,
		"MSpanSys":      memStats.MSpanSys,
		"Mallocs":       memStats.Mallocs,
		"NextGC":        memStats.NextGC,
		"NumForcedGC":   memStats.NumForcedGC,
		"NumGC":         memStats.NumGC,
		"OtherSys":      memStats.OtherSys,
		"PauseTotalNs":  memStats.PauseTotalNs,
		"StackInuse":    memStats.StackInuse,
		"StackSys":      memStats.StackSys,
		"Sys":           memStats.Sys,
		"TotalAlloc":    memStats.TotalAlloc,
	}
}
func CollectMetrics(storage MetricStorage) {
	fmt.Println("Collecting metrics...")
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	for name, value := range getRuntimeMetrics() {
		switch v := value.(type) {
		case uint32:
			storage.SetGauge(name, float64(v))
		case uint64:
			storage.SetGauge(name, float64(v))
		case float64:
			storage.SetGauge(name, v)
		default:
			fmt.Printf("Warning: unknown metric type %T for %s\n", value, name)
		}
	}
	storage.AddCounter("PollCount", 1)
	storage.SetGauge("RandomValue", rand.Float64())
}
