package events

import (
	"bosch-data-exporter/internal/devices"
	"errors"
	"net/http"
	"reflect"
	"testing"
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
		wantErr bool
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
				exporter: &mockExporter{},
			},
			want:    nil,
			wantErr: true,
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
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}
