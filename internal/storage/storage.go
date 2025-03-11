package storage

import (
	"metralert/internal/metrics"
)

type MemStorage struct {
	Gdb map[string]metrics.Gauge
	Cdb map[string]metrics.Counter
}

func New() MemStorage {
	return MemStorage{
		Gdb: make(map[string]metrics.Gauge),
		Cdb: make(map[string]metrics.Counter),
	}
}

func (m *MemStorage) UpdateGauge(metricName string, metricValue metrics.Gauge) {
	m.Gdb[metricName] = metrics.Gauge(metricValue)
}

func (m *MemStorage) UpdateCounter(metricName string, metricValue metrics.Counter) {
	m.Cdb[metricName] += metrics.Counter(metricValue)
}

func (m *MemStorage) ReadGauge(metricName string) (metrics.Gauge, bool) {
	metricValue, ok := m.Gdb[metricName]
	return metricValue, ok
}

func (m *MemStorage) ReadCounter(metricName string) (metrics.Counter, bool) {
	metricValue, ok := m.Cdb[metricName]
	return metricValue, ok
}

func (m *MemStorage) ReadAll() *MemStorage {
	return m
}
