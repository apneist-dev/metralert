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
	database *sql.DB
	logger   *zap.SugaredLogger
}

func Retry(ctx context.Context, fn func(ctx context.Context) error) error {
	var errs []error
	var err error
	for i := range 3 {
		err = fn(ctx)
		if err == nil {
			return nil
		}

		errs = append(errs, err)
		delay := (i*2 + 1)
		time.Sleep(time.Duration(delay) * time.Second)
	}
	return fmt.Errorf("failed after 3 retries %s", errs)
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

	ctx, ctxCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer ctxCancel()

	database, err := sql.Open("pgx", databaseAddress)
	if err != nil {
		pg.logger.Fatalw("Unable to open DB")
	}
	pg.logger.Infow("Database connected")
	pg.database = database

	_, err = pg.database.ExecContext(ctx, queryCreateTable)
	if err != nil {
		pg.logger.Fatalw("Unable to create table", "error", err)
	}
	return &pg
}

func (pg *PgStorage) UpdateMetric(reqCtx context.Context, metric metrics.Metrics) (*metrics.Metrics, error) {
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

	ctx, ctxCancel := context.WithTimeout(reqCtx, 3*time.Second)
	defer ctxCancel()

	switch metric.MType {
	case "gauge":
		var scannedGauge metrics.Metrics
		err := Retry(ctx, func(ctx context.Context) error {
			return pg.database.QueryRowContext(ctx, queryUpdateGauge,
				metric.ID, metric.Value).Scan(&scannedGauge.ID, &scannedGauge.MType, &scannedGauge.Value)
		})

		return &scannedGauge, err

	case "counter":
		var scannedCounter metrics.Metrics

		err := Retry(ctx, func(ctx context.Context) error {
			return pg.database.QueryRowContext(ctx, queryUpdateCounter,
				metric.ID, metric.Delta).Scan(&scannedCounter.ID, &scannedCounter.MType, &scannedCounter.Delta)
		})

		return &scannedCounter, err
	default:
		err := errors.New("invalid Mtype")
		return &metrics.Metrics{}, err
	}
}

func (pg *PgStorage) UpdateBatchMetrics(reqCtx context.Context, metricsSlice []metrics.Metrics) ([]metrics.Metrics, error) {
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

	ctx, ctxCancel := context.WithTimeout(reqCtx, 3*time.Second)
	defer ctxCancel()

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
			err := Retry(ctx, func(ctx context.Context) error {
				return stmtGauge.QueryRowContext(ctx,
					metric.ID, metric.Value).Scan(&scannedGauge.ID, &scannedGauge.MType, &scannedGauge.Value)
			})

			result = append(result, scannedGauge)
			if err != nil {
				errs = append(errs, err)
			}

		case "counter":
			var scannedCounter metrics.Metrics
			err := Retry(ctx, func(ctx context.Context) error {
				return stmtCounter.QueryRowContext(ctx,
					metric.ID, metric.Delta).Scan(&scannedCounter.ID, &scannedCounter.MType, &scannedCounter.Delta)
			})

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

func (pg *PgStorage) GetMetricByName(reqCtx context.Context, metric metrics.Metrics) (*metrics.Metrics, bool) {
	queryGetMetric := `
		SELECT id, mtype, delta, value 
		FROM metrics WHERE id = $1
		`

	ctx, ctxCancel := context.WithTimeout(reqCtx, 3*time.Second)
	defer ctxCancel()

	ok := true
	result := metrics.Metrics{}
	err := Retry(ctx, func(ctx context.Context) error {
		return pg.database.QueryRowContext(ctx, queryGetMetric,
			metric.ID).Scan(&result.ID, &result.MType, &result.Delta, &result.Value)
	})

	if err != nil {
		ok = false
	}
	return &result, ok
}

func (pg *PgStorage) GetMetrics(reqCtx context.Context) (map[string]any, error) {
	var rows *sql.Rows
	result := make(map[string]any)
	queryGetMetrics := `
		SELECT id, mtype, delta, value 
		FROM metrics
		`

	ctx, ctxCancel := context.WithTimeout(reqCtx, 3*time.Second)
	defer ctxCancel()

	err := Retry(ctx, func(ctx context.Context) error {
		var err error
		rows, err = pg.database.QueryContext(ctx, queryGetMetrics)
		if err != nil {
			return err
		}

		err = rows.Err()
		if err != nil {
			pg.logger.Warnw("get_metrics error")
			return err
		}
		return nil
	})

	if err != nil {
		pg.logger.Warnw("get_metrics error")
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var metric metrics.Metrics
		err = rows.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value)
		if err != nil {
			pg.logger.Warnw("got error when reading metric")
		}

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

	return result, nil
}

func (pg *PgStorage) Shutdown() error {
	pg.logger.Infow("Closing database connection")
	// pg.ctxCancel()
	pg.database.Close()
	return nil
}

func (pg *PgStorage) PingDatabase(reqCtx context.Context) error {
	if pg.database == nil {
		return errors.New("no database connected")
	}

	ctx, cancel := context.WithTimeout(reqCtx, 1*time.Second)
	defer cancel()
	if err := pg.database.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

func (pg *PgStorage) BackupService(storeInterval int) error {
	return nil
}
