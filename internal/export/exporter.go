package export

import (
	"bosch-data-exporter/internal/polling"
	"net/http"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxHttp "github.com/influxdata/influxdb-client-go/v2/api/http"
	"github.com/rs/zerolog/log"
)

func Start(events <-chan *polling.Event) {
	client := influxdb2.NewClientWithOptions(
		"http://localhost:8086",
		"adminToken",
		influxdb2.DefaultOptions(),
	)
	writeAPI := client.WriteAPI("home", "smarthome")
	writeAPI.SetWriteFailedCallback(func(batch string, error influxHttp.Error, retryAttempts uint) bool {
		log.Err(error.Err).
			Str("batch", batch).
			Uint("retryAttempts", retryAttempts).
			Str("message", error.Message).
			Int("status", error.StatusCode).
			Uint("retryAfter", error.RetryAfter).
			Msg("Error writing data to influx")
		if error.StatusCode == http.StatusUnauthorized {
			return false
		}
		return true
	})
	for event := range events {
		log.Info().
			Str("type", event.Type).
			Interface("state", event.State).
			Str("device", event.Device.Name).
			Str("room", event.Device.Room.Name).
			Str("id", event.Id).
			Msg("Got Event")
		p := influxdb2.NewPoint(event.Id,
			map[string]string{
				"device": event.Device.Name,
				"room":   event.Device.Room.Name,
			},
			event.State,
			time.Now(),
		)
		writeAPI.WritePoint(p)
	}

}
