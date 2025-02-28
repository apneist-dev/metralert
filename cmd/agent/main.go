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

type Client struct{ url string }

func NewClient(url string) Client { return Client{url} }

func main() {
	client := NewClient(serverurl)
	go CollectMetric()
	go client.SendAllMetrics()

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
			case k.Type == reflect.TypeFor[uint64]():
				endpoint = fmt.Sprintf("%s%s/%d", "/update/gauge/", k.Name, value.Interface().(uint64))
			case k.Type == reflect.TypeFor[uint32]():
				endpoint = fmt.Sprintf("%s%s/%d", "/update/gauge/", k.Name, value.Interface().(uint32))
			case k.Type == reflect.TypeFor[float64]():
				endpoint = fmt.Sprintf("%s%s/%f", "/update/gauge/", k.Name, value.Interface().(float64))
			}
			if endpoint != "" {
				result = append(result, endpoint)
			}
		}

		RandomValue = gauge(rand.Float64())
		endpointrandom := fmt.Sprintf("%s%s/%f", "/update/gauge/", "RandomValue", RandomValue)
		result = append(result, endpointrandom)
		endpointpollcounter := fmt.Sprintf("%s%s/%d", "/update/counter/", "PollCount", PollCount)
		result = append(result, endpointpollcounter)

		PollCount += counter(1)
		// fmt.Println("PollCount:", PollCount)

		time.Sleep(time.Duration(PollInterval) * time.Second)

		mutex.Lock()
		endpoints = result[:]
		mutex.Unlock()
		fmt.Println(len(endpoints), "metrics collected")
	}
}

func (c Client) SendPost(endpoint string) (*http.Response, error) {
	url := serverurl + "/" + endpoint
	// fmt.Println(url)
	resp, err := http.Post(url, "text/plain", http.NoBody)
	if err != nil {
		// fmt.Println(err)
		return resp, err
	}
	defer resp.Body.Close()
	// fmt.Println(resp.StatusCode)
	return resp, nil
}

func (c Client) SendAllMetrics() error {
	for {
		mutex.Lock()
		for _, s := range endpoints {
			resp, err := c.SendPost(s)
			if err != nil {
				return err
			}
			// fmt.Println(s)
			defer resp.Body.Close()
		}
		mutex.Unlock()
		time.Sleep(time.Duration(ReportInterval) * time.Second)
		// fmt.Println("SendAllMetrics finished")
	}
}

// func SendAllMetrics
