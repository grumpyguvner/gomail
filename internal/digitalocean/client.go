package digitalocean

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a DigitalOcean API client
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new DigitalOcean API client
func NewClient(token string) *Client {
	return &Client{
		token:   token,
		baseURL: "https://api.digitalocean.com/v2",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request to the DigitalOcean API
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errorResp struct {
			ID      string `json:"id"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Message != "" {
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, errorResp.Message)
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}