package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	config "metralert/config/agent"
	"metralert/internal/metrics"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	pb "metralert/internal/proto"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	updatePath      = "/update/"
	batchUpdatePath = "/updates/"
	metricsMax      = 50
)

// Agent представляет агент для сбора и отправки метрик.
type Agent struct {
	GRPCPort string
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
	LocalAddress  string
	AgentGRPC     bool
}

// New создает новый экземпляр Agent.
// address - адрес сервера для отправки метрик.
// pollInterval - интервал опроса метрик в секундах.
// reportInterval - интервал отправки метрик в секундах.
// hashKey - ключ для вычисления хеша тела запроса.
// logger - логгер для записи логов.
// batch - флаг, указывающий, использовать ли пакетную отправку метрик.
func New(cfg config.Config) *Agent {
	if !strings.Contains(cfg.ServerAddress, "http") {
		cfg.ServerAddress = "http://" + cfg.ServerAddress
	}
	destinationAddress, err := url.ParseRequestURI(cfg.ServerAddress)
	if err != nil {
		cfg.Logger.Fatalw("Invalid URL:", err)
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

	localAddress, err := FindOutIP(cfg.ServerAddress)
	if err != nil {
		cfg.Logger.Fatalln("unable to find local ip: ", err)
	}

	workerChanIn := make(chan metrics.Metrics, metricsMax)
	workerChanOut := make(chan struct {
		response *http.Response
		err      error
	}, metricsMax)

	return &Agent{
		GRPCPort:         cfg.GRPCPort,
		BaseURL:          destinationAddress.String(),
		pollInterval:     cfg.PollInterval,
		reportInterval:   cfg.ReportInterval,
		pollCount:        metrics.Counter(0),
		mutex:            sync.Mutex{},
		memoryStatistics: []metrics.Metrics{},
		rtm:              runtime.MemStats{},
		client:           standardClient,
		logger:           cfg.Logger,
		batch:            cfg.Batch,
		hashKey:          cfg.HashKey,
		WorkerChanIn:     workerChanIn,
		WorkerChanOut:    workerChanOut,
		PublicKeyPath:    cfg.CryptoKey,
		LocalAddress:     localAddress,
		AgentGRPC:        cfg.AgentGRPC,
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

		req.Header.Add("X-Real-IP", a.LocalAddress)

		if a.hashKey != "" {
			h := hmac.New(sha256.New, []byte(a.hashKey))
			h.Write(compressedBody)
			hash := hex.EncodeToString(h.Sum(nil))
			req.Header.Add("Hash", hash)
			a.logger.Infof("Hash is %s", hash)
		}

		a.logger.Infoln("req: ", req)

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

			req.Header.Add("X-Real-IP", a.LocalAddress)

			if a.hashKey != "" {
				buf, err := io.ReadAll(bytes.NewReader(compressedBody))
				if err != nil {
					a.logger.Warnf("read body error: %w", err)
				}

				h := hmac.New(sha256.New, []byte(a.hashKey))
				h.Write(buf)
				req.Header.Add("Hash", hex.EncodeToString(h.Sum(nil)))
			}

			a.logger.Infoln("req: ", req.Header)

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

	SendMetricsGRPC := func(ctx context.Context, c pb.MetricsClient) error {
		metrics := make([]*pb.Metric, 0, len(memoryStatistics))
		for _, m := range memoryStatistics {

			var metric pb.Metric

			if m.Delta != nil {
				metric = *pb.Metric_builder{
					Id:    m.ID,
					Type:  *pb.Metric_COUNTER.Enum(),
					Delta: *m.Delta,
				}.Build()
			}
			if m.Value != nil {
				metric = *pb.Metric_builder{
					Id:    m.ID,
					Type:  *pb.Metric_GAUGE.Enum(),
					Value: *m.Value,
				}.Build()

			}
			metrics = append(metrics, &metric)
		}

		if a.LocalAddress != "" {
			md := metadata.New(map[string]string{"x-real-ip": a.LocalAddress})
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		response, err := c.UpdateMetrics(ctx, pb.UpdateMetricsRequest_builder{
			Metrics: metrics,
		}.Build())
		if err != nil {
			return fmt.Errorf("unable to update metrics: %s", err)
		}
		a.logger.Infoln("response received", response.String())
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
			if a.AgentGRPC {
				conn, err := grpc.NewClient(a.GRPCPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					a.logger.Infoln("error, while connecting to server", "error", err)
					continue
				}
				defer conn.Close()
				c := pb.NewMetricsClient(conn)

				err = SendMetricsGRPC(ctx, c)
				if err != nil {
					a.logger.Infoln("error while sendind metrics", "error", err)
					continue
				}
				continue
			}
			err := SendMetrics()
			if err != nil {
				return err
			}
		}
	}
}

func FindOutIP(serverAddress string) (string, error) {
	if strings.Contains(serverAddress, "http://") {
		serverAddress = strings.TrimPrefix(serverAddress, "http://")
	}
	serverAddress = strings.TrimRight(serverAddress, ":")

	c, err := net.Dial("udp4", serverAddress)
	if err != nil {
		return "", err
	}
	defer c.Close()

	addr := c.LocalAddr().(*net.UDPAddr).IP.String()
	return addr, nil
}
