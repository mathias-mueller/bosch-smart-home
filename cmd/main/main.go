package main

import (
	"bosch-data-exporter/internal/client"
	"bosch-data-exporter/internal/conf"
	"bosch-data-exporter/internal/devices"
	"bosch-data-exporter/internal/events"
	"bosch-data-exporter/internal/export"
	"bosch-data-exporter/internal/polling"
	"bosch-data-exporter/internal/register"
	"bosch-data-exporter/internal/rooms"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}

	httpClient := client.Init(config)

	err = register.Register(httpClient, config)
	if err != nil {
		log.Fatal().Err(err).Msg("Error registering client")
	}

	roomPolling := rooms.NewRoomPolling(httpClient, config)

	roomChan := roomPolling.GetRooms()

	devicePolling := devices.NewDevicePolling(httpClient, config)

	deviceChan := devicePolling.GetDevices(roomChan)

	pollID, err := polling.Subscribe(httpClient, config)
	if err != nil {
		log.Fatal().Err(err).Msg("error getting poll id")
	}

	eventPolling := events.NewSmartHomeEventPolling(httpClient, config)

	eventChan := eventPolling.Start(pollID, deviceChan)

	go export.Start(eventChan)

	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.Port),
		ReadHeaderTimeout: time.Second,
		Handler:           handler,
	}

	err = server.ListenAndServe()
	if err != nil {
		log.Err(err).Msg("Server failed")
		os.Exit(1)
	}
}
