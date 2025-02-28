package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestClientSendAllMetrics(t *testing.T) {
// 	// Dummy JSON response
// 	expected := "{'data': 'dummy'}"
// 	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, expected) }))
// 	defer svr.Close()
// 	c := NewClient(svr.URL)
// 	res, err := c.SendPost()
// 	if err != nil {
// 		t.Errorf("expected err to be nil got %v", err)
// 	}
// 	res = strings.TrimSpace(res)
// 	if res != expected {
// 		t.Errorf("expected res to be %s got %s", expected, res)
// 	}
// }

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
			name:   "positive test #1",
			fields: fields{url: "http://localhost:8080"},
			args: args{
				endpoint: "http://localhost:8080/update/gauge/RandomValue/1232131",
			},
			want: want{
				code:        200,
				response:    `{"status":"ok"}`,
				contentType: "text/plain",
			},
			wantErr: false,
		},
	}
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
		})
	}
}
