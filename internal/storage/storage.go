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
	SQLCreateTable = `CREATE TABLE IF NOT EXISTS metrics (
		"id" VARCHAR(250) PRIMARY KEY,
		"mtype" VARCHAR(250) NOT NULL DEFAULT '',
		"delta" BIGINT,
		"value" DOUBLE PRECISION
	) `
	SQLUpdateCounter = `INSERT INTO metrics (id, mtype, delta)
    	VALUES ( $1 , 'counter', $2 )
		ON CONFLICT (id) 
		DO UPDATE SET delta = $2 + metrics.delta
		RETURNING id, mtype, delta`
	SQLUpdateGauge = `INSERT INTO metrics (id, mtype, value)
    	VALUES ( $1 , 'gauge', $2 )
		ON CONFLICT (id) 
		DO UPDATE SET value = $2
		RETURNING id, mtype, value`
	SQLGetMetric  = `SELECT id, mtype, delta, value FROM metrics WHERE id = $1`
	SQLGetMetrics = `SELECT id, mtype, delta, value FROM metrics`
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

		_, err = m.database.ExecContext(m.ctx, SQLCreateTable)
		if err != nil {
			m.logger.Fatalw("Unable to create table", "error", err)
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
	// currentMetric, ok := m.db[metric.ID]
	// if ok && currentMetric.MType != metric.MType {
	// 	return fmt.Errorf("metric %s exists with another type %s", metric.ID, m.db[metric.ID].MType)
	// }
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
			err := m.database.QueryRowContext(m.ctx, SQLUpdateGauge,
				metric.ID,
				metric.Value).Scan(&scannedGauge.ID, &scannedGauge.MType, &scannedGauge.Value)

			return scannedGauge, err

		case "counter":
			var scannedCounter metrics.Metrics
			err := m.database.QueryRowContext(m.ctx, SQLUpdateCounter,
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

func (m *MemStorage) UpdateBatchMetrics(metricsSlice []metrics.Metrics) ([]metrics.Metrics, error) {
	var result []metrics.Metrics
	var errs []error

	// var emptyMetric metrics.Metrics
	// err := m.validateMetric(metric)
	// if err != nil {
	// 	return emptyMetric, err
	// }

	// case database
	if m.database != nil {
		ctx, cancel := context.WithTimeout(m.ctx, time.Second*120) //120 for for debug
		defer cancel()

		tx, err := m.database.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()

		stmtCounter, err := tx.PrepareContext(ctx, SQLUpdateCounter)
		if err != nil {
			return nil, err
		}

		stmtGauge, err := tx.PrepareContext(ctx, SQLUpdateGauge)
		if err != nil {
			return nil, err
		}

		for _, metric := range metricsSlice {
			switch metric.MType {
			case "gauge":
				var scannedGauge metrics.Metrics
				err := stmtGauge.QueryRowContext(ctx, metric.ID, metric.Value).Scan(&scannedGauge.ID, &scannedGauge.MType, &scannedGauge.Value)

				result = append(result, scannedGauge)
				if err != nil {
					errs = append(errs, err)
				}

			case "counter":
				var scannedCounter metrics.Metrics
				err := stmtCounter.QueryRowContext(ctx, metric.ID, metric.Delta).Scan(&scannedCounter.ID, &scannedCounter.MType, &scannedCounter.Delta)

				result = append(result, scannedCounter)
				if err != nil {
					errs = append(errs, err)
				}

			default:
				err := errors.New("invalid Mtype")
				errs = append(errs, err)

			}
		}
		err = tx.Commit()
		if err != nil {
			errs = append(errs, err)
			return nil, errors.Join(errs...)
		}
		return result, errors.Join(errs...)
	}

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

func (m *MemStorage) GetMetricByName(metric metrics.Metrics) (metrics.Metrics, bool) {
	// case database
	if m.database != nil {
		ok := true
		result := metrics.Metrics{}
		err := m.database.QueryRowContext(m.ctx, SQLGetMetric,
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
		rows, err := m.database.QueryContext(m.ctx, SQLGetMetrics)
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
