package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
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

type Server struct {
	storage    storage.StorageInterface
	logger     *zap.SugaredLogger
	HTTPServer *http.Server
	Router     *chi.Mux
}

func New(address string, repo storage.StorageInterface, logger *zap.SugaredLogger) *Server {
	s := &Server{}
	s.Router = chi.NewRouter()
	s.Router.Use(s.loggingMiddleware)

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

	s.storage = repo
	s.logger = logger

	s.HTTPServer = &http.Server{
		Addr:    address,
		Handler: s.Router,
	}

	return s
}

func (server *Server) Start() {
	server.logger.Infow(
		"Starting server",
		"url", server.HTTPServer.Addr)

	err := server.HTTPServer.ListenAndServe()
	if err != http.ErrServerClosed {
		server.logger.Fatalw("Unable to start server:", err)
	}
}

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
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

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

// Обработчик для вывод всех метрик в html страницу
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

// Обработчик для записи одной метрики в хранилище
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

func (server *Server) ReadMetricJSONHandler(w http.ResponseWriter, r *http.Request) {
	var metric metrics.Metrics
	var buf bytes.Buffer

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

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

func (server *Server) UpdateBatchMetricsJSONHandler(w http.ResponseWriter, r *http.Request) {
	var metrics []metrics.Metrics
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

	if err = json.Unmarshal(body, &metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resultMetrics, err := server.storage.UpdateBatchMetrics(r.Context(), metrics)
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

func (server *Server) DatabasePinger(w http.ResponseWriter, r *http.Request) {
	err := server.storage.PingDatabase(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "database is accessed\n")
}
