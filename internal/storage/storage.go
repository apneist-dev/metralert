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
	ctx             context.Context
	ctxCancel       context.CancelFunc
}

const (
	SqlCreateTable = `CREATE TABLE IF NOT EXISTS metrics (
		"id" VARCHAR(250) PRIMARY KEY,
		"mtype" VARCHAR(250) NOT NULL DEFAULT '',
		"delta" BIGINT,
		"value" DOUBLE PRECISION
	) `
	SqlUpdateCounter = `INSERT INTO metrics (id, mtype, delta)
    	VALUES ( $1 , 'counter', $2 )
		ON CONFLICT (id) 
		DO UPDATE SET delta = $2 + metrics.delta
		RETURNING id, mtype, delta`
	SqlUpdateGauge = `INSERT INTO metrics (id, mtype, value)
    	VALUES ( $1 , 'gauge', $2 )
		ON CONFLICT (id) 
		DO UPDATE SET value = $2
		RETURNING id, mtype, value`
	SqlGetMetric  = `SELECT id, mtype, delta, value FROM metrics WHERE id = $1`
	SqlGetMetrics = `SELECT id, mtype, delta, value FROM metrics`
)

func New(fileStoragePath string, recover bool, databaseAddress string, logger *zap.SugaredLogger) *MemStorage {
	m := MemStorage{
		db:              make(map[string]metrics.Metrics),
		fileStoragePath: fileStoragePath,
		logger:          logger,
	}
	m.ctx, m.ctxCancel = context.WithCancel(context.Background())

	if databaseAddress != "" {
		database, err := sql.Open("pgx", databaseAddress)
		if err != nil {
			m.logger.Fatalw("Unable to open DB")
		}
		m.logger.Infow("Database connected")
		m.database = database

		_, err = m.database.ExecContext(m.ctx, SqlCreateTable)
		if err != nil {
			m.logger.Fatalw("Unable to create table")
		}
		return &m
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

	// case database
	if m.database != nil {
		switch metric.MType {
		case "gauge":
			var scannedGauge metrics.Metrics
			err := m.database.QueryRowContext(m.ctx, SqlUpdateGauge,
				metric.ID,
				metric.Value).Scan(&scannedGauge.ID, &scannedGauge.MType, &scannedGauge.Value)

			return scannedGauge, err

		case "counter":
			var scannedCounter metrics.Metrics
			err := m.database.QueryRowContext(m.ctx, SqlUpdateCounter,
				metric.ID,
				metric.Delta).Scan(&scannedCounter.ID, &scannedCounter.MType, &scannedCounter.Delta)

			return scannedCounter, err
		default:
			err = errors.New("invalid Mtype")
			return metrics.Metrics{}, err
		}
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
	return m.db[metric.ID], err
}

func (m *MemStorage) GetMetricByName(metric metrics.Metrics) (metrics.Metrics, bool) {
	// case database
	if m.database != nil {
		ok := true
		result := metrics.Metrics{}
		err := m.database.QueryRowContext(m.ctx, SqlGetMetric,
			metric.ID).Scan(&result.ID, &result.MType, &result.Delta, &result.Value)
		if err != nil {
			ok = false
		}
		return result, ok
	}

	// case in-memory
	result, ok := m.db[metric.ID]
	return result, ok
}

func (m *MemStorage) GetMetrics() map[string]any {
	// case database
	if m.database != nil {
		// var allMetrics []metrics.Metrics
		result := make(map[string]any)
		rows, err := m.database.QueryContext(m.ctx, SqlGetMetrics)
		if err != nil {
			m.logger.Warnw("get_metrics error")
		}
		defer rows.Close()

		for rows.Next() {
			var metric metrics.Metrics
			err = rows.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value)
			if err != nil {
				m.logger.Warnw("got error when reading metric")
			}

			// allMetrics = append(allMetrics, metric)

			switch metric.MType {
			case "gauge":
				// защищаемся от nil dereference
				if metric.Value == nil {
					continue
				}
				result[metric.ID] = fmt.Sprintf("%f", *metric.Value)
			case "counter":
				// защищаемся от nil dereference
				if metric.Delta == nil {
					continue
				}
				result[metric.ID] = fmt.Sprintf("%d", *metric.Delta)
			}
		}

		err = rows.Err()
		if err != nil {
			m.logger.Warnw("get_metrics error")
		}

		return result
	}

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
		m.ctxCancel()
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
