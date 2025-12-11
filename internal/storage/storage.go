package storage

import (
	"context"
	config "metralert/config/server"
	"metralert/internal/metrics"
)

type StorageInterface interface {
	UpdateMetric(ctx context.Context, metric metrics.Metrics) (*metrics.Metrics, error)
	UpdateBatchMetrics(ctx context.Context, metrics []metrics.Metrics) ([]metrics.Metrics, error)
	GetMetricByName(ctx context.Context, metric metrics.Metrics) (*metrics.Metrics, bool)
	GetMetrics(ctx context.Context) (map[string]any, error)
	PingDatabase(ctx context.Context) error
	BackupService(storeInterval int) error
	Shutdown() error
}

func NewStorage(cfg config.Config) StorageInterface {
	switch cfg.DatabaseAddress {
	case "":
		return NewMemstorage(cfg.FileStoragePath, cfg.Restore, cfg.Logger)
	default:
		return NewPgStorage(cfg.DatabaseAddress, cfg.Logger)
	}
}
