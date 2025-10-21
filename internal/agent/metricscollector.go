package agent

import (
	"math/rand/v2"
	"metralert/internal/metrics"
	"reflect"
	"runtime"

	"github.com/shirou/gopsutil/v4/mem"
)

// Сбор метрик MemStats
func (a *Agent) CollectRuntimeMetrics() chan []metrics.Metrics {
	out := make(chan []metrics.Metrics, 1)
	go func() {
		for {
			var RandomValue metrics.Gauge
			runtime.ReadMemStats(&a.rtm)

			// result := []string{}
			result := []metrics.Metrics{}

			for i, k := range reflect.VisibleFields(reflect.TypeOf(a.rtm)) {
				value := reflect.ValueOf(a.rtm).Field(i)
				switch {
				case k.Type == reflect.TypeFor[uint64]():
					v := float64(value.Interface().(uint64))
					result = append(result, metrics.Metrics{
						ID:    k.Name,
						MType: "gauge",
						Value: &v,
					})
				case k.Type == reflect.TypeFor[uint32]():
					v := float64(value.Interface().(uint32))
					result = append(result, metrics.Metrics{
						ID:    k.Name,
						MType: "gauge",
						Value: &v,
					})
				case k.Type == reflect.TypeFor[float64]():
					v := float64(value.Interface().(float64))
					result = append(result, metrics.Metrics{
						ID:    k.Name,
						MType: "gauge",
						Value: &v,
					})
				}
			}

			RandomValue = metrics.Gauge(rand.Float64())

			result = append(result, metrics.Metrics{
				ID:    "PollCount",
				MType: "counter",
				Delta: (*int64)(&a.pollCount),
			})

			result = append(result, metrics.Metrics{
				ID:    "RandomValue",
				MType: "gauge",
				Value: (*float64)(&RandomValue),
			})

			a.pollCount += metrics.Counter(1)

			a.logger.Infow("Runtime Metrics collected", "number", len(result))

			out <- result
		}

	}()
	return out
}

func (a *Agent) CollectGopsutilMetrics() chan []metrics.Metrics {
	out := make(chan []metrics.Metrics)
	go func() {
		for {
			v, _ := mem.VirtualMemory()
			result := []metrics.Metrics{}

			valueTotalMemory := float64(v.Total)
			result = append(result, metrics.Metrics{
				ID:    "TotalMemory",
				MType: "gauge",
				Value: &valueTotalMemory,
			})
			valueFreeMemory := float64(v.Total)
			result = append(result, metrics.Metrics{
				ID:    "FreeMemory",
				MType: "gauge",
				Value: &valueFreeMemory,
			})
			valueCPUutilization1 := float64(runtime.GOMAXPROCS(0))
			result = append(result, metrics.Metrics{
				ID:    "CPUutilization1",
				MType: "gauge",
				Value: &valueCPUutilization1,
			})

			a.logger.Infow("Gopsutil Metrics collected", "number", len(result))
			out <- result
		}

	}()
	return out
}
