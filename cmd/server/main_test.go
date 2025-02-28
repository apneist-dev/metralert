package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
		url    string
		method string
	}
	db := MemStorage{
		Gdb: make(map[string]gauge),
		Cdb: make(map[string]counter),
	}
	tests := []struct {
		name string
		args struct {
			url    string
			method string
		}
		want want
	}{
		{
			name: "Gauge test #1 StatusOK",
			args: args{
				url:    "/update/gauge/TestGauge/123.232323",
				method: http.MethodPost,
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
		},
		{
			name: "Gauge test #1 StatusMethodNotAllowed",
			args: args{
				url:    "/update/gauge/TestGauge/123.231",
				method: http.MethodGet,
			},
			want: want{
				code:        405,
				response:    `{"status":"Method Not Allowed"}`,
				contentType: "text/plain",
			},
		},
		{
			name: "Gauge test #2",
			args: args{
				url:    "/update/gauge/PollCount/-12s3.4343",
				method: http.MethodPost,
			},
			want: want{
				code:        400,
				response:    `{"status":"Bad Request"}`,
				contentType: "text/plain",
			},
		},
		{
			name: "Gauge test #3 StatusNotFound",
			args: args{
				url:    "/update/gauge/TestGauge/-1.323223/23432/2323",
				method: http.MethodPost,
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
		url    string
		method string
	}
	db := MemStorage{
		Gdb: make(map[string]gauge),
		Cdb: make(map[string]counter),
	}
	tests := []struct {
		name string
		args struct {
			url    string
			method string
		}
		want want
	}{
		{
			name: "Counter test #1 StatusOK",
			args: args{
				url:    "/update/counter/PollCount/123",
				method: http.MethodPost,
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
		},
		{
			name: "Counter test #1 StatusMethodNotAllowed",
			args: args{
				url:    "/update/counter/PollCount/123",
				method: http.MethodGet,
			},
			want: want{
				code:        405,
				response:    `{"status":"Method Not Allowed"}`,
				contentType: "text/plain",
			},
		},
		{
			name: "Counter test #2",
			args: args{
				url:    "/update/counter/PollCount/-123.4343",
				method: http.MethodPost,
			},
			want: want{
				code:        400,
				response:    `{"status":"Bad Request"}`,
				contentType: "text/plain",
			},
		},
		{
			name: "Counter test #3 StatusNotFound",
			args: args{
				url:    "/update/counter/PollCount/-123/23432/2323",
				method: http.MethodPost,
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

func TestMainHandler(t *testing.T) {
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
			name: "positive test #1",
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
			name: "positive test #1",
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

			MainHandler(w, request)

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
			name: "SaveCounterMetric test #1",
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
