package devices

import (
	"bosch-data-exporter/internal/conf"
	"bosch-data-exporter/internal/rooms"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type DeviceResponse struct {
	Type             string   `json:"@type"`
	RootDeviceId     string   `json:"rootDeviceId"`
	Id               string   `json:"id"`
	DeviceServiceIds []string `json:"deviceServiceIds"`
	Manufacturer     string   `json:"manufacturer"`
	RoomId           string   `json:"roomId"`
	DeviceModel      string   `json:"deviceModel"`
	Serial           string   `json:"serial"`
	Profile          string   `json:"profile"`
	Name             string   `json:"name"`
	Status           string   `json:"status"`
	ChildDeviceIds   []string `json:"childDeviceIds"`
}

type Device struct {
	Type        string
	Id          string
	DeviceModel string
	Serial      string
	Name        string
	Profile     string
	Room        *rooms.Room
}

var currentRooms []*rooms.Room
var lock = sync.Mutex{}

var DefaultDevice = &Device{
	Type:        "default",
	Id:          "",
	DeviceModel: "none",
	Serial:      "",
	Name:        "default",
	Profile:     "",
	Room:        rooms.DefaultRoom,
}

func GetDevices(client *http.Client, roomChan <-chan []*rooms.Room, config *conf.Config) <-chan []*Device {
	lock.Lock()
	currentRooms = <-roomChan
	lock.Unlock()
	go func() {
		for r := range roomChan {
			lock.Lock()
			currentRooms = r
			lock.Unlock()
		}
	}()
	output := make(chan []*Device)
	go pipeSingle(client, output, config)
	ticker := time.NewTicker(time.Minute * time.Duration(config.DeviceUpdateInterval))
	go func() {
		defer close(output)
		for {
			select {
			case <-ticker.C:
				go pipeSingle(client, output, nil)
			}
		}
	}()
	return output
}

func pipeSingle(client *http.Client, output chan []*Device, config *conf.Config) {
	if devices, err := getSingle(client, config); err != nil {
		log.Err(err).Msg("Error getting devices")
	} else {
		devices = append(devices, DefaultDevice)
		output <- devices
	}
}

func getSingle(client *http.Client, config *conf.Config) ([]*Device, error) {
	log.Debug().Msg("Getting devices...")
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/smarthome/devices", config.BaseURL),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
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
		log.Info().
			Str("id", jsonBody[i].Id).
			Str("name", jsonBody[i].Name).
			Str("type", jsonBody[i].Type).
			Str("status", jsonBody[i].Status).
			Str("roomId", jsonBody[i].RoomId).
			Str("serial", jsonBody[i].Serial).
			Msg("Got device")
		room := getRoom(jsonBody[i].RoomId)
		if room == nil {
			log.Error().
				Str("roomID", jsonBody[i].RoomId).
				Str("deviceId", jsonBody[i].Id).
				Str("deviceName", jsonBody[i].Name).
				Msg("Cannot find room")
			continue
		}
		devices = append(
			devices,
			&Device{
				Type:        jsonBody[i].Type,
				Id:          jsonBody[i].Id,
				DeviceModel: jsonBody[i].DeviceModel,
				Serial:      jsonBody[i].Serial,
				Name:        jsonBody[i].Name,
				Profile:     jsonBody[i].Profile,
				Room:        room,
			},
		)
	}
	return devices, nil
}

func getRoom(id string) *rooms.Room {
	lock.Lock()
	defer lock.Unlock()
	for _, r := range currentRooms {
		if r.Id == id {
			return r
		}
	}
	return nil
}
