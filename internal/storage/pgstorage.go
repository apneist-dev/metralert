package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"metralert/internal/metrics"
	"time"

	"go.uber.org/zap"
)

type PgStorage struct {
	database  *sql.DB
	logger    *zap.SugaredLogger
	ctx       context.Context
	ctxCancel context.CancelFunc
}

func NewPgStorage(databaseAddress string, logger *zap.SugaredLogger) *PgStorage {
	pg := PgStorage{
		logger: logger,
	}

	queryCreateTable := `CREATE TABLE IF NOT EXISTS metrics (
		"id" VARCHAR(250) PRIMARY KEY,
		"mtype" VARCHAR(250) NOT NULL DEFAULT '',
		"delta" BIGINT,
		"value" DOUBLE PRECISION
	) `

	pg.ctx, pg.ctxCancel = context.WithCancel(context.Background())
	// defer pg.ctxCancel()

	database, err := sql.Open("pgx", databaseAddress)
	if err != nil {
		pg.logger.Fatalw("Unable to open DB")
	}
	pg.logger.Infow("Database connected")
	pg.database = database

	_, err = pg.database.ExecContext(pg.ctx, queryCreateTable)
	if err != nil {
		pg.logger.Fatalw("Unable to create table", "error", err)
	}
	return &pg
}

func (pg *PgStorage) UpdateMetric(metric metrics.Metrics) (metrics.Metrics, error) {
	queryUpdateGauge := `
		INSERT INTO metrics (id, mtype, value)
    	VALUES ( $1 , 'gauge', $2 )
		ON CONFLICT (id) 
		DO UPDATE SET value = $2
		RETURNING id, mtype, value
		`

	queryUpdateCounter := `
		INSERT INTO metrics (id, mtype, delta)
    	VALUES ( $1 , 'counter', $2 )
		ON CONFLICT (id) 
		DO UPDATE SET delta = $2 + metrics.delta
		RETURNING id, mtype, delta
		`

	switch metric.MType {
	case "gauge":
		var scannedGauge metrics.Metrics
		err := pg.database.QueryRowContext(pg.ctx, queryUpdateGauge,
			metric.ID,
			metric.Value).Scan(&scannedGauge.ID, &scannedGauge.MType, &scannedGauge.Value)

		return scannedGauge, err

	case "counter":
		var scannedCounter metrics.Metrics
		err := pg.database.QueryRowContext(pg.ctx, queryUpdateCounter,
			metric.ID,
			metric.Delta).Scan(&scannedCounter.ID, &scannedCounter.MType, &scannedCounter.Delta)

		return scannedCounter, err
	default:
		err := errors.New("invalid Mtype")
		return metrics.Metrics{}, err
	}
}

func (pg *PgStorage) UpdateBatchMetrics(metricsSlice []metrics.Metrics) ([]metrics.Metrics, error) {
	var result []metrics.Metrics
	var errs []error

	queryUpdateGauge := `
		INSERT INTO metrics (id, mtype, value)
		VALUES ( $1 , 'gauge', $2 )
		ON CONFLICT (id) 
		DO UPDATE SET value = $2
		RETURNING id, mtype, value
		`

	queryUpdateCounter := `
		INSERT INTO metrics (id, mtype, delta)
		VALUES ( $1 , 'counter', $2 )
		ON CONFLICT (id) 
		DO UPDATE SET delta = $2 + metrics.delta
		RETURNING id, mtype, delta
		`

	ctx, cancel := context.WithTimeout(pg.ctx, time.Second*120) //120 for for debug
	defer cancel()

	tx, err := pg.database.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmtCounter, err := tx.PrepareContext(ctx, queryUpdateCounter)
	if err != nil {
		return nil, err
	}

	stmtGauge, err := tx.PrepareContext(ctx, queryUpdateGauge)
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

func (pg *PgStorage) GetMetricByName(metric metrics.Metrics) (metrics.Metrics, bool) {
	queryGetMetric := `
		SELECT id, mtype, delta, value 
		FROM metrics WHERE id = $1
		`

	ok := true
	result := metrics.Metrics{}
	err := pg.database.QueryRowContext(pg.ctx, queryGetMetric,
		metric.ID).Scan(&result.ID, &result.MType, &result.Delta, &result.Value)
	if err != nil {
		ok = false
	}
	return result, ok
}

func (pg *PgStorage) GetMetrics() map[string]any {
	result := make(map[string]any)
	queryGetMetrics := `
		SELECT id, mtype, delta, value 
		FROM metrics
		`
	rows, err := pg.database.QueryContext(pg.ctx, queryGetMetrics)
	if err != nil {
		pg.logger.Warnw("get_metrics error")
	}
	defer rows.Close()

	for rows.Next() {
		var metric metrics.Metrics
		err = rows.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value)
		if err != nil {
			pg.logger.Warnw("got error when reading metric")
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
		pg.logger.Warnw("get_metrics error")
	}

	return result
}

func (pg *PgStorage) Shutdown() error {
	pg.logger.Infow("Backing up storage before shutdown")
	pg.ctxCancel()
	pg.database.Close()
	return nil
}

func (pg *PgStorage) PingDatabase() error {
	if pg.database == nil {
		return errors.New("no database connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := pg.database.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

func (pg *PgStorage) BackupService(storeInterval int) error {
	return nil
}
