package polling

import (
	"bosch-data-exporter/internal/conf"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type pollRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type pollSubscribeResponse struct {
	Result  string `json:"result"`
	Jsonrpc string `json:"jsonrpc"`
}

func Subscribe(client *http.Client, config *conf.Config) (<-chan string, error) {
	output := make(chan string, 1)
	pollID, err := getSinglePollID(client, config)
	if err != nil {
		return nil, err
	}
	output <- pollID
	ticker := time.NewTicker(time.Minute * time.Duration(config.PollIDUpdateInterval))
	go func() {
		defer close(output)
		for range ticker.C {
			pollID2, e := getSinglePollID(client, config)
			if err != nil {
				log.Err(e).Msg("Error getting polling id")
			}
			output <- pollID2
		}
	}()
	return output, nil
}

func getSinglePollID(client *http.Client, config *conf.Config) (string, error) {
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
	shcPollURL := fmt.Sprintf("%s/remote/json-rpc", config.BoschConfig.BaseURL)
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

	resp, err := client.Do(req)
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
