package agent

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"math/rand/v2"
	"metralert/internal/metrics"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
)

const (
	updatePath      = "/update/"
	batchUpdatePath = "/updates/"
)

type Agent struct {
	BaseURL          string
	pollInterval     int
	reportInterval   int
	pollCount        metrics.Counter
	mutex            sync.Mutex
	memoryStatistics []metrics.Metrics
	rtm              runtime.MemStats
	client           http.Client
	logger           *zap.SugaredLogger
	batch            bool
	hashKey          string
}

// Конструктор агента
func New(address string, pollInterval int, reportInterval int, hashKey string, logger *zap.SugaredLogger, batch bool) *Agent {
	if !strings.Contains(address, "http") {
		address = "http://" + address
	}
	destinationAddress, err := url.ParseRequestURI(address)
	if err != nil {
		logger.Fatalw("Invalid URL:", err)
	}
	transport := &http.Transport{
		DisableCompression: false,
	}

	retryClient := *retryablehttp.NewClient()
	retryClient.HTTPClient.Transport = transport
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1
	retryClient.RetryWaitMax = 5
	retryClient.Backoff = retryablehttp.LinearJitterBackoff
	retryClient.Logger = nil

	standardClient := *retryClient.StandardClient()

	return &Agent{
		BaseURL:          destinationAddress.String(),
		pollInterval:     pollInterval,
		reportInterval:   reportInterval,
		pollCount:        metrics.Counter(0),
		mutex:            sync.Mutex{},
		memoryStatistics: []metrics.Metrics{},
		rtm:              runtime.MemStats{},
		client:           standardClient,
		logger:           logger,
		batch:            batch,
		hashKey:          hashKey,
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
		a.memoryStatistics = make([]metrics.Metrics, len(result))
		copy(a.memoryStatistics, result)
		a.mutex.Unlock()

		a.logger.Infow("Metrics collected", "number", len(a.memoryStatistics))

	}
}

func gzipCompress(body []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	_, err := gz.Write(body)
	if err != nil {
		return nil, err
	}

	err1 := gz.Close()
	if err1 != nil {
		return nil, err1
	}
	return buf.Bytes(), nil
}

// Отправка одного запроса Post
func (a *Agent) SendPost(metric metrics.Metrics) (*http.Response, error) {
	endpoint := a.BaseURL + updatePath
	jsonData, err := json.Marshal(metric)
	if err != nil {
		log.Println("Unable to Marshal metric")
		return nil, err
	}

	compressedBody, err := gzipCompress(jsonData)
	if err != nil {
		a.logger.Fatalw("Unable to compress body")
	}

	compressedBodyReader := bytes.NewReader(compressedBody)

	req, err := http.NewRequest("POST", endpoint, compressedBodyReader)
	if err != nil {
		a.logger.Fatalw("Unable to form request")
	}

	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Add("Content-Type", "application/json")

	if a.hashKey != "" {
		buf, err := io.ReadAll(bytes.NewReader(compressedBody))
		if err != nil {
			a.logger.Warnf("read body error: %w", err)
		}

		h := hmac.New(sha256.New, []byte(a.hashKey))
		h.Write(buf)
		req.Header.Add("HashSHA256", hex.EncodeToString(h.Sum(nil)))
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	return resp, nil
}

// Отправка всех метрик
func (a *Agent) SendAllMetrics() error {
	a.logger.Infow("Waiting for server")
	for {
		resp, err := a.client.Get(a.BaseURL)
		if err != nil {
			continue
		}
		resp.Body.Close()
		break
	}
	a.logger.Infow("Server is reachable")
	for {
		a.mutex.Lock()
		memoryStatisticsCopy := make([]metrics.Metrics, len(a.memoryStatistics))
		copy(memoryStatisticsCopy, a.memoryStatistics)
		a.mutex.Unlock()
		// batch mode
		if a.batch {
			endpoint := a.BaseURL + batchUpdatePath
			if len(memoryStatisticsCopy) == 0 {
				continue
			}
			jsonData, err := json.Marshal(memoryStatisticsCopy)
			if err != nil {
				a.logger.Fatalw("unable to marshal []metric")
			}

			compressedBody, err := gzipCompress(jsonData)
			if err != nil {
				a.logger.Fatalw("Unable to compress body")
			}

			compressedBodyReader := bytes.NewReader(compressedBody)

			req, err := http.NewRequest("POST", endpoint, compressedBodyReader)
			if err != nil {
				a.logger.Fatalw("Unable to form request")
			}

			req.Header.Set("Content-Encoding", "gzip")
			req.Header.Add("Content-Type", "application/json")

			if a.hashKey != "" {
				buf, err := io.ReadAll(bytes.NewReader(compressedBody))
				if err != nil {
					a.logger.Warnf("read body error: %w", err)
				}

				h := hmac.New(sha256.New, []byte(a.hashKey))
				h.Write(buf)
				req.Header.Add("HashSHA256", hex.EncodeToString(h.Sum(nil)))
			}

			resp, err := a.client.Do(req)
			if err != nil {
				return err
			}
			a.logger.Infow("Batch Metrics sent successfully")
			defer resp.Body.Close()
		}

		// single metric mode
		if !a.batch {
			for _, s := range memoryStatisticsCopy {
				resp, err := a.SendPost(s)
				if err != nil {
					log.Printf("При отправке метрик произошла ошибка: %v", err)
					continue
				}
				a.logger.Infow("Response received",
					"status", resp.StatusCode,
					"Content-Type", resp.Header.Get("Content-Type"),
					"Content-Encoding", resp.Header.Get("Content-Encoding"))
				resp.Body.Close()
			}
		}
		time.Sleep(time.Duration(a.reportInterval) * time.Second)
	}
}
