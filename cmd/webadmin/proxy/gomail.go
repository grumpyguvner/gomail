package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
)

type GoMailProxy struct {
	baseURL     string
	bearerToken string
	logger      *logging.Logger
	client      *http.Client
}

func NewGoMailProxy(baseURL, bearerToken string, logger *logging.Logger) *GoMailProxy {
	return &GoMailProxy{
		baseURL:     baseURL,
		bearerToken: bearerToken,
		logger:      logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *GoMailProxy) ListEmails(query url.Values) (interface{}, error) {
	url := fmt.Sprintf("%s/api/emails", p.baseURL)
	if len(query) > 0 {
		url += "?" + query.Encode()
	}

	resp, err := p.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func (p *GoMailProxy) GetEmail(emailID string) (interface{}, error) {
	url := fmt.Sprintf("%s/api/emails/%s", p.baseURL, emailID)

	resp, err := p.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("email not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func (p *GoMailProxy) DeleteEmail(emailID string) error {
	url := fmt.Sprintf("%s/api/emails/%s", p.baseURL, emailID)

	resp, err := p.makeRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("email not found")
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (p *GoMailProxy) GetEmailRaw(emailID string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/emails/%s/raw", p.baseURL, emailID)

	resp, err := p.makeRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("email not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

func (p *GoMailProxy) makeRequest(method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+p.bearerToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	p.logger.Debug("Making request to GoMail API",
		"method", method,
		"url", url,
	)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}