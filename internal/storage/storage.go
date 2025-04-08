package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"metralert/internal/metrics"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

type MemStorage struct {
	db              map[string]metrics.Metrics
	fileStoragePath string
	logger          *zap.SugaredLogger
	database        *sql.DB
}

func New(fileStoragePath string, recover bool, databaseAddress string, logger *zap.SugaredLogger) *MemStorage {
	m := MemStorage{
		db:              make(map[string]metrics.Metrics),
		fileStoragePath: fileStoragePath,
		logger:          logger,
	}

	if databaseAddress != "" {
		database, err := sql.Open("pgx", databaseAddress)
		if err != nil {
			m.logger.Fatalw("Unable to open DB")
		}
		m.database = database
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

func (m *MemStorage) UpdateMetric(metric metrics.Metrics) (metrics.Metrics, error) {
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
	return m.db[metric.ID], err
}

func (m *MemStorage) GetMetricByName(metric metrics.Metrics) (metrics.Metrics, bool) {
	result, ok := m.db[metric.ID]
	return result, ok
}

func (m *MemStorage) GetMetrics() map[string]string {
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
	if m.database != nil {
		m.database.Close()
		return nil
	}

	err := m.SaveDatabase()
	if err != nil {
		return err
	}
	return nil
}

func (m *MemStorage) PingDatabase() error {
	// ps := m.databaseAddress

	// db, err := sql.Open("pgx", ps)
	// if err != nil {
	// 	panic(err)
	// }
	// defer db.Close()
	if m.database == nil {
		return errors.New("no database connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := m.database.PingContext(ctx); err != nil {
		return err
	}

	return nil
}
