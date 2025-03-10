package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"slices"
	"strconv"

	. "metralert/internal/metrics"
	"metralert/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	url     string
	storage *storage.MemStorage
}

func New(url string, repo *storage.MemStorage) Server {
	return Server{
		url:     url,
		storage: repo,
	}
}

func (server *Server) Start() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Route("/update", func(r chi.Router) {
		r.Post("/{metrictype}/{metricname}/{metricvalue}", server.UpdateHandler)
	})
	r.Get("/", server.GetMainHandler)
	r.Route("/value", func(r chi.Router) {
		r.Get("/{metrictype}/{metricname}", server.GetMetricHandler)
	})
	http.ListenAndServe(server.url, r)
}

// Обработчик для вывод всех метрик в html страницу
func (server *Server) GetMainHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("internal/server/templates/mainpage.html")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	tmpl.Execute(w, server.storage.Gdb)
	tmpl.Execute(w, server.storage.Cdb)
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
	var counterMetricValue = Counter(value)
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

	gaugemetricvalue := Gauge(value)
	server.storage.UpdateGauge(metricname, gaugemetricvalue)
	w.WriteHeader(http.StatusOK)
	storageMetricValue, ok := server.storage.ReadGauge(metricname)
	if !ok {
		log.Printf("Не найдена метрика %s в хранилище", metricname)
	}
	fmt.Fprintf(w, "Принята метрика: (Тип: gauge, Имя: %s, Значение: %f)\n", metricname, gaugemetricvalue)
	fmt.Fprintf(w, "Значение метрики в DB: (Тип: gauge, Имя: %s, Значение: %f)\n", metricname, storageMetricValue)
}
