package agent

import (
	"metralert/internal/metrics"
	"metralert/internal/server"
	"metralert/internal/storage"
	"net/http"
	"reflect"
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
			a := &Agent{
				url:              tt.fields.agenturl,
				pollInterval:     tt.fields.pollInterval,
				reportInterval:   tt.fields.reportInterval,
				pollCount:        tt.fields.pollCount,
				memoryStatistics: tt.fields.memoryStatistics,
				rtm:              tt.fields.rtm,
				client:           tt.fields.client,
			}
			logger, _ := zap.NewDevelopment()
			sugar := logger.Sugar()
			storage := storage.New("internal/storage/metrics_database.json", false, logger.Sugar())

			server := server.New(tt.fields.serverurl, storage, sugar)
			go server.Start()
			time.Sleep(time.Second * 3)

			tt.args.metric.Value = (&tt.args.randValue)
			got, err := a.SendPost(tt.args.metric)
			if (err != nil) != tt.wantErr {
				t.Errorf("Agent.SendPost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer got.Body.Close()

			if !reflect.DeepEqual(got.StatusCode, tt.want.code) {
				t.Errorf("Agent.SendPost() = %v, want %v", got.StatusCode, tt.want.code)
			}
		})
	}
}
