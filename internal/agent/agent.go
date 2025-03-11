package agent

import (
	"fmt"
	"log"
	"math/rand/v2"
	"metralert/internal/metrics"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

// type gauge float64
// type counter int64

type Agent struct {
	url            string
	pollInterval   int
	reportInterval int
	pollCount      metrics.Counter
	mutex          sync.Mutex
	endpoints      []string
	rtm            runtime.MemStats
	client         http.Client
}

// Конструктор агента
func New(url string, poll int, report int) Agent {
	return Agent{
		url:            url,
		pollInterval:   poll,
		reportInterval: report,
		pollCount:      metrics.Counter(0),
		mutex:          sync.Mutex{},
		endpoints:      []string{},
		rtm:            runtime.MemStats{},
		client: http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// Сбор метрик MemStats
func (a *Agent) CollectMetric() {
	for {
		var RandomValue metrics.Gauge
		runtime.ReadMemStats(&a.rtm)

		result := []string{}

		for i, k := range reflect.VisibleFields(reflect.TypeOf(a.rtm)) {
			value := reflect.ValueOf(a.rtm).Field(i)
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

		RandomValue = metrics.Gauge(rand.Float64())
		endpointrandom := fmt.Sprintf("%s%s/%f", "/update/gauge/", "RandomValue", RandomValue)
		result = append(result, endpointrandom)

		endpointpollcounter := fmt.Sprintf("%s%s/%d", "/update/counter/", "PollCount", a.pollCount)
		result = append(result, endpointpollcounter)

		a.pollCount += metrics.Counter(1)

		time.Sleep(time.Duration(a.pollInterval) * time.Second)

		a.mutex.Lock()
		a.endpoints = result[:]
		a.mutex.Unlock()

		log.Println(len(a.endpoints), "метрик собрано")
	}
}

// Отправка одного запроса Post
func (a *Agent) SendPost(endpoint string) (*http.Response, error) {
	url := ""
	if !strings.Contains(a.url, "http") {
		url = "http://" + a.url + endpoint
	} else {
		url = a.url + endpoint
	}
	resp, err := a.client.Post(url, "text/plain", http.NoBody)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// Отправка всех метрик
func (a *Agent) SendAllMetrics() error {
	for {
		a.mutex.Lock()
		endpointsCopy := a.endpoints[:]
		a.mutex.Unlock()
		for _, s := range endpointsCopy {
			resp, err := a.SendPost(s)
			if err != nil {
				log.Printf("При отправке метрик произошла ошибка: %v", err)
				return err
			}
			log.Println("Получен ответ", resp.StatusCode)
			resp.Body.Close()
		}
		time.Sleep(time.Duration(a.reportInterval) * time.Second)
	}
}
