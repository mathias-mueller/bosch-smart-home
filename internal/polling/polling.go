package polling

import (
	"bosch-data-exporter/internal/conf"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type pollRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type pollSubscribeResponse struct {
	Result  string `json:"result"`
	Jsonrpc string `json:"jsonrpc"`
}

type PollIDGenerator struct {
	client  httpClient
	baseURL string
}

func New(client httpClient, config *conf.Config) *PollIDGenerator {
	return &PollIDGenerator{
		client:  client,
		baseURL: config.BoschConfig.BaseURL,
	}
}

func (p *PollIDGenerator) Get() (string, error) {
	requestBody := []pollRequest{
		{
			Jsonrpc: "2.0",
			Method:  "RE/subscribe",
			Params:  []interface{}{"com/bosch/sh/remote/*", nil},
		},
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}
	shcPollURL := fmt.Sprintf("%s/remote/json-rpc", p.baseURL)
	log.Info().
		Str("url", shcPollURL).
		Bytes("body", requestBodyBytes).
		Msg("Creating poll subscription")

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		shcPollURL,
		bytes.NewReader(requestBodyBytes),
	)
	if err != nil {
		return "", err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		e := resp.Body.Close()
		if e != nil {
			log.Err(e).Msg("Error closing response body")
		}
	}()
	buf := &bytes.Buffer{}
	if _, e := buf.ReadFrom(resp.Body); e != nil {
		return "", e
	}
	body := buf.Bytes()
	log.Trace().
		Bytes("responseBody", body).
		Int("status", resp.StatusCode).
		Msg("Response of poll subscription")

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("response status of get geive call is not %d, but %d", http.StatusOK, resp.StatusCode)
	}

	var response []pollSubscribeResponse
	if e := json.Unmarshal(body, &response); e != nil {
		return "", e
	}
	pollID := response[0].Result
	log.Info().Str("pollID", pollID).Msg("Created poll subscription")
	return pollID, nil
}
