// Server represents the main server structure that handles HTTP requests and manages metrics storage.
package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"metralert/internal/metrics"
	"metralert/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"go.uber.org/zap"
)

// Server represents the main server structure that handles HTTP requests and manages metrics storage.
type Server struct {
	storage    storage.StorageInterface
	logger     *zap.SugaredLogger
	HTTPServer *http.Server
	Router     *chi.Mux
	hashKey    string
	AuditCh    chan metrics.AuditMetrics
}

// New creates and configures a new Server instance with the specified address, storage repository,
// hash key for request validation, and logger.
//
// Parameters:
//   - address: The network address (host:port) on which the server will listen
//   - repo: The storage interface implementation for persisting metrics
//   - hashKey: The key used for HMAC hash validation of requests
//   - logger: The structured logger instance for server logging
//
// Returns:
//   - A pointer to the newly created Server instance
func New(address string, repo storage.StorageInterface, hashKey string, logger *zap.SugaredLogger) *Server {
	s := &Server{}
	s.Router = chi.NewRouter()
	s.Router.Use(s.loggingMiddleware, s.verifyHashMiddleware, s.hashMiddleware)

	s.Router.Use(middleware.Compress(5, "application/json", "text/html"))
	s.Router.Get("/ping", s.DatabasePinger)
	s.Router.Route("/update", func(router chi.Router) {
		router.Post("/{metrictype}/{metricname}/{metricvalue}", s.UpdateHandler)
		router.Post("/", s.UpdateMetricJSONHandler)
	})
	s.Router.Get("/", s.GetMainHandler)
	s.Router.Route("/value", func(router chi.Router) {
		router.Get("/{metrictype}/{metricname}", s.GetMetricHandler)
		router.Post("/", s.ReadMetricJSONHandler)
	})
	s.Router.Post("/updates/", s.UpdateBatchMetricsJSONHandler)

	s.Router.Mount("/debug/pprof", http.DefaultServeMux)

	s.storage = repo
	s.logger = logger
	s.hashKey = hashKey

	s.HTTPServer = &http.Server{
		Addr:    address,
		Handler: s.Router,
	}
	s.AuditCh = make(chan metrics.AuditMetrics, 50)

	return s
}

// Start begins listening for and serving HTTP requests on the configured address.
// It logs the server start event and any fatal errors that occur during startup.
func (server *Server) Start() {
	server.logger.Infow(
		"Starting server",
		"url", server.HTTPServer.Addr)

	err := server.HTTPServer.ListenAndServe()
	if err != http.ErrServerClosed {
		server.logger.Fatalw("Unable to start server:", err)
	}
}

// Shutdown gracefully shuts down the server with a 5-second timeout context.
// It logs the shutdown event and any fatal errors that occur during shutdown.
func (server *Server) Shutdown() {
	server.logger.Infow(
		"Shutting down server",
		"url", server.HTTPServer.Addr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := server.HTTPServer.Shutdown(ctx); err != nil {
		server.logger.Fatalw(err.Error(), "event", "shutdown server")
	}
	defer cancel()
}

type (
	// responseData holds HTTP response metadata for logging purposes.
	responseData struct {
		status int
		size   int
	}

	// loggingResponseWriter wraps http.ResponseWriter to capture response metadata.
	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

// Write delegates to the underlying ResponseWriter's Write method and tracks the number of bytes written.
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

// WriteHeader delegates to the underlying ResponseWriter's WriteHeader method and captures the status code.
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

// loggingMiddleware is a middleware function that logs request and response details including
// URI, method, time spent, response size, and response status.
func (server *Server) loggingMiddleware(next http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		response := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   response,
		}

		start := time.Now()
		next.ServeHTTP(&lw, r)
		server.logger.Infow(
			"Request received",
			"URI", r.RequestURI,
			"Method", r.Method,
			"TimeSpent", time.Since(start),
			"ResponseSize", response.size,
			"ResponseStatus", response.status,
		)
	}
	return http.HandlerFunc(logFn)
}

