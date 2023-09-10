package rooms

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockHTTPClient struct {
	mockDo func(r *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(request *http.Request) (*http.Response, error) {
	return m.mockDo(request)
}

func TestRoomPolling_Get(t *testing.T) {
	type args struct {
		body   string
		status int
		error  error
	}
	tests := []struct {
		name    string
		args    args
		want    []*Room
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "simple",
			args: args{body: "[{" +
				"\"@type\": \"room\"," +
				"\"id\": \"hz_2\"," +
				"\"iconId\": \"icon_room_office\"," +
				"\"name\": \"Büro\"," +
				"\"extProperties\": {\"humidity\": \"69.0\"}" +
				"}]",
				status: 200,
				error:  nil,
			},
			want: []*Room{
				{
					ID:   "hz_2",
					Name: "Büro",
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "no json response",
			args: args{body: "fooBar",
				status: 200,
				error:  nil,
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "status 400",
			args: args{
				body: "[{" +
					"\"@type\": \"room\"," +
					"\"id\": \"hz_2\"," +
					"\"iconId\": \"icon_room_office\"," +
					"\"name\": \"Büro\"," +
					"\"extProperties\": {\"humidity\": \"69.0\"}" +
					"}]",
				status: 503,
				error:  nil,
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "http error",
			args: args{
				body: "[{" +
					"\"@type\": \"room\"," +
					"\"id\": \"hz_2\"," +
					"\"iconId\": \"icon_room_office\"," +
					"\"name\": \"Büro\"," +
					"\"extProperties\": {\"humidity\": \"69.0\"}" +
					"}]",
				status: 200,
				error:  errors.New("test"),
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RoomPolling{
				client: &mockHTTPClient{mockDo: func(r *http.Request) (*http.Response, error) {
					assert.Equal(t, r.URL.String(), "http://localhost:8080/smarthome/rooms")
					assert.Equal(t, r.Method, http.MethodGet)
					return &http.Response{
						Body:       io.NopCloser(strings.NewReader(tt.args.body)),
						StatusCode: tt.args.status,
					}, tt.args.error
				}},
				updateInterval:  1,
				baseURL:         "http://localhost:8080",
				reqDurationHist: nil,
			}
			got, err := r.Get()
			if !tt.wantErr(t, err, fmt.Sprintf("getSingle(%s)", tt.name)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getSingle(%s)", tt.name)
		})
	}
}
