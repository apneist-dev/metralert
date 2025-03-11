package storage

import (
	"metralert/internal/metrics"
)

type MemStorage struct {
	Gaugedb   map[string]metrics.Gauge
	Counterdb map[string]metrics.Counter
}

func New() MemStorage {
	return MemStorage{
		Gaugedb:   make(map[string]metrics.Gauge),
		Counterdb: make(map[string]metrics.Counter),
	}
}

func (m *MemStorage) UpdateGauge(metricName string, metricValue metrics.Gauge) {
	m.Gaugedb[metricName] = metrics.Gauge(metricValue)
}

func (m *MemStorage) UpdateCounter(metricName string, metricValue metrics.Counter) {
	m.Counterdb[metricName] += metrics.Counter(metricValue)
}

func (m *MemStorage) ReadGauge(metricName string) (metrics.Gauge, bool) {
	metricValue, ok := m.Gaugedb[metricName]
	return metricValue, ok
}

func (m *MemStorage) ReadCounter(metricName string) (metrics.Counter, bool) {
	metricValue, ok := m.Counterdb[metricName]
	return metricValue, ok
}

func (m *MemStorage) ReadAllGauge() map[string]metrics.Gauge {
	return m.Gaugedb
}

func (m *MemStorage) ReadAllCounter() map[string]metrics.Counter {
	return m.Counterdb
}

func (m *MemStorage) ReadAll() *MemStorage {
	return m
}
