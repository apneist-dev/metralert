package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type gauge float64
type counter int64

type MemStorage struct {
	Gdb map[string]gauge
	Cdb map[string]counter
}

type db interface {
	UpdateHandler()
}

// обработчик Update
func (db MemStorage) UpdateHandler(w http.ResponseWriter, r *http.Request) {

	wrongmetricvalue := func() {
		msg := fmt.Sprintf("Запрос с некорректным типом метрики или значением.\n%s\n", r.URL)
		http.Error(w, msg, http.StatusNotFound)
	}
	wrongmetricname := func() {
		msg := fmt.Sprintf("Отсутствует имя метрики в запросе.\n%s\n", r.URL)
		http.Error(w, msg, http.StatusNotFound)
	}
	// fmt.Fprintf(w, "URL.Path = %q\n", r.URL.Path)

	path, _ := strings.CutPrefix(r.URL.Path, "/update/")
	params := strings.Split(path, "/")
	//проверяем Path
	if len(params) != 3 {
		wrongmetricname()
		return
	}

	metrictype, metricname, metricvalue := params[0], params[1], params[2]
	switch metrictype {
	case "gauge":
		value, err := strconv.ParseFloat(metricvalue, 64)
		if err != nil {
			wrongmetricvalue()
			return
		}
		var metricvalue gauge = gauge(value)
		db.Gdb[metricname] = metricvalue
		fmt.Printf("тип метрики gauge, имя метрики %s, значение %f\n", metricname, metricvalue)
	case "counter":
		value, err := strconv.Atoi(metricvalue)
		if err != nil {
			wrongmetricvalue()
			return
		}
		var metricvalue counter = counter(value)
		db.Cdb[metricname] += metricvalue
		fmt.Printf("тип метрики counter, имя метрики %s, значение %d\n", metricname, metricvalue)
		fmt.Printf("значение метрики %s в DB: %d\n", metricname, db.Cdb[metricname])
	default:
		wrongmetricvalue()
		return
	}
}

func main() {

	db := MemStorage{
		Gdb: make(map[string]gauge),
		Cdb: make(map[string]counter),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/update/", db.UpdateHandler)
	log.Fatal(http.ListenAndServe("localhost:8080", mux))
}
