package agent

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"
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
	zap.S().Debugw("collecting metrics")
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
			zap.S().Warnw("unknown metric type", "type", fmt.Sprintf("%T", value), "name", name)
		}
	}
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