// verifyHashMiddleware is a middleware function that verifies the HMAC SHA256 hash of the request body
// against the "Hash" header. If the hash key is not set or the hashes don't match, it returns a 400 error.
func (server *Server) verifyHashMiddleware(next http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {

		receivedHash := r.Header.Get("Hash")
		server.logger.Info("Headers ", r.Header)
		server.logger.Info("Received hash ", receivedHash)

		if server.hashKey == "" || receivedHash == "" || receivedHash == "none" {
			next.ServeHTTP(w, r)
			return
		}

		// Читаем тело запроса
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Восстанавливаем тело
		r.Body = io.NopCloser(bytes.NewReader(body))

		// Вычисляем хеш
		h := hmac.New(sha256.New, []byte(server.hashKey))
		h.Write(body)
		calculatedHash := hex.EncodeToString(h.Sum(nil))
		server.logger.Info("Calculated hash ", calculatedHash)
		// Сравниваем хеши
		if calculatedHash != receivedHash {
			http.Error(w, "Invalid body hash", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(logFn)
}

// hashMiddleware is a middleware function that calculates the HMAC SHA256 hash of the request body
// and adds it to the response headers as "Hashsha256".
func (server *Server) hashMiddleware(next http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {

		// Читаем тело запроса
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Восстанавливаем тело
		r.Body = io.NopCloser(bytes.NewReader(body))

		// Вычисляем хеш
		h := hmac.New(sha256.New, []byte(server.hashKey))
		h.Write(body)
		calculatedHash := hex.EncodeToString(h.Sum(nil))

		// Пишем хеш
		w.Header().Set("Hashsha256", calculatedHash)

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(logFn)
}

// GetMainHandler handles GET requests to the root path and renders all metrics in an HTML page.
// It retrieves all metrics from storage and executes the mainpage.html template.
func (server *Server) GetMainHandler(w http.ResponseWriter, r *http.Request) {

	allMetrics, err := server.storage.GetMetrics(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("internal/server/templates/mainpage.html")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, allMetrics)
}

// GetMetricHandler handles GET requests to retrieve a specific metric by type and name.
// It returns the metric value as a string in the response body.
func (server *Server) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	metrictype := chi.URLParam(r, "metrictype")
	metricname := chi.URLParam(r, "metricname")

	metric := metrics.Metrics{
		ID:    metricname,
		MType: metrictype,
	}

	storageMetric, ok := server.storage.GetMetricByName(r.Context(), metric)
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if storageMetric.Value != nil {
		fmt.Fprint(w, *storageMetric.Value)
	}
	if storageMetric.Delta != nil {
		fmt.Fprint(w, *storageMetric.Delta)
	}
}

// UpdateHandler handles POST requests to update a single metric via URL parameters.
// It supports both counter (integer) and gauge (float64) metric types.
func (server *Server) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metrictype := chi.URLParam(r, "metrictype")
	metricname := chi.URLParam(r, "metricname")
	metricvalue := chi.URLParam(r, "metricvalue")

	metric := metrics.Metrics{
		ID:    metricname,
		MType: metrictype,
	}

	resultMetric := metrics.Metrics{}

	types := []string{"gauge", "counter"}
	if !slices.Contains(types, metrictype) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	switch metrictype {
	case "counter":
		metricvalueInt64, err := strconv.ParseInt(metricvalue, 10, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		metric.Delta = &metricvalueInt64
		resultMetric, err = server.storage.UpdateMetric(r.Context(), metric)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Принята метрика: (Тип: counter, Имя: %s, Значение: %d)\n", metricname, *resultMetric.Delta)
	case "gauge":
		metricvalueFloat64, err := strconv.ParseFloat(metricvalue, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		metric.Value = &metricvalueFloat64
		resultMetric, err = server.storage.UpdateMetric(r.Context(), metric)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Принята метрика: (Тип: counter, Имя: %s, Значение: %f)\n", metricname, *resultMetric.Value)
	default:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
}

// ReadMetricJSONHandler handles POST requests to retrieve a metric in JSON format.
// It expects a JSON payload with metric ID and type, and returns the full metric object as JSON.
func (server *Server) ReadMetricJSONHandler(w http.ResponseWriter, r *http.Request) {
	var metric metrics.Metrics
	var buf bytes.Buffer

	w.Header().Set("Content-Type", "application/json")
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &metric); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	storageMetric, ok := server.storage.GetMetricByName(r.Context(), metric)
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	resp, err := json.Marshal(storageMetric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// gzipDecompress decompresses a gzipped byte slice and returns the decompressed data.
// It returns an error if decompression fails.
func gzipDecompress(body []byte) ([]byte, error) {
	reader := bytes.NewReader(body)
	gzreader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}

	result, err := io.ReadAll(gzreader)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateMetricJSONHandler handles POST requests to update a single metric via JSON payload.
// It supports both counter and gauge metric types, with optional gzip compression.
func (server *Server) UpdateMetricJSONHandler(w http.ResponseWriter, r *http.Request) {
	var metric metrics.Metrics
	var buf bytes.Buffer

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	body := buf.Bytes()

	if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
		body, err = gzipDecompress(buf.Bytes())
		if err != nil {
			server.logger.Infow("Unable to decompress body")
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}

	if err = json.Unmarshal(body, &metric); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resultMetric, err := server.storage.UpdateMetric(r.Context(), metric)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	resp, err := json.Marshal(resultMetric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// UpdateBatchMetricsJSONHandler handles POST requests to update multiple metrics in a batch via JSON payload.
// It supports optional gzip compression and performs audit logging of the updated metrics.
func (server *Server) UpdateBatchMetricsJSONHandler(w http.ResponseWriter, r *http.Request) {
	var (
		metricsRead []metrics.Metrics
		metricNames []string
		buf         bytes.Buffer
	)

	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	body := buf.Bytes()

	if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
		body, err = gzipDecompress(buf.Bytes())
		if err != nil {
			server.logger.Infow("Unable to decompress body")
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}

	if err = json.Unmarshal(body, &metricsRead); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	server.logger.Infoln(r.RemoteAddr)
	// Audit section ITER16
	metricNames = make([]string, 0, len(metricsRead))
	for _, v := range metricsRead {
		metricNames = append(metricNames, v.ID)
	}

	auditEntry := metrics.AuditMetrics{
		TS:          time.Now().Unix(),
		MetricNames: metricNames,
		IP:          r.RemoteAddr,
	}

	server.AuditCh <- auditEntry

	resultMetrics, err := server.storage.UpdateBatchMetrics(r.Context(), metricsRead)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(resultMetrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// DatabasePinger handles GET requests to check database connectivity.
// It performs a ping operation on the database and returns a success message if the database is accessible.
func (server *Server) DatabasePinger(w http.ResponseWriter, r *http.Request) {
	err := server.storage.PingDatabase(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "database is accessed\n")
}

// AuditLogger runs a background goroutine that processes audit entries from the AuditCh channel.
// It writes audit entries to a file and/or sends them to an audit URL.
// If both auditFile and auditURL are empty, the function returns immediately.
func (server *Server) AuditLogger(auditFile string, auditURL string) {
	var file *os.File
	var err error
	var client = http.Client{}

	if auditFile == "" && auditURL == "" {
		return
	}

	if auditFile != "" {
		file, err = os.OpenFile(auditFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			server.logger.Errorln("Unable to open or create audit file")
			return
		}
		server.logger.Infoln("Audit file created", auditFile)
	}

	for {
		auditEntry := <-server.AuditCh
		data, err := json.MarshalIndent(&auditEntry, "", "  ")
		if err != nil {
			server.logger.Warnln("Unable to marshal metrics to audit", err)
			continue
		}

		_, err = file.Write(data)
		if err != nil {
			server.logger.Warnln("Unable to write to audit file", err)
			continue
		}
		_, err = file.Write([]byte("\n"))
		if err != nil {
			server.logger.Warnln("Unable to write linebraker to audit file", err)
			continue
		}
		if auditURL != "" {
			resp, err := client.Post(auditURL, "json", bytes.NewBuffer(data))
			if err != nil {
				server.logger.Warnln("Unable to write to auditURL", err)
			}
			defer resp.Body.Close()
			server.logger.Infoln(resp)
			resp.Body.Close()
		}
		server.logger.Debugln(auditEntry)
	}
}
