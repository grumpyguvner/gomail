package digitalocean

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckDomainExists(t *testing.T) {
	tests := []struct {
		name       string
		domain     string
		statusCode int
		response   string
		expected   bool
		expectErr  bool
	}{
		{
			name:       "domain exists",
			domain:     "example.com",
			statusCode: 200,
			response:   `{"domain": {"name": "example.com"}}`,
			expected:   true,
			expectErr:  false,
		},
		{
			name:       "domain not found",
			domain:     "notfound.com",
			statusCode: 404,
			response:   `{"id": "not_found", "message": "Domain not found"}`,
			expected:   false,
			expectErr:  false,
		},
		{
			name:       "API error",
			domain:     "error.com",
			statusCode: 500,
			response:   `{"message": "Internal server error"}`,
			expected:   false,
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v2/domains/"+tt.domain, r.URL.Path)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := &Client{
				token:      "test-token",
				baseURL:    server.URL + "/v2",
				httpClient: &http.Client{},
			}

			exists, err := client.CheckDomainExists(tt.domain)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, exists)
			}
		})
	}
}

func TestCreateDomain(t *testing.T) {
	tests := []struct {
		name       string
		domain     string
		ip         string
		statusCode int
		response   string
		expectErr  bool
	}{
		{
			name:       "successful creation",
			domain:     "example.com",
			ip:         "192.168.1.1",
			statusCode: 201,
			response:   `{"domain": {"name": "example.com"}}`,
			expectErr:  false,
		},
		{
			name:       "domain already exists",
			domain:     "existing.com",
			ip:         "192.168.1.1",
			statusCode: 422,
			response:   `{"id": "domain_exists", "message": "Domain already exists"}`,
			expectErr:  false, // Should not error when domain exists
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v2/domains", r.URL.Path)
				assert.Equal(t, "POST", r.Method)

				var body map[string]string
				json.NewDecoder(r.Body).Decode(&body)
				assert.Equal(t, tt.domain, body["name"])
				assert.Equal(t, tt.ip, body["ip_address"])

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := &Client{
				token:      "test-token",
				baseURL:    server.URL + "/v2",
				httpClient: &http.Client{},
			}

			err := client.CreateDomain(tt.domain, tt.ip)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpsertDNSRecord(t *testing.T) {
	t.Run("creates new record", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// First call: get existing records (none found)
				assert.Equal(t, "/v2/domains/example.com/records", r.URL.Path)
				assert.Equal(t, "GET", r.Method)
				w.Write([]byte(`{"domain_records": []}`))
			} else {
				// Second call: create new record
				assert.Equal(t, "/v2/domains/example.com/records", r.URL.Path)
				assert.Equal(t, "POST", r.Method)

				var record DNSRecord
				json.NewDecoder(r.Body).Decode(&record)
				assert.Equal(t, "MX", record.Type)
				assert.Equal(t, "@", record.Name)
				assert.Equal(t, "mail.example.com.", record.Data)
				assert.Equal(t, 10, record.Priority)

				w.Write([]byte(`{"domain_record": {"id": 123, "type": "MX", "name": "@"}}`))
			}
		}))
		defer server.Close()

		client := &Client{
			token:      "test-token",
			baseURL:    server.URL + "/v2",
			httpClient: &http.Client{},
		}

		record := DNSRecord{
			Type:     "MX",
			Name:     "@",
			Data:     "mail.example.com.",
			Priority: 10,
		}

		err := client.UpsertDNSRecord("example.com", record)
		assert.NoError(t, err)
		assert.Equal(t, 2, callCount)
	})

	t.Run("updates existing record", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// First call: get existing records (one found)
				assert.Equal(t, "/v2/domains/example.com/records", r.URL.Path)
				assert.Equal(t, "GET", r.Method)
				w.Write([]byte(`{
					"domain_records": [
						{"id": 456, "type": "MX", "name": "@", "data": "old.example.com.", "priority": 20}
					]
				}`))
			} else {
				// Second call: update existing record
				assert.Equal(t, "/v2/domains/example.com/records/456", r.URL.Path)
				assert.Equal(t, "PUT", r.Method)

				var record DNSRecord
				json.NewDecoder(r.Body).Decode(&record)
				assert.Equal(t, 456, record.ID)
				assert.Equal(t, "MX", record.Type)
				assert.Equal(t, "@", record.Name)
				assert.Equal(t, "mail.example.com.", record.Data)

				w.Write([]byte(`{"domain_record": {"id": 456, "type": "MX", "name": "@"}}`))
			}
		}))
		defer server.Close()

		client := &Client{
			token:      "test-token",
			baseURL:    server.URL + "/v2",
			httpClient: &http.Client{},
		}

		record := DNSRecord{
			Type:     "MX",
			Name:     "@",
			Data:     "mail.example.com.",
			Priority: 10,
		}

		err := client.UpsertDNSRecord("example.com", record)
		assert.NoError(t, err)
		assert.Equal(t, 2, callCount)
	})
}

func TestSetupMailDNS(t *testing.T) {
	requestLog := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestLog = append(requestLog, r.Method+" "+r.URL.Path)

		if r.URL.Path == "/v2/domains/example.com" && r.Method == "GET" {
			w.WriteHeader(404) // Domain doesn't exist
		} else if r.URL.Path == "/v2/domains" && r.Method == "POST" {
			w.WriteHeader(201)
			w.Write([]byte(`{"domain": {"name": "example.com"}}`))
		} else if r.URL.Path == "/v2/domains/example.com/records" && r.Method == "GET" {
			w.Write([]byte(`{"domain_records": []}`)) // No existing records
		} else if r.URL.Path == "/v2/domains/example.com/records" && r.Method == "POST" {
			w.Write([]byte(`{"domain_record": {"id": 1}}`))
		}
	}))
	defer server.Close()

	client := &Client{
		token:      "test-token",
		baseURL:    server.URL + "/v2",
		httpClient: &http.Client{},
	}

	err := client.SetupMailDNS("example.com", "mail.example.com", "192.168.1.1")
	require.NoError(t, err)

	// Verify the expected API calls were made
	assert.Contains(t, requestLog, "GET /v2/domains/example.com")   // Check domain
	assert.Contains(t, requestLog, "POST /v2/domains")              // Create domain
	assert.Contains(t, requestLog, "GET /v2/domains/example.com/records") // Check for existing records
	
	// Should create A, MX, SPF, and DMARC records
	postCount := 0
	for _, req := range requestLog {
		if req == "POST /v2/domains/example.com/records" {
			postCount++
		}
	}
	assert.Equal(t, 4, postCount, "Should create 4 DNS records (A, MX, SPF, DMARC)")
}