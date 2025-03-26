package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"metralert/internal/metrics"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"go.uber.org/zap"
)

type StorageInterface interface {
	Update(metric metrics.Metrics) (metrics.Metrics, error)
	Read(metric metrics.Metrics) (metrics.Metrics, bool)
	ReadAll() map[string]string
}

type Server struct {
	url     string
	storage StorageInterface
	logger  *zap.SugaredLogger
}

func New(url string, repo StorageInterface, logger *zap.SugaredLogger) Server {
	return Server{
		url:     url,
		storage: repo,
		logger:  logger,
	}
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

func (server *Server) Start() {
	r := chi.NewRouter()

	server.logger.Infow(
		"Starting server",
		"url", server.url)

	r.Use(server.loggingMiddleware)
	r.Use(middleware.Compress(5, "application/json", "text/html"))
	r.Route("/update", func(r chi.Router) {
		r.Post("/{metrictype}/{metricname}/{metricvalue}", server.UpdateHandler)
		r.Post("/", server.UpdateMetricJSONHandler)
	})
	r.Get("/", server.GetMainHandler)
	r.Route("/value", func(r chi.Router) {
		r.Get("/{metrictype}/{metricname}", server.GetMetricHandler)
		r.Post("/", server.ReadMetricJSONHandler)
	})
	if err := http.ListenAndServe(server.url, r); err != nil {
		server.logger.Fatalw(err.Error(), "event", "start server")
	}
}

// Обработчик для вывод всех метрик в html страницу
func (server *Server) GetMainHandler(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.ParseFiles("internal/server/templates/mainpage.html")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Type", "gzip")
	}
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, server.storage.ReadAll())
}

func (server *Server) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	metrictype := chi.URLParam(r, "metrictype")
	metricname := chi.URLParam(r, "metricname")

	metric := metrics.Metrics{
		ID:    metricname,
		MType: metrictype,
	}

	storageMetric, ok := server.storage.Read(metric)
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
		resultMetric, err = server.storage.Update(metric)
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
		resultMetric, err = server.storage.Update(metric)
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

	storageMetric, ok := server.storage.Read(metric)
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

func (server *Server) UpdateMetricJSONHandler(w http.ResponseWriter, r *http.Request) {
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

	resultMetric, err := server.storage.Update(metric)
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
