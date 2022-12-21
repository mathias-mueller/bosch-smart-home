package main

import (
	"bosch-data-exporter/internal/client"
	"bosch-data-exporter/internal/conf"
	"bosch-data-exporter/internal/devices"
	"bosch-data-exporter/internal/export"
	"bosch-data-exporter/internal/polling"
	"bosch-data-exporter/internal/register"
	"bosch-data-exporter/internal/rooms"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(
		zerolog.ConsoleWriter{
			Out: os.Stdout,
		},
	)
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	config, err := conf.LoadConfig()

	httpClient := client.Init(config)

	err = register.Register(httpClient, config)
	if err != nil {
		log.Fatal().Err(err).Msg("Error registering client")
	}

	roomChan := rooms.GetRooms(httpClient, config)

	deviceChan := devices.GetDevices(httpClient, roomChan, config)

	pollID, err := polling.Subscribe(httpClient, config)
	if err != nil {
		log.Fatal().Err(err).Msg("error getting poll id")
	}

	events := polling.Start(httpClient, pollID, deviceChan, config)

	export.Start(events)

}