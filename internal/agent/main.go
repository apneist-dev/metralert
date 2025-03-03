package agent

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"reflect"
	"runtime"
	"strings"
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
)

type Client struct {
	url            string
	pollInterval   int
	reportInterval int
}

// Конструктор агента
func NewClient(url string, pollInterval int, reportInterval int) Client {
	return Client{url,
		pollInterval,
		reportInterval}
}

// Сбор метрик MemStats
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

		time.Sleep(time.Duration(PollInterval) * time.Second)

		mutex.Lock()
		endpoints = result[:]
		mutex.Unlock()

		log.Println(len(endpoints), "метрик собрано")
	}
}

// Отправка одного запроса Post
func (c Client) SendPost(endpoint string) (*http.Response, error) {
	url := ""
	if !strings.Contains(c.url, "http") {
		url = "http://" + c.url + endpoint
	} else {
		url = c.url + endpoint
	}
	resp, err := http.Post(url, "text/plain", http.NoBody)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()
	return resp, nil
}

// Отправка всех метрик
func (c Client) SendAllMetrics() error {
	for {
		mutex.Lock()
		for _, s := range endpoints {
			resp, err := c.SendPost(s)
			if err != nil {
				return err
			}
			log.Println("Получен ответ", resp.StatusCode)
			defer resp.Body.Close()
		}
		mutex.Unlock()
		time.Sleep(time.Duration(ReportInterval) * time.Second)
	}
}
