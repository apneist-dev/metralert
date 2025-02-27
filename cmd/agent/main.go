package main

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"reflect"
	"runtime"
	"sync"
	"time"
)

type gauge float64
type counter int64

var (
	mutex          sync.Mutex
	endpoints      []string = []string{}
	PollInterval   int      = 2
	ReportInterval int      = 10
	PollCount      counter
	rtm            runtime.MemStats
	serverurl      = "http://localhost:8080"
)

func main() {

	// pollInterval := 2
	// e := &endpoints

	go CollectMetric()
	go SendAllMetrics()

	select {}
}

// определяем поля структуры MemStats и создаём слайс url
func CollectMetric() {
	for {
		var RandomValue gauge
		runtime.ReadMemStats(&rtm)

		result := []string{}

		for i, k := range reflect.VisibleFields(reflect.TypeOf(rtm)) {
			value := reflect.ValueOf(rtm).Field(i)
			var endpoint string
			switch {
			case k.Type == reflect.TypeFor[uint64]() || k.Type == reflect.TypeFor[uint32]():
				endpoint = fmt.Sprintf("%s%s%s/%d", serverurl, "/update/gauge/", k.Name, value)
			case k.Type == reflect.TypeFor[float64]():
				endpoint = fmt.Sprintf("%s%s%s/%f", serverurl, "/update/gauge/", k.Name, value)
			}
			if endpoint != "" {
				result = append(result, endpoint)
			}

		}

		RandomValue = gauge(rand.Float64())
		endpointrandom := fmt.Sprintf("%s%s%s/%f", serverurl, "/update/gauge/", "RandomValue", RandomValue)
		result = append(result, endpointrandom)
		endpointpollcounter := fmt.Sprintf("%s%s%s/%d", serverurl, "/update/gauge/", "PollCount", PollCount)
		result = append(result, endpointpollcounter)

		PollCount += counter(1)
		fmt.Println(PollCount)

		time.Sleep(time.Duration(PollInterval) * time.Second)

		mutex.Lock()
		endpoints = result[:]
		mutex.Unlock()
		fmt.Println("Metrics collected")
	}
}

func SendPost(endpoint string) {
	resp, err := http.Post(endpoint, "text/plain", http.NoBody)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(resp.StatusCode)
}

func SendAllMetrics() {
	for {
		mutex.Lock()
		for _, s := range endpoints {
			SendPost(s)
			fmt.Println(s)
		}
		mutex.Unlock()
		time.Sleep(time.Duration(ReportInterval) * time.Second)
		fmt.Println("Metrics sent")
	}
}

// func SendAllMetrics
