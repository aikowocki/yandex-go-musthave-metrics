package agent

import (
	"math/rand"
	"runtime"
	"strconv"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"
)

// getRuntimeMetrics: Deprecated: отказался для оптимизации. пишем сразу напрямую в стор
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
	zap.S().Debugw("collecting metrics")
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	storage.SetGauge("Alloc", float64(memStats.Alloc))
	storage.SetGauge("BuckHashSys", float64(memStats.BuckHashSys))
	storage.SetGauge("Frees", float64(memStats.Frees))
	storage.SetGauge("GCCPUFraction", memStats.GCCPUFraction)
	storage.SetGauge("GCSys", float64(memStats.GCSys))
	storage.SetGauge("HeapAlloc", float64(memStats.HeapAlloc))
	storage.SetGauge("HeapIdle", float64(memStats.HeapIdle))
	storage.SetGauge("HeapInuse", float64(memStats.HeapInuse))
	storage.SetGauge("HeapObjects", float64(memStats.HeapObjects))
	storage.SetGauge("HeapReleased", float64(memStats.HeapReleased))
	storage.SetGauge("HeapSys", float64(memStats.HeapSys))
	storage.SetGauge("LastGC", float64(memStats.LastGC))
	storage.SetGauge("Lookups", float64(memStats.Lookups))
	storage.SetGauge("MCacheInuse", float64(memStats.MCacheInuse))
	storage.SetGauge("MCacheSys", float64(memStats.MCacheSys))
	storage.SetGauge("MSpanInuse", float64(memStats.MSpanInuse))
	storage.SetGauge("MSpanSys", float64(memStats.MSpanSys))
	storage.SetGauge("Mallocs", float64(memStats.Mallocs))
	storage.SetGauge("NextGC", float64(memStats.NextGC))
	storage.SetGauge("NumForcedGC", float64(memStats.NumForcedGC))
	storage.SetGauge("NumGC", float64(memStats.NumGC))
	storage.SetGauge("OtherSys", float64(memStats.OtherSys))
	storage.SetGauge("PauseTotalNs", float64(memStats.PauseTotalNs))
	storage.SetGauge("StackInuse", float64(memStats.StackInuse))
	storage.SetGauge("StackSys", float64(memStats.StackSys))
	storage.SetGauge("Sys", float64(memStats.Sys))
	storage.SetGauge("TotalAlloc", float64(memStats.TotalAlloc))

	storage.AddCounter("PollCount", 1)
	storage.SetGauge("RandomValue", rand.Float64())
}

func CollectSystemMetrics(storage MetricStorage) {
	stat, err := mem.VirtualMemory()
	if err != nil {
		zap.S().Warnw("failed to collect memory metrics", zap.Error(err))
	} else {
		storage.SetGauge("TotalMemory", float64(stat.Total))
		storage.SetGauge("FreeMemory", float64(stat.Free))
	}
	percents, err := cpu.Percent(0, true)
	if err != nil {
		zap.S().Warnw("failed to collect cpu metrics", zap.Error(err))
	} else {
		for i, u := range percents {
			storage.SetGauge("CPUUtilization"+strconv.Itoa(i+1), u)
		}
	}
}
