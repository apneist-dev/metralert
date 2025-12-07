package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	config "metralert/config/server"
	"metralert/internal/metrics"
	"metralert/internal/storage"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestServer_UpdateMetricJSONHandler(t *testing.T) {
	type args struct {
		requestBody metrics.Metrics
		url         string
		metricDelta int64
	}
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name        string
		args        args
		metricDelta int64
		want        want
	}{
		{
			name: "Update Test #1",
			args: args{
				requestBody: metrics.Metrics{
					ID:    "NewCounter",
					MType: "counter",
				},
				metricDelta: 123222,
			},
			want: want{
				code:        200,
				contentType: "application/json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := zap.NewDevelopment()
			storage := storage.NewStorage(config.Config{
				FileStoragePath: "internal/storage/metrics_database.json",
				Logger:          logger.Sugar(),
				Restore:         false,
			})
			server := New(config.Config{
				ServerAddress: tt.args.url,
				Storage:       storage,
				Logger:        logger.Sugar(),
				Restore:       false,
			})
			tt.args.requestBody.Delta = (*int64)(&tt.args.metricDelta)
			jsonBody, err := json.Marshal(tt.args.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal JSON: %v", err)
			}

			r := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(jsonBody))
			w := httptest.NewRecorder()
			server.UpdateMetricJSONHandler(w, r)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.want.code, res.StatusCode)

		})
	}
}

func ExampleServer_UpdateMetricJSONHandler() {
	logger, _ := zap.NewDevelopment()
	storage := storage.NewStorage(config.Config{
		FileStoragePath: "internal/storage/metrics_database.json",
		Logger:          logger.Sugar(),
		Restore:         false,
	})
	server := New(config.Config{
		ServerAddress: "http://localhost:8080",
		Storage:       storage,
		Logger:        logger.Sugar(),
		Restore:       false,
	})
	jsonBody, err := json.Marshal(metrics.Metrics{
		ID:    "NewCounter",
		MType: "counter",
	})
	if err != nil {
		fmt.Printf("Failed to marshal JSON: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()
	server.UpdateMetricJSONHandler(w, r)
}

func ExampleServer_ReadMetricJSONHandler() {
	var TestDelta int64 = 123

	logger, _ := zap.NewDevelopment()
	storage := storage.NewStorage(config.Config{
		FileStoragePath: "internal/storage/metrics_database.json",
		Logger:          logger.Sugar(),
		Restore:         false,
	})
	server := New(config.Config{
		ServerAddress: "http://localhost:8080",
		Storage:       storage,
		Logger:        logger.Sugar(),
		Restore:       false,
	})
	metricsNewCounter := metrics.Metrics{
		ID:    "NewCounter",
		MType: "counter",
	}

	metricsNewCounter.Delta = &TestDelta

	jsonBody, err := json.Marshal(metricsNewCounter)
	if err != nil {
		fmt.Printf("Failed to marshal JSON: %v", err)
	}

	ctx := context.Background()

	_, err = server.storage.UpdateMetric(ctx, metricsNewCounter)
	if err != nil {
		fmt.Printf("Failed to update metric: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBuffer(jsonBody))
	w := httptest.NewRecorder()

	server.ReadMetricJSONHandler(w, r)

	fmt.Println("Response Code is", w.Code)
	fmt.Println("Response Body is", w.Body)

	// Output:
	// Response Code is 200
	// Response Body is {"id":"NewCounter","type":"counter","delta":123}
}
