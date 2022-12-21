package conf

import (
	"encoding/json"
	"os"
)

type Config struct {
	DeviceUpdateInterval int
	PollIDUpdateInterval int
	ClientCertPath       string
	ClientKeyPath        string
	InfluxConfig         *InfluxConfig
	BoschConfig          *BoschConfig
}

type BoschConfig struct {
	ClientID   string
	ClientName string
	BaseURL    string
}

type InfluxConfig struct {
	ServerURL string
	AuthToken string
	Org       string
	Bucket    string
}

func LoadConfig() (*Config, error) {
	content, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}
	var result *Config
	err = json.Unmarshal(content, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
