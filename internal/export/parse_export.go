package export

import (
	"bosch-data-exporter/internal/events"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
)

type ShutterContactState struct {
	Type  string `json:"@type"`
	Value string `json:"value"`
}

type TemperatureLevelState struct {
	Type        string  `json:"@type"`
	Temperature float64 `json:"temperature"`
}

type HumidityLevelState struct {
	Type     string  `json:"@type"`
	Humidity float64 `json:"humidity"`
}

type ValveTappetState struct {
	Type     string `json:"@type"`
	Position int    `json:"position"`
	Value    string `json:"value"`
}

type ClimateControlState struct {
	Type            string `json:"@type"`
	BoostMode       bool   `json:"boostMode"`
	Low             bool   `json:"low"`
	OperationMode   string `json:"operationMode"`
	RoomControlMode string `json:"roomControlMode"`
	Schedule        struct {
		Profiles []struct {
			Day          string `json:"day"`
			SwitchPoints []struct {
				StartTimeMinutes int `json:"startTimeMinutes"`
				Value            struct {
					Type             string `json:"@type"`
					TemperatureLevel string `json:"temperatureLevel"`
				} `json:"value"`
			} `json:"switchPoints"`
		} `json:"profiles"`
	} `json:"schedule"`
	SetpointTemperature                float64 `json:"setpointTemperature"`
	SetpointTemperatureForLevelComfort float64 `json:"setpointTemperatureForLevelComfort"`
	SetpointTemperatureForLevelEco     int     `json:"setpointTemperatureForLevelEco"`
	SummerMode                         bool    `json:"summerMode"`
	SupportsBoostMode                  bool    `json:"supportsBoostMode"`
	VentilationMode                    bool    `json:"ventilationMode"`
}

func (e *InfluxExporter) parseAndExport(event *events.Event) {
	switch event.ID {
	case "RoomClimateControl":
		e.exportRoomClimateControl(event)
	case "ShutterContact":
		e.exportShutterContact(event)
	case "TemperatureLevel":
		e.ExportTemperatureLevelState(event)
	case "HumidityLevel":
		e.ExportHumidityLevelState(event)
	case "ValveTappet":
		e.ExportValveTappetState(event)
	}
}

func (e *InfluxExporter) exportRoomClimateControl(event *events.Event) {
	var parsedState ClimateControlState

	if err := parseState(&parsedState, event.State); err != nil {
		log.Err(err).Msg("Error parsing state")
		return
	}

	fields := map[string]interface{}{
		"setpointTemperature":                parsedState.SetpointTemperature,
		"setpointTemperatureForLevelComfort": parsedState.SetpointTemperatureForLevelComfort,
		"setpointTemperatureForLevelEco":     parsedState.SetpointTemperatureForLevelEco,
	}
	if parsedState.SummerMode {
		fields["summerMode"] = 1
	} else {
		fields["summerMode"] = 0
	}
	if parsedState.VentilationMode {
		fields["ventilationMode"] = 1
	} else {
		fields["ventilationMode"] = 0
	}
	if parsedState.BoostMode {
		fields["boostMode"] = 1
	} else {
		fields["boostMode"] = 0
	}
	if parsedState.Low {
		fields["low"] = 1
	} else {
		fields["low"] = 0
	}
	p := influxdb2.NewPoint("room_climate",
		map[string]string{
			"device": event.Device.Name,
			"room":   event.Device.Room.Name,
		},
		fields,
		time.Now(),
	)
	e.writeAPI.WritePoint(p)
}

func (e *InfluxExporter) exportShutterContact(event *events.Event) {
	var parsedState ShutterContactState

	if err := parseState(&parsedState, event.State); err != nil {
		log.Err(err).Msg("Error parsing state")
		return
	}

	fields := map[string]interface{}{}
	if parsedState.Value == "OPEN" {
		fields["open"] = 1
	} else {
		fields["open"] = 0
	}
	p := influxdb2.NewPoint("shutter_contact",
		map[string]string{
			"device": event.Device.Name,
			"room":   event.Device.Room.Name,
		},
		fields,
		time.Now(),
	)
	e.writeAPI.WritePoint(p)
}

func (e *InfluxExporter) ExportTemperatureLevelState(event *events.Event) {
	var parsedState TemperatureLevelState

	if err := parseState(&parsedState, event.State); err != nil {
		log.Err(err).Msg("Error parsing state")
		return
	}

	fields := map[string]interface{}{
		"temperature": parsedState.Temperature,
	}
	p := influxdb2.NewPoint("temperature",
		map[string]string{
			"device": event.Device.Name,
			"room":   event.Device.Room.Name,
		},
		fields,
		time.Now(),
	)
	e.writeAPI.WritePoint(p)
}

func (e *InfluxExporter) ExportValveTappetState(event *events.Event) {
	var parsedState ValveTappetState

	if err := parseState(&parsedState, event.State); err != nil {
		log.Err(err).Msg("Error parsing state")
		return
	}

	fields := map[string]interface{}{
		"position": parsedState.Position,
	}
	p := influxdb2.NewPoint("valve_tappet",
		map[string]string{
			"device": event.Device.Name,
			"room":   event.Device.Room.Name,
		},
		fields,
		time.Now(),
	)
	e.writeAPI.WritePoint(p)
}

func (e *InfluxExporter) ExportHumidityLevelState(event *events.Event) {
	var parsedState HumidityLevelState

	if err := parseState(&parsedState, event.State); err != nil {
		log.Err(err).Msg("Error parsing state")
		return
	}

	fields := map[string]interface{}{
		"humidity": parsedState.Humidity,
	}
	p := influxdb2.NewPoint("humidity",
		map[string]string{
			"device": event.Device.Name,
			"room":   event.Device.Room.Name,
		},
		fields,
		time.Now(),
	)
	e.writeAPI.WritePoint(p)
}

func parseState(x interface{}, input map[string]interface{}) error {
	config := &mapstructure.DecoderConfig{
		TagName: "json",
	}
	config.Result = x
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}
