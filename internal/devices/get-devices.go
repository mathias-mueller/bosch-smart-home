package devices

import (
	"bosch-data-exporter/internal/conf"
	"bosch-data-exporter/internal/rooms"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

type DeviceResponse struct {
	Type             string   `json:"@type"`
	RootDeviceID     string   `json:"rootDeviceId"`
	ID               string   `json:"id"`
	DeviceServiceIds []string `json:"deviceServiceIds"`
	Manufacturer     string   `json:"manufacturer"`
	RoomID           string   `json:"roomId"`
	DeviceModel      string   `json:"deviceModel"`
	Serial           string   `json:"serial"`
	Profile          string   `json:"profile"`
	Name             string   `json:"name"`
	Status           string   `json:"status"`
	ChildDeviceIds   []string `json:"childDeviceIds"`
}

type Device struct {
	Type        string
	ID          string
	DeviceModel string
	Serial      string
	Name        string
	Profile     string
	Room        *rooms.Room
}

type DevicePolling struct {
	currentRooms    []*rooms.Room
	lock            sync.Mutex
	client          *http.Client
	config          *conf.Config
	reqDurationHist prometheus.Histogram
}

func DefaultDevice() *Device {
	return &Device{
		Type:        "default",
		ID:          "",
		DeviceModel: "none",
		Serial:      "",
		Name:        "default",
		Profile:     "",
		Room:        rooms.DefaultRoom(),
	}
}

func NewDevicePolling(client *http.Client, config *conf.Config) *DevicePolling {
	return &DevicePolling{
		lock:   sync.Mutex{},
		client: client,
		config: config,
		reqDurationHist: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "bosch_device_poll_duration",
			Help: "Duration of the GET Device call",
		}),
	}
}

func (d *DevicePolling) GetDevices(roomChan <-chan []*rooms.Room) <-chan []*Device {
	d.lock.Lock()
	d.currentRooms = <-roomChan
	d.lock.Unlock()
	go func() {
		for r := range roomChan {
			d.lock.Lock()
			d.currentRooms = r
			d.lock.Unlock()
		}
	}()
	output := make(chan []*Device)
	go d.pipeSingle(output)
	ticker := time.NewTicker(time.Minute * time.Duration(d.config.DeviceUpdateInterval))
	go func() {
		defer close(output)
		for range ticker.C {
			go d.pipeSingle(output)
		}
	}()
	return output
}

func (d *DevicePolling) pipeSingle(output chan []*Device) {
	timer := prometheus.NewTimer(d.reqDurationHist)
	defer timer.ObserveDuration()
	devices, err := d.getSingle()
	if err != nil {
		log.Err(err).Msg("Error getting devices")
		return
	}
	devices = append(devices, DefaultDevice())
	output <- devices
}

func (d *DevicePolling) getSingle() ([]*Device, error) {
	log.Debug().Msg("Getting devices...")
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/smarthome/devices", d.config.BoschConfig.BaseURL),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		e := resp.Body.Close()
		if e != nil {
			log.Err(e).Msg("Error closing response body")
		}
	}()
	buf := &bytes.Buffer{}
	if _, e := buf.ReadFrom(resp.Body); e != nil {
		return nil, e
	}
	body := buf.Bytes()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status of get geive call is not %d, but %d", http.StatusOK, resp.StatusCode)
	}
	var jsonBody []DeviceResponse

	if e := json.Unmarshal(body, &jsonBody); e != nil {
		return nil, e
	}

	devices := make([]*Device, 0)

	for i := range jsonBody {
		log.Debug().
			Str("id", jsonBody[i].ID).
			Str("name", jsonBody[i].Name).
			Str("type", jsonBody[i].Type).
			Str("status", jsonBody[i].Status).
			Str("roomId", jsonBody[i].RoomID).
			Str("serial", jsonBody[i].Serial).
			Msg("Got device")
		room := d.getRoom(jsonBody[i].RoomID)
		if room == nil {
			log.Error().
				Str("roomID", jsonBody[i].RoomID).
				Str("deviceId", jsonBody[i].ID).
				Str("deviceName", jsonBody[i].Name).
				Msg("Cannot find room")
			continue
		}
		devices = append(
			devices,
			&Device{
				Type:        jsonBody[i].Type,
				ID:          jsonBody[i].ID,
				DeviceModel: jsonBody[i].DeviceModel,
				Serial:      jsonBody[i].Serial,
				Name:        jsonBody[i].Name,
				Profile:     jsonBody[i].Profile,
				Room:        room,
			},
		)
	}
	log.Info().Int("number", len(devices)).Msg("Got devices")
	return devices, nil
}

func (d *DevicePolling) getRoom(id string) *rooms.Room {
	d.lock.Lock()
	defer d.lock.Unlock()
	for _, r := range d.currentRooms {
		if r.ID == id {
			return r
		}
	}
	return nil
}
