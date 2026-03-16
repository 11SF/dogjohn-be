package access

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type HomeAssistantClient interface {
	TriggerFeeder(ctx context.Context) error
}

type homeAssistantClient struct {
	baseURL  string
	token    string
	entityID string
	client   *http.Client
}

func NewHomeAssistantClient(baseURL, token, entityID string, client *http.Client) HomeAssistantClient {
	return &homeAssistantClient{
		baseURL:  baseURL,
		token:    token,
		entityID: entityID,
		client:   client,
	}
}

func (c *homeAssistantClient) TriggerFeeder(ctx context.Context) error {
	body, err := json.Marshal(map[string]string{"entity_id": c.entityID})
	if err != nil {
		return fmt.Errorf("ha: marshal body: %w", err)
	}

	url := fmt.Sprintf("%s/api/services/script/turn_on", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("ha: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("ha: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("ha: unexpected status %d", resp.StatusCode)
	}
	return nil
}
