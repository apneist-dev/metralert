package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var (
	serverurl = "http://localhost:8080"
)

type gauge float64
type counter int64

type MemStorage struct {
	Gdb map[string]gauge
	Cdb map[string]counter
}

func main() {

	db := MemStorage{
		Gdb: make(map[string]gauge),
		Cdb: make(map[string]counter),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/update/counter/", db.CounterUpdateHandler)
	mux.HandleFunc("/update/gauge/", db.GaugeUpdateHandler)
	mux.HandleFunc("/", MainHandler)
	log.Fatal(http.ListenAndServe(serverurl, mux))
}

func (db MemStorage) SaveGaugeMetric(metricname string, metricvalue gauge) {
	db.Gdb[metricname] = metricvalue
}

func (db MemStorage) SaveCounterMetric(metricname string, metricvalue counter) {
	db.Cdb[metricname] += metricvalue
}

func MainHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
}

// обработчик counter
func (db MemStorage) CounterUpdateHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	// fmt.Fprintf(w, "URL.Path = %q\n", r.URL.Path)

	path, _ := strings.CutPrefix(r.URL.Path, "/update/counter/")
	params := strings.Split(path, "/")
	//проверяем Path
	if len(params) != 2 {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	metricname, metricvalue := params[0], params[1]
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

// обработчик gauge
func (db MemStorage) GaugeUpdateHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	// fmt.Fprintf(w, "URL.Path = %q\n", r.URL.Path)

	path, _ := strings.CutPrefix(r.URL.Path, "/update/gauge/")
	params := strings.Split(path, "/")
	//проверяем Path
	if len(params) != 2 {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	metricname, metricvalue := params[0], params[1]
	value, err := strconv.ParseFloat(metricvalue, 64)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var gaugemetricvalue = gauge(value)
	db.SaveGaugeMetric(metricname, gaugemetricvalue)
	fmt.Fprintf(w, "Принята метрика: (Тип: gauge, Имя: %s, Значение: %f)\n", metricname, gaugemetricvalue)
	fmt.Fprintf(w, "Значение метрики в DB: (Тип: gauge, Имя: %s, Значение: %f)\n", metricname, db.Gdb[metricname])
}
