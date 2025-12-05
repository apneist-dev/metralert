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

// Agent представляет агент для сбора и отправки метрик.
type Agent struct {
	// BaseURL - базовый URL сервера для отправки метрик.
	BaseURL string
	// pollInterval - интервал опроса метрик в секундах.
	pollInterval int
	// reportInterval - интервал отправки метрик в секундах.
	reportInterval int
	// pollCount - счетчик опросов.
	pollCount metrics.Counter
	// mutex - мьютекс для синхронизации доступа к memoryStatistics.
	mutex sync.Mutex
	// memoryStatistics - слайс собранных метрик.
	memoryStatistics []metrics.Metrics
	// rtm - структура для хранения статистики памяти Go runtime.
	rtm runtime.MemStats
	// client - HTTP клиент для отправки запросов.
	client http.Client
	// logger - логгер для записи логов.
	logger *zap.SugaredLogger
	// batch - флаг, указывающий, использовать ли пакетную отправку метрик.
	batch bool
	// hashKey - ключ для вычисления хеша тела запроса.
	hashKey string
	// WorkerChanIn - канал для передачи метрик воркерам.
	WorkerChanIn chan metrics.Metrics
	// WorkerChanOut - канал для получения результатов от воркеров.
	WorkerChanOut chan struct {
		response *http.Response
		err      error
	}
	PublicKeyPath string
}

// New создает новый экземпляр Agent.
// address - адрес сервера для отправки метрик.
// pollInterval - интервал опроса метрик в секундах.
// reportInterval - интервал отправки метрик в секундах.
// hashKey - ключ для вычисления хеша тела запроса.
// logger - логгер для записи логов.
// batch - флаг, указывающий, использовать ли пакетную отправку метрик.
func New(address string, pollInterval int, reportInterval int, hashKey string, logger *zap.SugaredLogger, batch bool, publicKeyPath string) *Agent {
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
		PublicKeyPath:    publicKeyPath,
	}
}

// StartSendPostWorkers запускает заданное количество воркеров для отправки метрик.
// numWorkers - количество воркеров для запуска.
func (a *Agent) StartSendPostWorkers(numWorkers int) {
	for w := 1; w <= numWorkers; w++ {
		go a.SendPostWorker(w, a.WorkerChanIn, a.WorkerChanOut)
	}
}

// gzipCompress сжимает тело запроса с помощью gzip.
// body - тело запроса для сжатия.
// Возвращает сжатое тело запроса и ошибку, если сжатие не удалось.
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

// SendPostWorker - воркер для отправки метрик.
// id - идентификатор воркера.
// jobs - канал для получения метрик.
// results - канал для отправки результатов.
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

// SendAllMetrics отправляет все собранные метрики на сервер.
// ctx - контекст для управления жизненным циклом функции.
// memIn - канал для получения метрик из runtime.
// gopsIn - канал для получения метрик из gopsutil.
// workerIn - канал для передачи метрик воркерам.
// workerOut - канал для получения результатов от воркеров.
// Возвращает ошибку, если отправка метрик не удалась.
func (a *Agent) SendAllMetrics(ctx context.Context, memIn chan []metrics.Metrics, gopsIn chan []metrics.Metrics, workerIn chan metrics.Metrics, workerOut chan struct {
	response *http.Response
	err      error
}) error {
	memoryStatistics := make([]metrics.Metrics, 0)

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

	SendMetrics := func() error {
		if a.batch {
			endpoint := a.BaseURL + batchUpdatePath
			if len(memoryStatistics) == 0 {
				return nil
			}
			jsonData, err := json.Marshal(memoryStatistics)
			if err != nil {
				a.logger.Fatalw("unable to marshal []metric")
			}

			compressedBody, err := gzipCompress(jsonData)
			if err != nil {
				a.logger.Fatalw("Unable to compress body")
			}

			Data := compressedBody

			if a.PublicKeyPath != "" {
				EncrypredData, err := RetrieveEncrypt(Data, a.PublicKeyPath)
				if err != nil {
					return err
				}
				Data = EncrypredData
			}

			compressedBodyReader := bytes.NewReader(Data)

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
				a.logger.Infow("Server unreachable", err)
				return nil
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
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			a.logger.Infoln("Signal received. Shutting down agent.")
			err := SendMetrics()
			if err != nil {
				return err
			}
			a.logger.Infoln("All collected metrics are sent")
			return nil
		case <-reportTicker.C:
			err := SendMetrics()
			if err != nil {
				return err
			}
		}
	}
}
