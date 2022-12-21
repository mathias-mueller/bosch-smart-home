package polling

import (
	"bosch-data-exporter/internal/conf"
	"bosch-data-exporter/internal/devices"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type pollRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type pollSubscribeResponse struct {
	Result  string `json:"result"`
	Jsonrpc string `json:"jsonrpc"`
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

type Event struct {
	ID     string
	Type   string
	Device *devices.Device
	State  map[string]interface{}
}

type pollResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func Subscribe(client *http.Client, config *conf.Config) (<-chan string, error) {
	output := make(chan string, 1)
	pollID, err := getSinglePollID(client, config)
	if err != nil {
		return nil, err
	}
	output <- pollID
	ticker := time.NewTicker(time.Minute * time.Duration(config.PollIDUpdateInterval))
	go func() {
		defer close(output)
		for range ticker.C {
			pollID2, e := getSinglePollID(client, config)
			if err != nil {
				log.Err(e).Msg("Error getting polling id")
			}
			output <- pollID2
		}
	}()
	return output, nil
}

func getSinglePollID(client *http.Client, config *conf.Config) (string, error) {
	requestBody := []pollRequest{
		{
			Jsonrpc: "2.0",
			Method:  "RE/subscribe",
			Params:  []interface{}{"com/bosch/sh/remote/*", nil},
		},
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}
	shcPollURL := fmt.Sprintf("%s/remote/json-rpc", config.BoschConfig.BaseURL)
	log.Info().
		Str("url", shcPollURL).
		Bytes("body", requestBodyBytes).
		Msg("Creating poll subscription")

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		shcPollURL,
		bytes.NewReader(requestBodyBytes),
	)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		e := Body.Close()
		if e != nil {
			log.Err(e).Msg("Error closing response body")
		}
	}(resp.Body)
	buf := &bytes.Buffer{}
	if _, e := buf.ReadFrom(resp.Body); e != nil {
		return "", e
	}
	body := buf.Bytes()
	log.Trace().
		Bytes("responseBody", body).
		Int("status", resp.StatusCode).
		Msg("Response of poll subscription")

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("response status of get geive call is not %d, but %d", http.StatusOK, resp.StatusCode)
	}

	var response []pollSubscribeResponse
	if e := json.Unmarshal(body, &response); e != nil {
		return "", e
	}
	pollID := response[0].Result
	log.Info().Str("pollID", pollID).Msg("Created poll subscription")
	return pollID, nil
}

var currentDevices []*devices.Device
var pollID string
var lock = sync.Mutex{}

func Start(
	client *http.Client,
	pollIDchan <-chan string,
	deviceChan <-chan []*devices.Device,
	config *conf.Config,
) <-chan *Event {
	lock.Lock()
	currentDevices = <-deviceChan
	pollID = <-pollIDchan
	lock.Unlock()
	go func() {
		for d := range deviceChan {
			lock.Lock()
			currentDevices = d
			lock.Unlock()
		}
	}()
	go func() {
		for id := range pollIDchan {
			lock.Lock()
			pollID = id
			lock.Unlock()
		}
	}()
	output := make(chan *Event)
	go func() {
		defer close(output)
		var err error
		for err == nil {
			var events []*Event
			events, err = Get(client, pollID, config)
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

func Get(client *http.Client, pollID string, config *conf.Config) ([]*Event, error) {
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
	shcPollURL := fmt.Sprintf("%s/remote/json-rpc", config.BoschConfig.BaseURL)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		shcPollURL,
		bytes.NewReader(requestBodyBytes),
	)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		e := Body.Close()
		if e != nil {
			log.Err(e).Msg("Error closing response body")
		}
	}(resp.Body)
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
		device := getDevice(event.DeviceID)
		if device == nil {
			device = devices.DefaultDevice
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

func getDevice(id string) *devices.Device {
	lock.Lock()
	defer lock.Unlock()
	for _, d := range currentDevices {
		if d.ID == id {
			return d
		}
	}
	return nil
}
