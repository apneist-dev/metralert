package storage

import (
	. "metralert/internal/metrics"
)

type MemStorage struct {
	Gdb map[string]Gauge
	Cdb map[string]Counter
}

func New() MemStorage {
	return MemStorage{
		Gdb: make(map[string]Gauge),
		Cdb: make(map[string]Counter),
	}
}

func (m *MemStorage) UpdateGauge(metricName string, metricValue Gauge) {
	m.Gdb[metricName] = Gauge(metricValue)
}

func (m *MemStorage) UpdateCounter(metricName string, metricValue Counter) {
	m.Cdb[metricName] += Counter(metricValue)
}

func (m *MemStorage) ReadGauge(metricName string) (Gauge, bool) {
	metricValue, ok := m.Gdb[metricName]
	return metricValue, ok
}

func (m *MemStorage) ReadCounter(metricName string) (Counter, bool) {
	metricValue, ok := m.Cdb[metricName]
	return metricValue, ok
}

func (m *MemStorage) ReadAll() *MemStorage {
	return m
}
