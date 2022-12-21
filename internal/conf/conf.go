package conf

type Config struct {
	DeviceUpdateInterval int
	BaseURL              string
	ClientCertPath       string
	ClientKeyPath        string
	InfluxConfig         *InfluxConfig
	BoschConfig          *BoschConfig
}

type BoschConfig struct {
	ClientID   string
	ClientName string
}

type InfluxConfig struct {
	ServerURL string
	AuthToken string
	Org       string
	Bucket    string
}

func LoadConfig() (*Config, error) {
	return &Config{
		DeviceUpdateInterval: 10,
		BaseURL:              "https://shc1084ad:8444",
		ClientCertPath:       "client-cert.pem",
		ClientKeyPath:        "client-key.pem",
		InfluxConfig: &InfluxConfig{
			ServerURL: "http://localhost:8086",
			AuthToken: "adminToken",
			Org:       "home",
			Bucket:    "smarthome",
		},
		BoschConfig: &BoschConfig{
			ClientID:   "oss_go_exporter",
			ClientName: "OSS Go Data Exporter",
		},
	}, nil
}
