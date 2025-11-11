package storage

import (
	"fmt"
	"metralert/internal/metrics"

	"go.uber.org/zap"
)

func ExampleMemStorage_ValidateMetric() {
	deltaMetrics := metrics.Metrics{
		ID:    "NewCounter",
		MType: "counter",
	}
	logger, _ := zap.NewDevelopment()
	storage := NewMemstorage("internal/storage/metrics_database.json", false, logger.Sugar())

	err := storage.ValidateMetric(deltaMetrics)
	fmt.Println(err)

	// Output:
	// invalid Delta

}

func ExampleMemStorage_ValidateMetric_second() {
	var delta int64 = 123

	deltaMetrics := metrics.Metrics{
		ID:    "NewCounter",
		MType: "counter",
	}
	deltaMetrics.Delta = &delta

	logger, _ := zap.NewDevelopment()
	storage := NewMemstorage("internal/storage/metrics_database.json", false, logger.Sugar())

	err := storage.ValidateMetric(deltaMetrics)
	fmt.Println(err)

	// Output:
	// <nil>

}
