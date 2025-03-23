package storage

import (
	"errors"
	"fmt"
	"metralert/internal/metrics"
)

type MemStorage struct {
	db map[string]metrics.Metrics
}

func New() MemStorage {
	return MemStorage{
		db: make(map[string]metrics.Metrics),
	}
}

func (m *MemStorage) validateMetric(metric metrics.Metrics) error {
	var err error
	currentMetric, ok := m.db[metric.ID]
	if ok && currentMetric.MType != metric.MType {
		return fmt.Errorf("metric %s exists with another type %s", metric.ID, m.db[metric.ID].MType)
	}
	switch metric.MType {
	case "gauge":
		if metric.Value == nil {
			return errors.New("invalid Value")
		}
	case "counter":
		if metric.Delta == nil {
			return errors.New("invalid Delta")
		}
	}
	return err
}

func (m *MemStorage) Update(metric metrics.Metrics) (metrics.Metrics, error) {
	var emptyMetric metrics.Metrics
	err := m.validateMetric(metric)
	if err != nil {
		return emptyMetric, err
	}

	switch metric.MType {
	case "gauge":
		m.db[metric.ID] = metrics.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Value: metric.Value,
		}
	case "counter":
		var newValue float64
		_, ok := m.db[metric.ID]
		if !ok {
			newValue = (float64)(*metric.Delta)
		} else {
			newValue = *m.db[metric.ID].Value + (float64)(*metric.Delta)
		}
		m.db[metric.ID] = metrics.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Delta: metric.Delta,
			Value: &newValue,
		}
	default:
		err = errors.New("invalid Mtype")
	}
	return m.db[metric.ID], err
}

func (m *MemStorage) Read(metric metrics.Metrics) (metrics.Metrics, bool) {
	result, ok := m.db[metric.ID]
	return result, ok
}

func (m *MemStorage) ReadAll() map[string]string {
	result := make(map[string]string)
	for id, metric := range m.db {
		switch metric.MType {
		case "gauge":
			result[id] = fmt.Sprintf("%f", *metric.Value)
		case "counter":
			result[id] = fmt.Sprintf("%d", *metric.Delta)
		}
	}
	return result
}
