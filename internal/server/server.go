package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"

	"metralert/internal/metrics"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type StorageInterface interface {
	UpdateGauge(string, metrics.Gauge)
	UpdateCounter(string, metrics.Counter)
	ReadGauge(string) (metrics.Gauge, bool)
	ReadCounter(string) (metrics.Counter, bool)
	ReadAllGauge() map[string]metrics.Gauge
	ReadAllCounter() map[string]metrics.Counter
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
			"ResponseStaus", response.status,
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
	r.Route("/update", func(r chi.Router) {
		r.Post("/{metrictype}/{metricname}/{metricvalue}", server.UpdateHandler)
	})
	r.Get("/", server.GetMainHandler)
	r.Route("/value", func(r chi.Router) {
		r.Get("/{metrictype}/{metricname}", server.GetMetricHandler)
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

	tmpl.Execute(w, server.storage.ReadAllGauge())
	tmpl.Execute(w, server.storage.ReadAllCounter())
}

// Обработчик для записи одной метрики в хранилище
func (server *Server) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metrictype := chi.URLParam(r, "metrictype")
	metricname := chi.URLParam(r, "metricname")
	metricvalue := chi.URLParam(r, "metricvalue")
	types := []string{"gauge", "counter"}
	if !slices.Contains(types, metrictype) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	switch metrictype {
	case "counter":
		server.SaveCounterMetric(w, metricname, metricvalue)
	case "gauge":
		server.SaveGaugeMetric(w, metricname, metricvalue)
	default:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}
}

// Обработчик для получения значения одной метрики
func (server *Server) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	metrictype := chi.URLParam(r, "metrictype")
	metricname := chi.URLParam(r, "metricname")
	switch metrictype {
	case "counter":
		metricvalue, ok := server.storage.ReadCounter(metricname)
		if !ok {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		fmt.Fprint(w, metricvalue)
	case "gauge":
		metricvalue, ok := server.storage.ReadGauge(metricname)
		if !ok {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		fmt.Fprint(w, metricvalue)
	default:
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
}

// Хелпер для сохранение Counter метрики в хранилище
func (server *Server) SaveCounterMetric(w http.ResponseWriter, metricname string, metricvalue string) {
	value, err := strconv.Atoi(metricvalue)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	var counterMetricValue = metrics.Counter(value)
	server.storage.UpdateCounter(metricname, counterMetricValue)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Принята метрика: (Тип: counter, Имя: %s, Значение: %d)\n", metricname, counterMetricValue)
	storageMetricValue, ok := server.storage.ReadCounter(metricname)
	if !ok {
		log.Printf("Не найдена метрика %s в хранилище", metricname)
	}
	fmt.Fprintf(w, "Значение метрики в DB: (Тип: counter, Имя: %s, Значение: %d)\n", metricname, storageMetricValue)
}

// Хелпер для сохранение Gauge метрики в хранилище
func (server *Server) SaveGaugeMetric(w http.ResponseWriter, metricname string, metricvalue string) {
	value, err := strconv.ParseFloat(metricvalue, 64)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	gaugemetricvalue := metrics.Gauge(value)
	server.storage.UpdateGauge(metricname, gaugemetricvalue)
	w.WriteHeader(http.StatusOK)
	storageMetricValue, ok := server.storage.ReadGauge(metricname)
	if !ok {
		log.Printf("Не найдена метрика %s в хранилище", metricname)
	}
	fmt.Fprintf(w, "Принята метрика: (Тип: gauge, Имя: %s, Значение: %f)\n", metricname, gaugemetricvalue)
	fmt.Fprintf(w, "Значение метрики в DB: (Тип: gauge, Имя: %s, Значение: %f)\n", metricname, storageMetricValue)
}
