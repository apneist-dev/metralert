package agent

import (
	"testing"
	"time"

	"metralert/internal/server"

	"github.com/stretchr/testify/assert"
)

func TestClient_SendPost(t *testing.T) {
	type fields struct {
		url string
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
			name:   "SendPost test #1",
			fields: fields{url: "http://localhost:8080"},
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
	serverurl := "http://localhost:8080"
	go server.NewServer(serverurl)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := Client{
				url: test.fields.url,
			}
			got, err := c.SendPost(test.args.endpoint)
			if (err != nil) != test.wantErr {
				t.Errorf("Client.SendPost() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			assert.Equal(t, test.want.code, got.StatusCode)
			defer got.Body.Close()
		})
	}
}

func TestClient_SendAllMetrics(t *testing.T) {
	type fields struct {
		url       string
		endpoints []string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "SendAllMetrics #1 err",
			fields: fields{
				url:       "htt1p://ll:8080",
				endpoints: []string{"/update/gauge/RandomValue/1232131"},
			},
			wantErr: true,
		},
	}

	serverurl := "http://localhost:8080"
	go server.NewServer(serverurl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoints := tt.fields.endpoints
			c := Client{
				url: tt.fields.url,
			}
			if err := c.SendAllMetrics(); (err != nil) != tt.wantErr {
				t.Errorf("For endpoints %s Client.SendAllMetrics() error = %v, wantErr %v", endpoints, err, tt.wantErr)
			}
		})
	}
}

func TestCollectMetric(t *testing.T) {
	tests := []struct {
		name      string
		endpoints []string
	}{
		// TODO: Add test cases.
	}
	serverurl := "http://localhost:8080"
	go server.NewServer(serverurl)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var endpoints = tt.endpoints
			CollectMetric()
			time.Sleep(15 * time.Second)
			if len(endpoints) == 0 {
				t.Error("CollectMetrics collected 0 metrics")
			}
		})
	}
}
