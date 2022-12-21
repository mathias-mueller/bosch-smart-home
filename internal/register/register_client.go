package register

import (
	"bosch-data-exporter/internal/conf"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
)

const clientCertFile = "client-cert.pem"

type registerRequest struct {
	Type        string `json:"@type"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	PrimaryRole string `json:"primaryRole"`
	Certificate string `json:"certificate"`
}

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

	log.Info().Msg("Register client...")
	cert, err := os.ReadFile(clientCertFile)
	if err != nil {
		return err
	}

	requestData := registerRequest{
		Type:        "client",
		ID:          config.BoschConfig.ClientID,
		Name:        config.BoschConfig.ClientName,
		PrimaryRole: "ROLE_RESTRICTED_CLIENT",
		Certificate: string(cert),
	}

	data, err := json.Marshal(requestData)
	if err != nil {
		return err
	}
	shcURL := fmt.Sprintf("%s/smarthome/clients", config.BaseURL)
	log.Trace().
		Bytes("body", data).
		Str("url", shcURL).
		Msg("Registering client")
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		shcURL,
		bytes.NewReader(data),
	)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Err(err).Msg("Error closing response body")
		}
	}(resp.Body)

	buf := &bytes.Buffer{}
	if _, e := buf.ReadFrom(resp.Body); e != nil {
		return e
	}
	body := buf.Bytes()
	log.Info().
		Bytes("response", body).
		Int("status", resp.StatusCode).
		Msg("Register response")

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response status of get geive call is not %d, but %d", http.StatusOK, resp.StatusCode)
	}

	return nil
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
	response, err := client.Get(fmt.Sprintf("%s/smarthome/clients", config.BaseURL))
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Err(err).Msg("Error closing response body")
		}
	}(response.Body)
	buf := &bytes.Buffer{}
	if _, e := buf.ReadFrom(response.Body); e != nil {
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
