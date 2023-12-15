package events

import (
	"bosch-data-exporter/internal/devices"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockDevices struct {
	mockGet func() []*devices.Device
}

func (m *mockDevices) Get() []*devices.Device {
	return m.mockGet()
}

type mockPollID struct {
	mockGet func() string
}

func (m *mockPollID) Get() string {
	return m.mockGet()
}

type mockClient struct {
	mockDo func(*http.Request) (*http.Response, error)
}

func (m *mockClient) Do(request *http.Request) (*http.Response, error) {
	return m.mockDo(request)
}

type mockExporter struct {
	mockExport func(*Event)
}

func (m mockExporter) Export(event *Event) {
	m.mockExport(event)
}

func TestSmartHomeEventPolling_Get(t *testing.T) {
	dev0 := &devices.Device{
		Type: "roomClimateControl",
		ID:   "roomClimateControl_hz_5",
		Name: "roomClimateControl",
		Room: nil,
	}
	type fields struct {
		devices  []*devices.Device
		pollID   string
		client   httpClient
		exporter exporter
	}
	tests := []struct {
		name    string
		fields  fields
		want    []*Event
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "error",
			fields: fields{
				devices: make([]*devices.Device, 0),
				pollID:  "poll-id",
				client: &mockClient{
					mockDo: func(request *http.Request) (*http.Response, error) {
						return nil, errors.New("test")
					},
				},
				exporter: &mockExporter{
					func(event *Event) {
						assert.Fail(t, "exporter should not be called")
					},
				},
			},
			want:    nil,
			wantErr: assert.Error,
		},
		{
			name: "no results",
			fields: fields{
				devices: make([]*devices.Device, 0),
				pollID:  "poll-id",
				client: &mockClient{
					mockDo: func(request *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader("[{\"result\":[],\"jsonrpc\":\"2.0\"}]\n")),
						}, nil
					},
				},
				exporter: &mockExporter{
					func(event *Event) {
						assert.Fail(t, "exporter should not be called")
					},
				},
			},
			want:    make([]*Event, 0),
			wantErr: assert.NoError,
		},
		{
			name: "results",
			fields: fields{
				devices: []*devices.Device{dev0},
				pollID:  "poll-id",
				client: &mockClient{
					mockDo: func(request *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body: io.NopCloser(strings.NewReader("[" +
								"{\"result\":[" +
								"{" +
								"\"path\":\"/devices/roomClimateControl_hz_5/services/TemperatureLevel\"," +
								"\"@type\":\"DeviceServiceData\"," +
								"\"id\":\"TemperatureLevel\"," +
								"\"state\":{\"@type\":\"temperatureLevelState\",\"temperature\":25}," +
								"\"deviceId\":\"roomClimateControl_hz_5\"" +
								"}],\"jsonrpc\":\"2.0\"}]\n",
							)),
						}, nil
					},
				},
				exporter: &mockExporter{
					func(event *Event) {
						assert.Fail(t, "exporter should not be called")
					},
				},
			},
			want: []*Event{
				{
					ID:     "TemperatureLevel",
					Type:   "DeviceServiceData",
					Device: dev0,
					State:  map[string]interface{}{"@type": "temperatureLevelState", "temperature": float64(25)},
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SmartHomeEventPolling{
				devices:  &mockDevices{func() []*devices.Device { return tt.fields.devices }},
				pollID:   &mockPollID{func() string { return tt.fields.pollID }},
				client:   tt.fields.client,
				baseURL:  "http://localhost:8080",
				exporter: tt.fields.exporter,
			}
			got, err := s.Get()
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
