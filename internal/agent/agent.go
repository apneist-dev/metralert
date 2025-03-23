package agent

import (
	"bytes"
	"encoding/json"
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

type Agent struct {
	url              string
	pollInterval     int
	reportInterval   int
	pollCount        metrics.Counter
	mutex            sync.Mutex
	memoryStatistics []metrics.Metrics
	rtm              runtime.MemStats
	client           http.Client
}

// Конструктор агента
func New(url string, poll int, report int) Agent {
	return Agent{
		url:              url,
		pollInterval:     poll,
		reportInterval:   report,
		pollCount:        metrics.Counter(0),
		mutex:            sync.Mutex{},
		memoryStatistics: []metrics.Metrics{},
		rtm:              runtime.MemStats{},
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

		time.Sleep(time.Duration(a.pollInterval) * time.Second)

		a.mutex.Lock()
		a.memoryStatistics = result[:]
		a.mutex.Unlock()

		log.Println(len(a.memoryStatistics), "метрик собрано")
	}
}

// Отправка одного запроса Post
func (a *Agent) SendPost(metric metrics.Metrics) (*http.Response, error) {
	url := ""
	if !strings.Contains(a.url, "http") {
		url = "http://" + a.url + "/update/"
	} else {
		url = a.url + "/update/"
	}
	jsonData, err := json.Marshal(metric)
	if err != nil {
		log.Println("Unable to Marshal metric")
		return nil, err
	}
	resp, err := a.client.Post(url, "text/plain", bytes.NewBuffer(jsonData))
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// Отправка всех метрик
func (a *Agent) SendAllMetrics() error {
	for {
		a.mutex.Lock()
		memoryStatisticsCopy := a.memoryStatistics[:]
		a.mutex.Unlock()
		for {
			resp, err := a.SendPost(metrics.Metrics{})
			resp.Body.Close()
			if err != nil {
				continue
			}
			break
		}
		for _, s := range memoryStatisticsCopy {
			resp, err := a.SendPost(s)
			if err != nil {
				log.Printf("При отправке метрик произошла ошибка: %v", err)
				continue
			}
			log.Println("Получен ответ", resp.StatusCode)
			resp.Body.Close()
		}
		time.Sleep(time.Duration(a.reportInterval) * time.Second)
	}
}
