package register

import (
	"bosch-data-exporter/internal/conf"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

func Register(client *http.Client, config *conf.Config) error {
	clients, err := getRegisteredClients(client, config)
	if err != nil {
		log.Err(err).Msg("Error getting registered clients")
		return err
	}
	for _, boschClient := range clients {
		log.Debug().
			Str("id", boschClient.ID).
			Str("name", boschClient.Name).
			Msg("Checking registered client")
		if boschClient.ID == config.BoschConfig.ClientID {
			log.Info().
				Str("client_id", boschClient.ID).
				Msg("Client already registered. Skipping creation")
			return nil
		}
	}
	return fmt.Errorf("client is not registered. Please Setup the CLient via the postman collection: https://github.com/BoschSmartHome/bosch-shc-api-docs/tree/master/postman")
}

type BoschClientResponse struct {
	Type         string        `json:"@type"`
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	PrimaryRole  string        `json:"primaryRole"`
	Roles        []string      `json:"roles"`
	DynamicRoles []interface{} `json:"dynamicRoles"`
	OsVersion    string        `json:"osVersion"`
	AppVersion   string        `json:"appVersion"`
	ClientType   string        `json:"clientType"`
	CreatedDate  string        `json:"createdDate"`
}

func getRegisteredClients(client *http.Client, config *conf.Config) ([]*BoschClientResponse, error) {
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		fmt.Sprintf("%s/smarthome/clients", config.BoschConfig.BaseURL),
		nil,
	)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		e := resp.Body.Close()
		if e != nil {
			log.Err(e).Msg("Error closing resp body")
		}
	}()
	buf := &bytes.Buffer{}
	if _, e := buf.ReadFrom(resp.Body); e != nil {
		return nil, e
	}
	body := buf.Bytes()

	var clients []*BoschClientResponse

	err = json.Unmarshal(body, &clients)
	if err != nil {
		return nil, err
	}
	return clients, nil
}
