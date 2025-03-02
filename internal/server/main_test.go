package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorage_GaugeUpdateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	type args struct {
		metricname  string
		metricvalue string
		url         string
		method      string
	}
	db := MemStorage{
		Gdb: make(map[string]gauge),
		Cdb: make(map[string]counter),
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Gauge test #1 StatusOK",
			args: args{
				metricname:  "TestGauge",
				metricvalue: "123.232323",
				url:         "/update/gauge/{metricname}/{metricvalue}",
				method:      http.MethodPost,
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain; charset=utf-8",
			},
		},
		// {
		// 	name: "Gauge test #1 StatusOK",
		// 	args: args{
		// 		url:    "/update/gauge/TestGauge/123.232323",
		// 		method: http.MethodPost,
		// 	},
		// 	want: want{
		// 		code:        200,
		// 		response:    `{"status":"ok"}`,
		// 		contentType: "text/plain",
		// 	},
		// },
		{
			name: "Gauge test #2 StatusOK",
			args: args{
				metricname:  "TestGauge",
				metricvalue: "123.231",
				url:         "/update/gauge/{metricname}/{metricvalue}",
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
			args: args{
				metricname:  "TestGauge",
				metricvalue: "-12s3.4343",
				url:         "/update/gauge/{metricname}/{metricvalue}",
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
			args: args{
				metricname:  "TestGauge",
				metricvalue: "-1.323223/23432/2323",
				url:         "/update/gauge/{metricname}/{metricvalue}",
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

			request := httptest.NewRequest(tt.args.method, tt.args.url, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("metricname", tt.args.metricname)
			rctx.URLParams.Add("metricvalue", tt.args.metricvalue)
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))
			// создаём новый Recorder
			w := httptest.NewRecorder()

			db.GaugeUpdateHandler(w, request)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			_, err := io.ReadAll(res.Body)
			require.NoError(t, err)

		})
	}
}

func TestMemStorage_CounterUpdateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	type args struct {
		metricname  string
		metricvalue string
		url         string
		method      string
	}
	db := MemStorage{
		Gdb: make(map[string]gauge),
		Cdb: make(map[string]counter),
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Counter test #1 StatusOK",
			args: args{
				metricname:  "PollCount",
				metricvalue: "123",
				url:         "/update/counter/{metricname}/{metricvalue}",
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
			args: args{
				metricname:  "PollCount",
				metricvalue: "-123.4343",
				url:         "/update/counter/{metricname}/{metricvalue}",
				method:      http.MethodPost,
			},
			want: want{
				code:        400,
				response:    `{"status":"Bad Request"}`,
				contentType: "text/plain",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			request := httptest.NewRequest(tt.args.method, tt.args.url, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("metricname", tt.args.metricname)
			rctx.URLParams.Add("metricvalue", tt.args.metricvalue)
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))
			// создаём новый Recorder
			w := httptest.NewRecorder()

			db.CounterUpdateHandler(w, request)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			_, err := io.ReadAll(res.Body)

			require.NoError(t, err)
		})
	}
}

func TestPostMainHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	tests := []struct {
		name string
		args struct {
			url string
		}
		want want
	}{
		{
			name: "MainHandler test #1",
			args: struct{ url string }{
				url: "/upd7a",
			},
			want: want{
				code:        400,
				response:    `{"status":"Bad Request"}`,
				contentType: "text/plain",
			},
		},
		{
			name: "MainHandler test #2",
			args: struct{ url string }{
				url: "/upd7ate/cou7nter/PollCount/123",
			},
			want: want{
				code:        400,
				response:    `{"status":"Bad Request"}`,
				contentType: "text/plain",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			request := httptest.NewRequest(http.MethodPost, test.args.url, nil)
			// создаём новый Recorder
			w := httptest.NewRecorder()

			PostMainHandler(w, request)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, test.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			_, err := io.ReadAll(res.Body)

			require.NoError(t, err)
		})
	}
}

func TestMemStorage_SaveCounterMetric(t *testing.T) {
	type fields struct {
		Gdb map[string]gauge
		Cdb map[string]counter
	}
	type args struct {
		metricname  string
		metricvalue counter
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   MemStorage
	}{
		{
			name: "SaveCounterMetric test #1",
			fields: fields{
				Gdb: make(map[string]gauge),
				Cdb: make(map[string]counter),
			},
			args: args{
				metricname:  "NewShinyMetric",
				metricvalue: 123,
			},
			want: MemStorage{
				Gdb: map[string]gauge{},
				Cdb: map[string]counter{"NewShinyMetric": 123},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := MemStorage{
				Gdb: tt.fields.Gdb,
				Cdb: tt.fields.Cdb,
			}
			db.SaveCounterMetric(tt.args.metricname, tt.args.metricvalue)
			assert.Equal(t, db, tt.want)
		})
	}
}

func TestMemStorage_SaveGaugeMetric(t *testing.T) {
	type fields struct {
		Gdb map[string]gauge
		Cdb map[string]counter
	}
	type args struct {
		metricname  string
		metricvalue gauge
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   MemStorage
	}{
		{
			name: "SaveGaugeMetric test #1",
			fields: fields{
				Gdb: make(map[string]gauge),
				Cdb: make(map[string]counter),
			},
			args: args{
				metricname:  "NewShinyGaugeMetric",
				metricvalue: 123.23232332,
			},
			want: MemStorage{
				Gdb: map[string]gauge{"NewShinyGaugeMetric": 123.23232332},
				Cdb: map[string]counter{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := MemStorage{
				Gdb: tt.fields.Gdb,
				Cdb: tt.fields.Cdb,
			}
			db.SaveGaugeMetric(tt.args.metricname, tt.args.metricvalue)
			assert.Equal(t, db, tt.want)
		})
	}
}

func TestMemStorage_GetMetricHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	type args struct {
		metrictype string
		metricname string
		url        string
		method     string
	}
	db := MemStorage{
		Gdb: make(map[string]gauge),
		Cdb: make(map[string]counter),
	}
	tests := []struct {
		name string
		args args
		want want
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
		{
			name: "Get metric #2 StatusNotFound",
			args: args{
				metricname: "PollCount222",
				metrictype: "Gauge",
				url:        "/value/{metrictype}/{metricname}",
				method:     http.MethodPost,
			},
			want: want{
				code:        404,
				response:    `{"status":"Not Found"}`,
				contentType: "text/plain",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			request := httptest.NewRequest(tt.args.method, tt.args.url, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("metricname", tt.args.metricname)
			rctx.URLParams.Add("metrictype", tt.args.metrictype)
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))
			// создаём новый Recorder
			w := httptest.NewRecorder()

			db.GetMetricHandler(w, request)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			_, err := io.ReadAll(res.Body)

			require.NoError(t, err)
		})
	}
}
