package rooms

import (
	"bosch-data-exporter/internal/conf"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

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
	client          httpClient
	updateInterval  int
	baseURL         string
	reqDurationHist prometheus.Histogram
	lock            *sync.Mutex
}

func NewRoomPolling(client httpClient, config *conf.Config) *RoomPolling {
	return &RoomPolling{
		client:         client,
		updateInterval: config.DeviceUpdateInterval,
		baseURL:        config.BoschConfig.BaseURL,
		reqDurationHist: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "bosch_room_poll_duration",
			Help: "Duration of the GET Room call",
		}),
		lock: &sync.Mutex{},
	}
}

func (r *RoomPolling) Get() ([]*Room, error) {
	timer := prometheus.NewTimer(r.reqDurationHist)
	defer timer.ObserveDuration()
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/smarthome/rooms", r.baseURL),
		nil,
	)
	if err != nil {
		return nil, err
	}
	log.Debug().
		Str("url", req.URL.Path).
		Msg("Getting rooms...")
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
		log.Debug().
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
	log.Info().Int("number", len(rooms)).Msg("Got rooms")
	return rooms, nil
}
