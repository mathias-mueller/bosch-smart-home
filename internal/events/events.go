package events

import (
	"bosch-data-exporter/internal/conf"
	"bosch-data-exporter/internal/devices"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"
)

type pollRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type pollResponse struct {
	Result  []pollResponseResult `json:"result"`
	Jsonrpc string               `json:"jsonrpc"`
	Error   pollResponseError    `json:"error"`
}

type pollResponseResult struct {
	Path     string                 `json:"path"`
	Type     string                 `json:"@type"`
	ID       string                 `json:"id"`
	State    map[string]interface{} `json:"state"`
	DeviceID string                 `json:"deviceId"`
}

type pollResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Event struct {
	ID     string
	Type   string
	Device *devices.Device
	State  map[string]interface{}
}

type SmartHomeEventPolling struct {
	currentDevices []*devices.Device
	pollID         string
	lock           sync.Mutex
	client         *http.Client
	config         *conf.Config
}

func NewSmartHomeEventPolling(client *http.Client, config *conf.Config) *SmartHomeEventPolling {
	return &SmartHomeEventPolling{
		lock:   sync.Mutex{},
		client: client,
		config: config,
	}
}

func (s *SmartHomeEventPolling) Start(
	pollIDchan <-chan string,
	deviceChan <-chan []*devices.Device,
) <-chan *Event {
	s.lock.Lock()
	s.currentDevices = <-deviceChan
	s.pollID = <-pollIDchan
	s.lock.Unlock()
	go func() {
		for d := range deviceChan {
			s.lock.Lock()
			s.currentDevices = d
			s.lock.Unlock()
		}
	}()
	go func() {
		for id := range pollIDchan {
			s.lock.Lock()
			s.pollID = id
			s.lock.Unlock()
		}
	}()
	output := make(chan *Event)
	go func() {
		defer close(output)
		var err error
		for err == nil {
			var events []*Event
			events, err = s.Get()
			go func() {
				for _, e := range events {
					if e != nil {
						output <- e
					}
				}
			}()
		}
		log.Err(err).Msg("Error while polling data")
	}()
	return output
}

func (s *SmartHomeEventPolling) Get() ([]*Event, error) {
	pollID := s.pollID
	log.Info().
		Str("pollID", pollID).
		Msg("Polling for changes")
	requestBody := []pollRequest{
		{
			Jsonrpc: "2.0",
			Method:  "RE/longPoll",
			Params:  []interface{}{pollID, 30},
		},
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	shcPollURL := fmt.Sprintf("%s/remote/json-rpc", s.config.BoschConfig.BaseURL)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		shcPollURL,
		bytes.NewReader(requestBodyBytes),
	)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
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
	log.Trace().
		Str("pollID", pollID).
		Int("status", resp.StatusCode).
		Bytes("body", body).
		Msg("Got poll response")

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status of get geive call is not %d, but %d", http.StatusOK, resp.StatusCode)
	}

	var jsonBody []pollResponse

	if e := json.Unmarshal(body, &jsonBody); e != nil {
		return nil, e
	}
	shcBody := jsonBody[0]

	if shcBody.Error.Message != "" {
		return nil, fmt.Errorf("poll returned error: %s", jsonBody[0].Error.Message)
	}
	events := make([]*Event, 0)

	for i := range shcBody.Result {
		event := &shcBody.Result[i]
		log.Info().
			Str("deviceID", event.DeviceID).
			Str("id", event.ID).
			Str("path", event.Path).
			Str("type", event.Type).
			Interface("state", event.State).
			Msg("poll result")
		device := s.getDevice(event.DeviceID)
		if device == nil {
			device = devices.DefaultDevice()
		}
		events = append(
			events,
			&Event{
				ID:     event.ID,
				Type:   event.Type,
				Device: device,
				State:  event.State,
			},
		)
	}

	return events, nil
}

func (s *SmartHomeEventPolling) getDevice(id string) *devices.Device {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, d := range s.currentDevices {
		if d.ID == id {
			return d
		}
	}
	return nil
}
