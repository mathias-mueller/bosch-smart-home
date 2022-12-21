package client

import (
	"bosch-data-exporter/internal/conf"
	"crypto/tls"
	"net/http"

	"github.com/rs/zerolog/log"
)

const clientCertFile = "client-cert.pem"
const clientKeyFile = "client-key.pem"

func Init(config *conf.Config) *http.Client {

	cert, err := tls.LoadX509KeyPair(config.ClientCertPath, config.ClientKeyPath)
	if err != nil {
		log.Fatal().Err(err).
			Str("clientKeyFile", config.ClientKeyPath).
			Str("clientCertFile", config.ClientCertPath).
			Msg("Error creating x509 keypair from client cert file %s and client key file ")
	}

	//caCert, err := ioutil.ReadFile(caCertFile)
	//if err != nil {
	//	log.Fatalf("Error opening cert file %s, Error: %s", caCertFile, err)
	//}
	//caCertPool := x509.NewCertPool()
	//caCertPool.AppendCertsFromPEM(caCert)

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
			//RootCAs:      caCertPool,
		},
	}
	log.Trace().
		Str("clientKeyFile", clientCertFile).
		Str("clientCertFile", clientCertFile).
		Msg("Created http client with certificates")
	return &http.Client{
		Transport: t,
	}
}
