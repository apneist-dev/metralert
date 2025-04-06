package server

import (
	"bytes"
	"encoding/json"
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
			sugar := logger.Sugar()
			storage := storage.New("internal/storage/metrics_database.json", false, "localhost", logger.Sugar())
			server := New(tt.args.url, storage, sugar)
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
