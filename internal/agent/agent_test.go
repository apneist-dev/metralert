package agent

import (
	agentConfig "metralert/config/agent"
	serverConfig "metralert/config/server"
	"metralert/internal/metrics"
	"metralert/internal/server"
	"metralert/internal/storage"
	"net/http"
	"runtime"
	"testing"
	"time"

	_ "github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestAgent_SendPost(t *testing.T) {
	type fields struct {
		serverurl        string
		agenturl         string
		pollInterval     int
		reportInterval   int
		pollCount        metrics.Counter
		memoryStatistics []metrics.Metrics
		rtm              runtime.MemStats
		client           http.Client
	}
	type args struct {
		metric    metrics.Metrics
		randValue float64
	}
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    want
		wantErr bool
	}{
		{
			name: "SendPost test #1",
			fields: fields{
				serverurl:      "localhost:8080",
				agenturl:       "http://localhost:8080",
				pollInterval:   2,
				reportInterval: 10,
				pollCount:      12,
			},
			args: args{
				metric: metrics.Metrics{
					ID:    "RandomValue",
					MType: "gauge",
				},
				randValue: 123123,
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := zap.NewDevelopment()
			storage := storage.NewStorage(serverConfig.Config{
				FileStoragePath: "internal/storage/metrics_database.json",
				Logger:          logger.Sugar(),
				Restore:         false,
			})

			server := server.New(serverConfig.Config{
				ServerAddress: tt.fields.serverurl,
				Storage:       storage,
				Logger:        logger.Sugar(),
				Restore:       false,
			})
			go server.Start()
			time.Sleep(time.Second * 3)

			tt.args.metric.Value = (&tt.args.randValue)
			a := New(agentConfig.Config{
				ServerAddress:  tt.fields.agenturl,
				PollInterval:   tt.fields.pollInterval,
				ReportInterval: tt.fields.reportInterval,
				Logger:         logger.Sugar(),
				Batch:          true,
			})
			a.logger.Info("Agent created successfully", a)
		})
	}
}
