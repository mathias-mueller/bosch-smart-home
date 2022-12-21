package client

import (
	"bosch-data-exporter/internal/conf"
	"crypto/tls"
	"net/http"

	"github.com/rs/zerolog/log"
)

func Init(config *conf.Config) *http.Client {
	cert, err := tls.LoadX509KeyPair(config.ClientCertPath, config.ClientKeyPath)
	if err != nil {
		log.Fatal().Err(err).
			Str("clientKeyFile", config.ClientKeyPath).
			Str("clientCertFile", config.ClientCertPath).
			Msg("Error creating x509 keypair from client cert file %s and client key file ")
	}

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			//nolint:gosec // https but only locally signed certificates
			InsecureSkipVerify: true,
		},
	}
	log.Trace().
		Str("clientKeyFile", config.ClientKeyPath).
		Str("clientCertFile", config.ClientCertPath).
		Msg("Created http client with certificates")
	return &http.Client{
		Transport: t,
	}
}
