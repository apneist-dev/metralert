package server

import (
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type gauge float64
type counter int64

type MemStorage struct {
	Gdb map[string]gauge
	Cdb map[string]counter
}

type Server struct{ url string }

func NewServer(url string) Server {
	db := MemStorage{
		Gdb: make(map[string]gauge),
		Cdb: make(map[string]counter),
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Route("/update", func(r chi.Router) {
		r.Post("/{metrictype}/{metricname}/{metricvalue}", db.UpdateHandler)
	})
	r.Get("/", db.GetMainHandler)
	r.Post("/", PostMainHandler)
	r.Route("/value", func(r chi.Router) {
		r.Get("/{metrictype}/{metricname}", db.GetMetricHandler)
	})
	http.ListenAndServe(url, r)
	return Server{url}
}

// TODO interface
func (db MemStorage) SaveGaugeMetric(metricname string, metricvalue gauge) {
	db.Gdb[metricname] = metricvalue
}

// TODO interface
func (db MemStorage) SaveCounterMetric(metricname string, metricvalue counter) {
	db.Cdb[metricname] += metricvalue
}

// Обработчик для Post "/"
func PostMainHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

// Обработчик для вывод всех метрик
func (db MemStorage) GetMainHandler(w http.ResponseWriter, r *http.Request) {
	// Output := struct {
	// 	Header     string
	// 	Gaugemap   map[string]gauge
	// 	Countermap map[string]counter
	// }{
	// 	Header:     "Метрики MemStats",
	// 	Gaugemap:   db.Gdb,
	// 	Countermap: db.Cdb,
	// }
	tmpl, err := template.ParseFiles("internal/server/templates/mainpage.html")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	tmpl.Execute(w, db.Gdb)
	tmpl.Execute(w, db.Cdb)
}

// Обработчик для записи одной метрики
func (db MemStorage) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metrictype := chi.URLParam(r, "metrictype")
	metricname := chi.URLParam(r, "metricname")
	metricvalue := chi.URLParam(r, "metricvalue")
	types := []string{"gauge", "counter"}
	if !slices.Contains(types, metrictype) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if metrictype == "counter" {
		value, err := strconv.Atoi(metricvalue)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		var countermetricvalue = counter(value)
		db.SaveCounterMetric(metricname, countermetricvalue)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Принята метрика: (Тип: counter, Имя: %s, Значение: %d)\n", metricname, countermetricvalue)
		fmt.Fprintf(w, "Значение метрики в DB: (Тип: counter, Имя: %s, Значение: %d)\n", metricname, db.Cdb[metricname])
	} else if metrictype == "gauge" {
		value, err := strconv.ParseFloat(metricvalue, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		gaugemetricvalue := gauge(value)
		db.SaveGaugeMetric(metricname, gaugemetricvalue)
		fmt.Fprintf(w, "Принята метрика: (Тип: gauge, Имя: %s, Значение: %f)\n", metricname, gaugemetricvalue)
		fmt.Fprintf(w, "Значение метрики в DB: (Тип: gauge, Имя: %s, Значение: %f)\n", metricname, db.Gdb[metricname])
	}
}

// Обработчик для получения одной метрики
func (db MemStorage) GetMetricHandler(w http.ResponseWriter, r *http.Request) {
	metrictype := chi.URLParam(r, "metrictype")
	metricname := chi.URLParam(r, "metricname")
	switch metrictype {
	case "counter":
		metricvalue, ok := db.Cdb[metricname]
		if !ok {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		fmt.Fprint(w, metricvalue)
	case "gauge":
		metricvalue, ok := db.Gdb[metricname]
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
