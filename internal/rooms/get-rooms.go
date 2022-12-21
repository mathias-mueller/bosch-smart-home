package rooms

import (
	"bosch-data-exporter/internal/conf"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type roomResponse struct {
	Type          string                 `json:"@type"`
	ID            string                 `json:"id"`
	IconID        string                 `json:"iconId"`
	Name          string                 `json:"name"`
	ExtProperties map[string]interface{} `json:"extProperties"`
}

type Room struct {
	ID   string
	Name string
}

var DefaultRoom = &Room{
	ID:   "",
	Name: "default",
}

func GetRooms(client *http.Client, config *conf.Config) <-chan []*Room {
	output := make(chan []*Room)
	go pipeSingle(client, output, config)
	ticker := time.NewTicker(time.Minute * time.Duration(config.DeviceUpdateInterval))
	go func() {
		defer close(output)
		for range ticker.C {
			go pipeSingle(client, output, nil)
		}
	}()
	return output
}

func pipeSingle(client *http.Client, output chan []*Room, config *conf.Config) {
	if rooms, err := getSingle(client, config); err != nil {
		log.Err(err).Msg("Error getting rooms")
	} else {
		rooms = append(rooms, DefaultRoom)
		output <- rooms
	}
}

func getSingle(client *http.Client, config *conf.Config) ([]*Room, error) {
	log.Debug().Msg("Getting rooms...")
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/smarthome/rooms", config.BaseURL),
		nil,
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response status of get geive call is not %d, but %d", http.StatusOK, resp.StatusCode)
	}
	var jsonBody []roomResponse

	if e := json.Unmarshal(body, &jsonBody); e != nil {
		return nil, e
	}
	rooms := make([]*Room, 0)
	for i := range jsonBody {
		log.Info().
			Str("id", jsonBody[i].ID).
			Str("name", jsonBody[i].Name).
			Msg("Got room")
		rooms = append(rooms,
			&Room{
				ID:   jsonBody[i].ID,
				Name: jsonBody[i].Name,
			},
		)
	}

	return rooms, nil
}
