package polling

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockClient struct {
	mockDo func(*http.Request) (*http.Response, error)
}

func (m *mockClient) Do(request *http.Request) (*http.Response, error) {
	return m.mockDo(request)
}

func TestPollIDGenerator_Get(t *testing.T) {
	type fields struct {
		client  httpClient
		baseURL string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "valid",
			fields: fields{
				client: &mockClient{
					mockDo: func(request *http.Request) (*http.Response, error) {
						assert.Equal(t, "http://localhost:8080/remote/json-rpc", request.URL.String())
						assert.Equal(t, http.MethodPost, request.Method)
						buf := new(strings.Builder)
						_, e := io.Copy(buf, request.Body)
						assert.NoError(t, e)
						request.Body.Close()
						assert.JSONEq(t, "[{"+
							"\"jsonrpc\":\"2.0\","+
							"\"method\":\"RE/subscribe\","+
							"\"params\":[\"com/bosch/sh/remote/*\", null]"+
							"}]", buf.String())
						return &http.Response{
							StatusCode: http.StatusOK,
							Body: io.NopCloser(strings.NewReader("[{" +
								"\"result\":\"poll-id\"," +
								"\"jsonrpc\":\"2.0\"" +
								"}]")),
						}, nil
					},
				},
				baseURL: "http://localhost:8080",
			},
			want:    "poll-id",
			wantErr: false,
		},
		{
			name: "status error",
			fields: fields{
				client: &mockClient{
					mockDo: func(request *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusBadRequest,
							Body:       io.NopCloser(strings.NewReader("bad request")),
						}, nil
					},
				},
				baseURL: "http://localhost:8080",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "http error",
			fields: fields{
				client: &mockClient{
					mockDo: func(request *http.Request) (*http.Response, error) {
						return nil, errors.New("test")
					},
				},
				baseURL: "http://localhost:8080",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PollIDGenerator{
				client:  tt.fields.client,
				baseURL: tt.fields.baseURL,
			}
			got, err := p.Get()
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}
