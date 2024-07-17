package alertmanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("alertmanager")

type Api struct {
	httpClient http.Client
	host       string
	token      string
}

type Silence struct {
	ID        string    `json:"id"`
	Comment   string    `json:"comment"`
	CreatedBy string    `json:"createdBy"`
	EndsAt    string    `json:"endsAt"`
	Matchers  []Matcher `json:"matchers"`
	StartsAt  string    `json:"startsAt"`
	Status    Status    `json:"status"`
}

type Status struct {
	State string `json:"state"`
}

type Matcher struct {
	IsEqual bool   `json:"isEqual"`
	IsRegex bool   `json:"isRegex"`
	Name    string `json:"name"`
	Value   string `json:"value"`
}

func NewAPI(httpClient http.Client, host string, token string) *Api {
	return &Api{
		httpClient: httpClient,
		host:       host,
		token:      token,
	}
}

func (api *Api) ListSilences() ([]Silence, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v2/silences", api.host), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+api.token)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list silences: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.V(1).Info("list silences http request", "req", req, "resp", resp)
		return nil, fmt.Errorf("failed to list silences: %s", resp.Status)
	}

	var amSilences []Silence
	err = json.NewDecoder(resp.Body).Decode(&amSilences)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return amSilences, nil
}

func (api *Api) CreateSilence(s Silence) error {
	body, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal silence: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v2/silences", api.host), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+api.token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create silence: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create silence: %s", resp.Status)
	}

	return nil
}

func (api *Api) DeleteSilence(id string) error {
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/v2/silence/%s", api.host, id), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+api.token)

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete silence: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete silence: %s", resp.Status)
	}

	return nil
}
