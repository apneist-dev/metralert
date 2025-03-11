package agent

import (
	. "metralert/internal/metrics"
	"metralert/internal/server"
	"metralert/internal/storage"
	"net/http"
	"reflect"
	"runtime"
	"testing"

	_ "github.com/stretchr/testify/assert"
)

// func TestClient_SendPost(t *testing.T) {
// 	type fields struct {
// 		url            string
// 		pollInterval   int
// 		reportInterval int
// 	}
// 	type args struct {
// 		endpoint string
// 	}
// 	type want struct {
// 		code        int
// 		response    string
// 		contentType string
// 	}

// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		args    args
// 		want    want
// 		wantErr bool
// 	}{
// 		{
// 			name: "SendPost test #1",
// 			fields: fields{
// 				url:            "localhost:8080",
// 				pollInterval:   2,
// 				reportInterval: 10,
// 			},
// 			args: args{
// 				endpoint: "/update/gauge/RandomValue/1232131",
// 			},
// 			want: want{
// 				code:        200,
// 				response:    `{"status":"ok"}`,
// 				contentType: "text/plain",
// 			},
// 			wantErr: false,
// 		},
// 	}
// 	serverurl := "localhost:8080"
// 	go server.NewServer(serverurl)

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			c := NewClient(
// 				tt.fields.url,
// 				tt.fields.pollInterval,
// 				tt.fields.reportInterval)
// 			got, err := c.SendPost(tt.args.endpoint)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("Client.SendPost() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			assert.Equal(t, tt.want.code, got.StatusCode)
// 			defer got.Body.Close()
// 		})
// 	}
// }

//Этот тест пришлось закомментировать, тк он стал выполняться бесконечно

// func TestClient_SendAllMetrics(t *testing.T) {
// 	type fields struct {
// 		url            string
// 		endpoints      []string
// 		pollInterval   int
// 		reportInterval int
// 	}
// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		wantErr bool
// 	}{
// 		{
// 			name: "SendAllMetrics #1 err",
// 			fields: fields{
// 				url:            "localhost:8080",
// 				endpoints:      []string{"/update/gauge/RandomValue/1232131"},
// 				pollInterval:   2,
// 				reportInterval: 10,
// 			},
// 			wantErr: true,
// 		},
// 	}

// 	serverurl := "localhost:8080"
// 	go server.NewServer(serverurl)

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			endpoints := tt.fields.endpoints
// 			c := NewClient(
// 				tt.fields.url,
// 				tt.fields.pollInterval,
// 				tt.fields.reportInterval)
// 			if err := c.SendAllMetrics(); (err != nil) != tt.wantErr {
// 				t.Errorf("For endpoints %s Client.SendAllMetrics() error = %v, wantErr %v", endpoints, err, tt.wantErr)
// 			}
// 		})
// 	}
// }

// func TestCollectMetric(t *testing.T) {
// 	tests := []struct {
// 		name      string
// 		endpoints []string
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	serverurl := "localhost:8080"
// 	go server.NewServer(serverurl)

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			var endpoints = tt.endpoints
// 			CollectMetric()
// 			time.Sleep(15 * time.Second)
// 			if len(endpoints) == 0 {
// 				t.Error("CollectMetrics collected 0 metrics")
// 			}
// 		})
// 	}
// }

func TestAgent_SendPost(t *testing.T) {
	type fields struct {
		url            string
		pollInterval   int
		reportInterval int
		pollCount      Counter
		endpoints      []string
		rtm            runtime.MemStats
		client         http.Client
	}
	type args struct {
		endpoint string
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
				url:            "localhost:8080",
				pollInterval:   2,
				reportInterval: 10,
				pollCount:      12,
			},
			args: args{
				endpoint: "/update/gauge/RandomValue/1232131",
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
				url:            tt.fields.url,
				pollInterval:   tt.fields.pollInterval,
				reportInterval: tt.fields.reportInterval,
				pollCount:      tt.fields.pollCount,
				endpoints:      tt.fields.endpoints,
				rtm:            tt.fields.rtm,
				client:         tt.fields.client,
			}
			storage := storage.New()
			server := server.New(tt.fields.url, &storage)
			go server.Start()

			got, err := a.SendPost(tt.args.endpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("Agent.SendPost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.StatusCode, tt.want.code) {
				t.Errorf("Agent.SendPost() = %v, want %v", got.StatusCode, tt.want.code)
			}
		})
	}
}
