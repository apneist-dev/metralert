package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"metralert/internal/metrics"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

const (
	GaugeStr   string = "gauge"
	CounterStr string = "counter"
)

type MemStorage struct {
	db              map[string]metrics.Metrics
	fileStoragePath string
	logger          *zap.SugaredLogger
}

func NewMemstorage(fileStoragePath string, recover bool, logger *zap.SugaredLogger) *MemStorage {
	m := MemStorage{
		db:              make(map[string]metrics.Metrics),
		fileStoragePath: fileStoragePath,
		logger:          logger,
	}

	if recover {
		jsonData, err := os.ReadFile(fileStoragePath)
		if err != nil {
			logger.Infow("Unable to open file, creating new", "Path", fileStoragePath)
			_, err := os.Create(m.fileStoragePath)
			if err != nil {
				m.logger.Fatalw("Unable to create file", "Path", m.fileStoragePath)
			}
			m.logger.Infow("File created successfilly", "Path", m.fileStoragePath)
		}
		logger.Infow("File opened", "Path", fileStoragePath)

		err = json.Unmarshal(jsonData, &m.db)
		if err != nil || m.db == nil {
			logger.Warnw("Unable to Unmarshal structure, creating empty database")
			return &MemStorage{
				db:              make(map[string]metrics.Metrics),
				fileStoragePath: fileStoragePath,
				logger:          logger,
			}
		}
		logger.Infow("Recovered sussessfully")
		logger.Debugw("Recovered database", "DB", m.db)
	}
	return &m
}

func (m *MemStorage) ValidateMetric(metric metrics.Metrics) error {
	var err error
	switch metric.MType {
	case GaugeStr:
		if metric.Value == nil {
			return errors.New("invalid Value")
		}
	case CounterStr:
		if metric.Delta == nil {
			return errors.New("invalid Delta")
		}
	}
	return err
}

func (m *MemStorage) UpdateMetric(_ context.Context, metric metrics.Metrics) (*metrics.Metrics, error) {
	err := m.ValidateMetric(metric)
	if err != nil {
		return nil, err
	}

	// case in-memory
	switch metric.MType {
	case "gauge":
		m.db[metric.ID] = metrics.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Value: metric.Value,
		}
	case "counter":
		var newDelta int64
		_, ok := m.db[metric.ID]
		if !ok {
			newDelta = (int64)(*metric.Delta)
		} else {
			newDelta = *m.db[metric.ID].Delta + (int64)(*metric.Delta)
		}
		m.db[metric.ID] = metrics.Metrics{
			ID:    metric.ID,
			MType: metric.MType,
			Delta: &newDelta,
			// Value: &newValue,
		}
	default:
		err = errors.New("invalid Mtype")
	}
	metricItem := m.db[metric.ID]
	return &metricItem, err
}

func (m *MemStorage) UpdateBatchMetrics(_ context.Context, metricsSlice []metrics.Metrics) ([]metrics.Metrics, error) {
	var result []metrics.Metrics
	var errs []error

	// case in-memory
	for _, metric := range metricsSlice {
		switch metric.MType {
		case "gauge":
			m.db[metric.ID] = metrics.Metrics{
				ID:    metric.ID,
				MType: metric.MType,
				Value: metric.Value,
			}
			result = append(result, m.db[metric.ID])
		case "counter":
			var newDelta int64
			_, ok := m.db[metric.ID]
			if !ok {
				newDelta = (int64)(*metric.Delta)
			} else {
				newDelta = *m.db[metric.ID].Delta + (int64)(*metric.Delta)
			}
			m.db[metric.ID] = metrics.Metrics{
				ID:    metric.ID,
				MType: metric.MType,
				Delta: &newDelta,
				// Value: &newValue,
			}
			result = append(result, m.db[metric.ID])
		default:
			err := errors.New("invalid Mtype")
			errs = append(errs, err)
		}
	}
	return result, errors.Join(errs...)
}

func (m *MemStorage) GetMetricByName(_ context.Context, metric metrics.Metrics) (*metrics.Metrics, bool) {
	// case in-memory
	result, ok := m.db[metric.ID]
	return &result, ok
}

func (m *MemStorage) GetMetrics(_ context.Context) (map[string]any, error) {
	// case in-memory
	result := make(map[string]any)
	for id, metric := range m.db {
		switch metric.MType {
		case "gauge":
			result[id] = fmt.Sprintf("%f", *metric.Value)
		case "counter":
			result[id] = fmt.Sprintf("%d", *metric.Delta)
		}
	}
	return result, nil
}

func (m *MemStorage) SaveDatabase() error {
	file, err := os.Create(m.fileStoragePath)
	if err != nil {
		return err
	}
	m.logger.Infow("File created successfilly", "Path", m.fileStoragePath)

	data, err := json.Marshal(m.db)
	if err != nil {
		m.logger.Warnw("Unable to marshal structure")
	}
	_, err1 := file.Write(data)
	if err1 != nil {
		m.logger.Warnw("Unable to write to file", "Path", m.fileStoragePath)
		return nil
	}

	m.logger.Infow("Database saved to file sucessfully", "Path", m.fileStoragePath)

	return nil
}
func (m *MemStorage) BackupService(storeInterval int) error {

	for {
		time.Sleep(time.Duration(storeInterval) * time.Second)
		err := m.SaveDatabase()
		if err != nil {
			return err
		}
	}
}

func (m *MemStorage) Shutdown() error {
	m.logger.Infow("Backing up storage before shutdown")

	err := m.SaveDatabase()
	if err != nil {
		return err
	}
	return nil
}

func (m *MemStorage) PingDatabase(_ context.Context) error {
	return errors.New("no database connected")
}
