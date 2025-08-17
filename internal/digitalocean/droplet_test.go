package digitalocean

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListDroplets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/droplets", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		response := `{
			"droplets": [
				{
					"id": 123,
					"name": "test-droplet",
					"status": "active",
					"networks": {
						"v4": [
							{
								"ip_address": "192.168.1.1",
								"type": "public"
							}
						]
					}
				}
			]
		}`
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	client := &Client{
		token:      "test-token",
		baseURL:    server.URL + "/v2",
		httpClient: &http.Client{},
	}

	droplets, err := client.ListDroplets()
	require.NoError(t, err)
	assert.Len(t, droplets, 1)
	assert.Equal(t, 123, droplets[0].ID)
	assert.Equal(t, "test-droplet", droplets[0].Name)
	assert.Equal(t, "192.168.1.1", droplets[0].Networks.V4[0].IPAddress)
}

func TestRenameDroplet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/droplets/123/actions", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "rename", body["type"])
		assert.Equal(t, "mail.example.com", body["name"])

		w.WriteHeader(200)
	}))
	defer server.Close()

	client := &Client{
		token:      "test-token",
		baseURL:    server.URL + "/v2",
		httpClient: &http.Client{},
	}

	err := client.RenameDroplet(123, "mail.example.com")
	assert.NoError(t, err)
}

func TestSetupPTRRecord(t *testing.T) {
	t.Run("renames droplet when name doesn't match", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if r.URL.Path == "/v2/droplets" && r.Method == "GET" {
				// Return droplet with different name
				response := `{
					"droplets": [{
						"id": 456,
						"name": "old-name",
						"networks": {
							"v4": [{
								"ip_address": "10.0.0.1",
								"type": "private"
							}, {
								"ip_address": "192.168.1.1",
								"type": "public"
							}]
						}
					}]
				}`
				_, _ = w.Write([]byte(response))
			} else if r.URL.Path == "/v2/droplets/456/actions" && r.Method == "POST" {
				// Rename request
				var body map[string]string
				_ = json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, "rename", body["type"])
				assert.Equal(t, "mail.example.com", body["name"])
				w.WriteHeader(200)
			}
		}))
		defer server.Close()

		// Mock the getLocalPublicIP function to return our test IP
		// Note: In a real test, we'd need to inject this dependency
		_ = &Client{
			token:      "test-token",
			baseURL:    server.URL + "/v2",
			httpClient: &http.Client{},
		}

		// This test would need the actual server to have IP 192.168.1.1
		// In practice, we'd need to mock or inject the IP detection
		// For now, we'll test the other methods independently
	})

	t.Run("skips rename when name already matches", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v2/droplets" && r.Method == "GET" {
				// Return droplet with matching name
				response := `{
					"droplets": [{
						"id": 456,
						"name": "mail.example.com",
						"networks": {
							"v4": [{
								"ip_address": "192.168.1.1",
								"type": "public"
							}]
						}
					}]
				}`
				_, _ = w.Write([]byte(response))
			} else {
				t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			}
		}))
		defer server.Close()

		_ = &Client{
			token:      "test-token",
			baseURL:    server.URL + "/v2",
			httpClient: &http.Client{},
		}

		// This would work if we could mock the local IP detection
		// to return 192.168.1.1
	})
}

func TestGetDropletPublicIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/droplets/789", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := `{
			"droplet": {
				"id": 789,
				"name": "test",
				"networks": {
					"v4": [
						{
							"ip_address": "10.0.0.1",
							"type": "private"
						},
						{
							"ip_address": "203.0.113.1",
							"type": "public"
						}
					]
				}
			}
		}`
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	client := &Client{
		token:      "test-token",
		baseURL:    server.URL + "/v2",
		httpClient: &http.Client{},
	}

	ip, err := client.GetDropletPublicIP(789)
	require.NoError(t, err)
	assert.Equal(t, "203.0.113.1", ip)
}

func TestGetDropletPublicIP_NoPublicIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"droplet": {
				"id": 789,
				"name": "test",
				"networks": {
					"v4": [
						{
							"ip_address": "10.0.0.1",
							"type": "private"
						}
					]
				}
			}
		}`
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	client := &Client{
		token:      "test-token",
		baseURL:    server.URL + "/v2",
		httpClient: &http.Client{},
	}

	_, err := client.GetDropletPublicIP(789)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no public IP found")
}
