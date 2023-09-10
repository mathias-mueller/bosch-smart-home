package devices

import (
	"bosch-data-exporter/internal/rooms"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const body = "[" +
	"{" +
	"\"@type\": \"device\"," +
	"\"rootDeviceId\": \"64-da-a0-10-84-ad\"," +
	"\"id\": \"roomClimateControl_hz_4\"," +
	"\"deviceServiceIds\": [" +
	"\"TemperatureLevelConfiguration\"," +
	"\"ThermostatSupportedControlMode\"," +
	"\"RoomClimateControl\"," +
	"\"TemperatureLevel\"" +
	"]," +
	"\"manufacturer\": \"BOSCH\"," +
	"\"roomId\": \"hz_4\"," +
	"\"deviceModel\": \"ROOM_CLIMATE_CONTROL\"," +
	"\"serial\": \"roomClimateControl_hz_4\"," +
	"\"iconId\": \"icon_room_bedroom_rcc\"," +
	"\"name\": \"-RoomClimateControl-\"," +
	"\"status\": \"AVAILABLE\"," +
	"\"childDeviceIds\": [" +
	"\"hdm:HomeMaticIP:3014F711A000005D58595588\"" +
	"]" +
	"}]"

type mockHTTPClient struct {
	mockDo func(r *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(request *http.Request) (*http.Response, error) {
	return m.mockDo(request)
}

type mockCurrentRooms struct {
	mockGet func() []*rooms.Room
}

func (m *mockCurrentRooms) Get() []*rooms.Room {
	return m.mockGet()
}

func TestDevicePolling_Get(t *testing.T) {
	type args struct {
		body   string
		status int
		error  error
		rooms  []*rooms.Room
	}

	tests := []struct {
		name    string
		args    args
		want    []*Device
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "simple",
			args: args{
				body: body,
				rooms: []*rooms.Room{{
					ID:   "hz_4",
					Name: "Schlafzimmer",
				}},
				status: 200,
				error:  nil,
			},
			want: []*Device{
				{
					Type:        "device",
					ID:          "roomClimateControl_hz_4",
					DeviceModel: "ROOM_CLIMATE_CONTROL",
					Serial:      "roomClimateControl_hz_4",
					Name:        "-RoomClimateControl-",
					Profile:     "",
					Room: &rooms.Room{
						ID:   "hz_4",
						Name: "Schlafzimmer",
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "no json response",
			args: args{body: "fooBar",
				rooms: []*rooms.Room{{
					ID:   "hz_4",
					Name: "Schlafzimmer",
				}},
				status: 200,
				error:  nil,
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "status 400",
			args: args{
				body: body,
				rooms: []*rooms.Room{{
					ID:   "hz_4",
					Name: "Schlafzimmer",
				}},
				status: 503,
				error:  nil,
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "unstarted",
			args: args{
				body: body,
				rooms: []*rooms.Room{{
					ID:   "hz_4",
					Name: "Schlafzimmer",
				}},
				status: 200,
				error:  errors.New("test"),
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "no room",
			args: args{
				body:   body,
				rooms:  []*rooms.Room{},
				status: 200,
				error:  nil,
			},
			want:    []*Device{},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DevicePolling{
				client: &mockHTTPClient{mockDo: func(r *http.Request) (*http.Response, error) {
					return &http.Response{
						Body:       io.NopCloser(strings.NewReader(tt.args.body)),
						StatusCode: tt.args.status,
					}, tt.args.error
				}},
				updateInterval: 1,
				baseURL:        "http://localhost:8080",
				rooms: &mockCurrentRooms{
					mockGet: func() []*rooms.Room {
						return tt.args.rooms
					},
				},
			}

			got, err := r.Get()
			if !tt.wantErr(t, err, fmt.Sprintf("getSingle(%s)", tt.name)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getSingle(%s)", tt.name)
		})
	}
}
