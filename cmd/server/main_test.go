package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGaugeUpdateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	db := MemStorage{
		Gdb: make(map[string]gauge),
		Cdb: make(map[string]counter),
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
				url: "/update/gauge/RandomValue/1232131",
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			request := httptest.NewRequest(http.MethodPost, test.args.url, nil)
			// создаём новый Recorder
			w := httptest.NewRecorder()

			db.GaugeUpdateHandler(w, request)

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

func TestCounterUpdateHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}
	db := MemStorage{
		Gdb: make(map[string]gauge),
		Cdb: make(map[string]counter),
	}
	tests := []struct {
		name string
		args struct {
			url string
		}
		want want
	}{
		{
			name: "Counter test #1",
			args: struct{ url string }{
				url: "/update/counter/PollCount/123",
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
		},
		{
			name: "Counter test #2",
			args: struct{ url string }{
				url: "/update/counter/PollCount/-123",
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
		},
		{
			name: "Counter test #3",
			args: struct{ url string }{
				url: "/update/counter/PollCount/1111111111",
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			request := httptest.NewRequest(http.MethodPost, test.args.url, nil)
			// создаём новый Recorder
			w := httptest.NewRecorder()

			db.CounterUpdateHandler(w, request)

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
