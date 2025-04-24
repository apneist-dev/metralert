package storage

import (
	"context"
	"metralert/internal/metrics"

	"go.uber.org/zap"
)

type StorageInterface interface {
	UpdateMetric(ctx context.Context, metric metrics.Metrics) (metrics.Metrics, error)
	UpdateBatchMetrics(ctx context.Context, metrics []metrics.Metrics) ([]metrics.Metrics, error)
	GetMetricByName(ctx context.Context, metric metrics.Metrics) (metrics.Metrics, bool)
	GetMetrics(ctx context.Context) (map[string]any, error)
	PingDatabase(ctx context.Context) error
	BackupService(storeInterval int) error
	Shutdown() error
}

func NewStorage(fileStoragePath string, recover bool, databaseAddress string, logger *zap.SugaredLogger) StorageInterface {
	switch databaseAddress {
	case "":
		return NewMemstorage(fileStoragePath, recover, logger)
	default:
		return NewPgStorage(databaseAddress, logger)
	}
}
