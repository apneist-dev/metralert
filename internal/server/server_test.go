package server

import (
	"context"
	"io"
	"log"
	. "metralert/internal/metrics"
	"metralert/internal/storage"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServer_GetMainHandler(t *testing.T) {
	type fields struct {
		url     string
		storage *storage.MemStorage
	}
	type args struct {
		metrictype string
		metricname string
		url        string
		method     string
	}
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "Get metric test #1 StatusNotFound",
			args: args{
				metricname: "PollCount222",
				metrictype: "counter",
				url:        "/value/{metrictype}/{metricname}",
				method:     http.MethodPost,
			},
			want: want{
				code:        404,
				response:    `{"status":"Not Found"}`,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				url:     tt.fields.url,
				storage: tt.fields.storage,
			}
			request := httptest.NewRequest(tt.args.method, tt.args.url, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("metricname", tt.args.metricname)
			rctx.URLParams.Add("metrictype", tt.args.metrictype)
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))
			// создаём новый Recorder
			w := httptest.NewRecorder()

			server.GetMainHandler(w, request)
		})
	}
}

func TestServer_UpdateHandler(t *testing.T) {
	type fields struct {
		url string
		// storage *storage.MemStorage
	}
	type want struct {
		code        int
		response    string
		contentType string
	}
	type args struct {
		metrictype  string
		metricname  string
		metricvalue string
		url         string
		method      string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "Counter test #1 StatusOK",
			fields: fields{
				url: "localhost:8080",
			},
			args: args{
				metrictype:  "counter",
				metricname:  "PollCount",
				metricvalue: "123",
				url:         "/update/{metrictype}/{metricname}/{metricvalue}",
				method:      http.MethodPost,
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "Counter test #2",
			fields: fields{
				url: "localhost:8080",
			},
			args: args{
				metrictype:  "counter",
				metricname:  "PollCount",
				metricvalue: "-123.4343",
				url:         "/update/{metrictype}/{metricname}/{metricvalue}",
				method:      http.MethodPost,
			},
			want: want{
				code:        400,
				response:    `{"status":"Bad Request"}`,
				contentType: "text/plain",
			},
		},
		{
			name: "Gauge test #1 StatusOK",
			fields: fields{
				url: "localhost:8080",
			},
			args: args{
				metrictype:  "gauge",
				metricname:  "TestGauge",
				metricvalue: "123.232323",
				url:         "/update/{metrictype}/{metricname}/{metricvalue}",
				method:      http.MethodPost,
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "Gauge test #2 StatusOK",
			fields: fields{
				url: "localhost:8080",
			},
			args: args{
				metrictype:  "gauge",
				metricname:  "TestGauge",
				metricvalue: "123.231",
				url:         "/update/{metrictype}/{metricname}/{metricvalue}",
				method:      http.MethodPost,
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "Gauge test #3 BadRequest",
			fields: fields{
				url: "localhost:8080",
			},
			args: args{
				metrictype:  "gauge",
				metricname:  "TestGauge",
				metricvalue: "-12s3.4343",
				url:         "/update/{metrictype}/{metricname}/{metricvalue}",
				method:      http.MethodPost,
			},
			want: want{
				code:        400,
				response:    `{"status":"Bad Request"}`,
				contentType: "text/plain",
			},
		},
		{
			name: "Gauge test #3 BadRequest",
			fields: fields{
				url: "localhost:8080",
			},
			args: args{
				metrictype:  "gauge",
				metricname:  "TestGauge",
				metricvalue: "-1.323223/23432/2323",
				url:         "/update/{metrictype}/{metricname}/{metricvalue}",
				method:      http.MethodPost,
			},
			want: want{
				code:        400,
				response:    `{"status":"BadRequest"}`,
				contentType: "text/plain",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := zap.NewDevelopment()
			sugar := logger.Sugar()
			storage := storage.New()
			server := New(tt.fields.url, &storage, sugar)
			log.Printf("Запущен сервер с адресом %s", tt.fields.url)
			// server.Start()

			request := httptest.NewRequest(tt.args.method, tt.args.url, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("metrictype", tt.args.metrictype)
			rctx.URLParams.Add("metricname", tt.args.metricname)
			rctx.URLParams.Add("metricvalue", tt.args.metricvalue)
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))
			// создаём новый Recorder
			w := httptest.NewRecorder()
			server.UpdateHandler(w, request)

			res := w.Result()
			defer res.Body.Close()
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			_, err := io.ReadAll(res.Body)

			require.NoError(t, err)
		})
	}
}

func TestServer_SaveGaugeMetric(t *testing.T) {
	type fields struct {
		url string
		// storage *storage.MemStorage
	}
	type args struct {
		metricname  string
		metricvalue string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   storage.MemStorage
	}{
		{
			name: "SaveGaugeMetric test #1",
			fields: fields{
				url: "localhost:8080",
			},
			args: args{
				metricname:  "NewShinyGaugeMetric",
				metricvalue: "123.23232332",
			},
			want: storage.MemStorage{
				Gaugedb:   map[string]Gauge{"NewShinyGaugeMetric": 123.23232332},
				Counterdb: map[string]Counter{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := zap.NewDevelopment()
			sugar := logger.Sugar()
			storage := storage.New()
			server := New(tt.fields.url, &storage, sugar)
			// server.Start()

			// создаём новый Recorder
			w := httptest.NewRecorder()

			server.SaveGaugeMetric(w, tt.args.metricname, tt.args.metricvalue)

		})
	}
}
