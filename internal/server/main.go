package server

import (
	"fmt"
	"html/template"
	"net/http"
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
		r.Post("/counter/{metricname}/{metricvalue}", db.CounterUpdateHandler)
		r.Post("/gauge/{metricname}/{metricvalue}", db.GaugeUpdateHandler)
	})
	r.Get("/", db.GetMainHandler)
	r.Post("/", PostMainHandler)
	r.Route("/value", func(r chi.Router) {
		r.Get("/{metrictype}/{metricname}", db.GetMetricHandler)
	})
	http.ListenAndServe("localhost:8080", r)
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

// post default handler
func PostMainHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

// print all metrics
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

// handler counter
func (db MemStorage) CounterUpdateHandler(w http.ResponseWriter, r *http.Request) {
	metricname := chi.URLParam(r, "metricname")
	metricvalue := chi.URLParam(r, "metricvalue")

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
}

// handler gauge
func (db MemStorage) GaugeUpdateHandler(w http.ResponseWriter, r *http.Request) {
	metricname := chi.URLParam(r, "metricname")
	metricvalue := chi.URLParam(r, "metricvalue")

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

// handler get metric
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
