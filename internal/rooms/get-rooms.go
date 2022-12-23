package rooms

import (
	"bosch-data-exporter/internal/conf"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

func DefaultRoom() *Room {
	return &Room{
		ID:   "",
		Name: "default",
	}
}

type RoomPolling struct {
	client          *http.Client
	config          *conf.Config
	reqDurationHist prometheus.Histogram
}

func NewRoomPolling(client *http.Client, config *conf.Config) *RoomPolling {
	return &RoomPolling{
		client: client,
		config: config,
		reqDurationHist: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "bosch_room_poll_duration",
			Help: "Duration of the GET Room call",
		}),
	}
}

func (r *RoomPolling) GetRooms() <-chan []*Room {
	output := make(chan []*Room)
	go r.pipeSingle(output)
	ticker := time.NewTicker(time.Minute * time.Duration(r.config.DeviceUpdateInterval))
	go func() {
		defer close(output)
		for range ticker.C {
			go r.pipeSingle(output)
		}
	}()
	return output
}

func (r *RoomPolling) pipeSingle(output chan []*Room) {
	timer := prometheus.NewTimer(r.reqDurationHist)
	defer timer.ObserveDuration()
	rooms, err := r.getSingle()
	if err != nil {
		log.Err(err).Msg("Error getting rooms")
		return
	}
	rooms = append(rooms, DefaultRoom())
	output <- rooms
}

func (r *RoomPolling) getSingle() ([]*Room, error) {
	log.Debug().Msg("Getting rooms...")
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/smarthome/rooms", r.config.BoschConfig.BaseURL),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
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
