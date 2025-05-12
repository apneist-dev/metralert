package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"metralert/internal/metrics"
	"net/http"
	"net/url"
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
	metricsMax      = 50
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
	WorkerChanIn     chan metrics.Metrics
	WorkerChanOut    chan struct {
		response *http.Response
		err      error
	}
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

	workerChanIn := make(chan metrics.Metrics, metricsMax)
	workerChanOut := make(chan struct {
		response *http.Response
		err      error
	}, metricsMax)

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
		WorkerChanIn:     workerChanIn,
		WorkerChanOut:    workerChanOut,
	}
}

func (a *Agent) StartSendPostWorkers(numWorkers int) {
	for w := 1; w <= numWorkers; w++ {
		go a.SendPostWorker(w, a.WorkerChanIn, a.WorkerChanOut)
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

// Воркеры
func (a *Agent) SendPostWorker(id int, jobs chan metrics.Metrics, results chan struct {
	response *http.Response
	err      error
}) {
	for metric := range jobs {
		a.logger.Infoln("Worker ", id, " is working on ", metric)
		endpoint := a.BaseURL + updatePath
		jsonData, err := json.Marshal(metric)
		if err != nil {
			a.logger.Warnln("Unable to Marshal metric")
			results <- struct {
				response *http.Response
				err      error
			}{nil, err}
			continue
		}

		compressedBody, err := gzipCompress(jsonData)
		if err != nil {
			a.logger.Warnln("Unable to compress body")
			continue
		}

		compressedBodyReader := bytes.NewReader(compressedBody)

		req, err := http.NewRequest("POST", endpoint, compressedBodyReader)
		if err != nil {
			a.logger.Fatalw("Unable to form request")
		}

		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Add("Content-Type", "application/json")

		if a.hashKey != "" {
			h := hmac.New(sha256.New, []byte(a.hashKey))
			h.Write(compressedBody)
			hash := hex.EncodeToString(h.Sum(nil))
			req.Header.Add("Hash", hash)
			a.logger.Infof("Hash is %s", hash)
		}

		resp, err := a.client.Do(req)
		if err != nil {
			results <- struct {
				response *http.Response
				err      error
			}{resp, err}
			continue
		}

		results <- struct {
			response *http.Response
			err      error
		}{resp, err}

		resp.Body.Close()
	}
}

// Отправка всех метрик
func (a *Agent) SendAllMetrics(ctx context.Context, memIn chan []metrics.Metrics, gopsIn chan []metrics.Metrics, workerIn chan metrics.Metrics, workerOut chan struct {
	response *http.Response
	err      error
}) error {
	memoryStatistics := make([]metrics.Metrics, 0)

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
	// горутина поддерживает pollinterval
	pollTicker := time.NewTicker(time.Duration(a.pollInterval) * time.Second)
	reportTicker := time.NewTicker(time.Duration(a.reportInterval) * time.Second)
	go func(memoryStatistics *[]metrics.Metrics) {
		for {
			select {
			case <-ctx.Done():
				return
			case <-pollTicker.C:
				memoryMetrics := make([]metrics.Metrics, 0)
				runtimeMetrics := <-memIn
				gopsutilMetrics := <-gopsIn
				memoryMetrics = append(memoryMetrics, runtimeMetrics...)
				memoryMetrics = append(memoryMetrics, gopsutilMetrics...)
				*memoryStatistics = memoryMetrics
			}
		}
	}(&memoryStatistics)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-reportTicker.C:
			// batch mode
			if a.batch {
				endpoint := a.BaseURL + batchUpdatePath
				if len(memoryStatistics) == 0 {
					continue
				}
				jsonData, err := json.Marshal(memoryStatistics)
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
					req.Header.Add("Hash", hex.EncodeToString(h.Sum(nil)))
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
				for _, s := range memoryStatistics {
					workerIn <- s
					response := <-workerOut

					if response.err != nil {
						log.Printf("При отправке метрик произошла ошибка: %v", response.err)
						continue
					}
					a.logger.Infow("Response received",
						"status", response.response.StatusCode,
						"Content-Type", response.response.Header.Get("Content-Type"),
						"Content-Encoding", response.response.Header.Get("Content-Encoding"))
					response.response.Body.Close()
				}
			}
		}
	}
}
