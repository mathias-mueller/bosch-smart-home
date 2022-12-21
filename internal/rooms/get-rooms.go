package rooms

import (
	"bosch-data-exporter/internal/conf"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type roomResponse struct {
	Type          string                 `json:"@type"`
	Id            string                 `json:"id"`
	IconId        string                 `json:"iconId"`
	Name          string                 `json:"name"`
	ExtProperties map[string]interface{} `json:"extProperties"`
}

type Room struct {
	Id   string
	Name string
}

var DefaultRoom = &Room{
	Id:   "",
	Name: "default",
}

func GetRooms(client *http.Client, config *conf.Config) <-chan []*Room {
	output := make(chan []*Room)
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
	req, err := http.NewRequest(
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
			Str("id", jsonBody[i].Id).
			Str("name", jsonBody[i].Name).
			Msg("Got room")
		rooms = append(rooms,
			&Room{
				Id:   jsonBody[i].Id,
				Name: jsonBody[i].Name,
			},
		)
	}

	return rooms, nil
}
