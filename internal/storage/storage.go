package storage

import (
	"metralert/internal/metrics"

	"go.uber.org/zap"
)

type Storage interface {
	UpdateMetric(metric metrics.Metrics) (metrics.Metrics, error)
	UpdateBatchMetrics(metrics []metrics.Metrics) ([]metrics.Metrics, error)
	GetMetricByName(metric metrics.Metrics) (metrics.Metrics, bool)
	GetMetrics() map[string]any
	PingDatabase() error
	BackupService(storeInterval int) error
	Shutdown() error
}

func NewStorage(fileStoragePath string, recover bool, databaseAddress string, logger *zap.SugaredLogger) Storage {
	switch databaseAddress {
	case "":
		return NewMemstorage(fileStoragePath, recover, logger)
	default:
		return NewPgStorage(databaseAddress, logger)
	}
}
