package main

import (
	"bosch-data-exporter/internal/cache"
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
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	config, err := conf.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}
	logLevel, err := zerolog.ParseLevel(config.LogLevel)
	if err != nil {
		log.Fatal().Err(err).
			Str("level", config.LogLevel).
			Msg("Failed to parse log level")
	}
	zerolog.SetGlobalLevel(logLevel)

	httpClient := client.Init(config)

	err = register.Register(httpClient, config)
	if err != nil {
		log.Fatal().Err(err).Msg("Error registering client")
	}

	roomPolling := rooms.NewRoomPolling(httpClient, config)
	cachedRooms := cache.New(roomPolling.Get, time.Duration(config.DeviceUpdateInterval)*time.Minute)

	devicePolling := devices.NewDevicePolling(httpClient, cachedRooms, config)
	cachedDevices := cache.New(devicePolling.Get, time.Duration(config.DeviceUpdateInterval)*time.Minute)

	pollID := polling.New(httpClient, config)
	cachedPollID := cache.New(pollID.Get, time.Minute*time.Duration(config.PollIDUpdateInterval))

	exporter, err := export.NewInfluxExporter(config)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not connect influx client")
	}

	eventPolling := events.NewSmartHomeEventPolling(httpClient, cachedDevices, cachedPollID, exporter, config)

	go eventPolling.Start()

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
