package export

import (
	"bosch-data-exporter/internal/conf"
	"bosch-data-exporter/internal/events"
	"context"
	"fmt"
	"net/http"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	influxHttp "github.com/influxdata/influxdb-client-go/v2/api/http"
	"github.com/rs/zerolog/log"
)

type InfluxExporter struct {
	client   influxdb2.Client
	writeAPI api.WriteAPI
	config   *conf.Config
}

func NewInfluxExporter(config *conf.Config) (*InfluxExporter, error) {
	client := influxdb2.NewClientWithOptions(
		config.InfluxConfig.ServerURL,
		config.InfluxConfig.AuthToken,
		influxdb2.DefaultOptions(),
	)
	writeAPI := client.WriteAPI(config.InfluxConfig.Org, config.InfluxConfig.Bucket)
	writeAPI.SetWriteFailedCallback(func(batch string, err influxHttp.Error, retryAttempts uint) bool {
		log.Err(err.Err).
			Str("batch", batch).
			Uint("retryAttempts", retryAttempts).
			Str("message", err.Message).
			Int("status", err.StatusCode).
			Uint("retryAfter", err.RetryAfter).
			Msg("Error writing data to influx")
		return err.StatusCode != http.StatusUnauthorized
	})
	ping, err := client.Ping(context.Background())
	if err != nil {
		return nil, err
	}
	if !ping {
		return nil, fmt.Errorf("ping did not succeed")
	}

	return &InfluxExporter{
		client:   client,
		writeAPI: writeAPI,
		config:   config,
	}, nil
}

func (e *InfluxExporter) Start(events <-chan *events.Event) {
	for event := range events {
		log.Debug().
			Str("type", event.Type).
			Interface("state", event.State).
			Str("device", event.Device.Name).
			Str("room", event.Device.Room.Name).
			Str("id", event.ID).
			Msg("Got Event")
		e.exportRaw(event)
		e.parseAndExport(event)
	}
}

func (e *InfluxExporter) exportRaw(event *events.Event) {
	p := influxdb2.NewPoint(fmt.Sprintf("raw_%s", event.ID),
		map[string]string{
			"device": event.Device.Name,
			"room":   event.Device.Room.Name,
		},
		event.State,
		time.Now(),
	)
	e.writeAPI.WritePoint(p)
}
