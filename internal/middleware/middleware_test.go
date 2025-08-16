package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "normal request - no panic",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, "OK", rec.Body.String())
			},
		},
		{
			name: "panic recovery",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("test panic")
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Contains(t, rec.Body.String(), "Internal Server Error")
				assert.Contains(t, rec.Body.String(), "An unexpected error occurred")
				assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			},
		},
		{
			name: "panic with request ID",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("test panic with ID")
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Contains(t, rec.Body.String(), "request_id")
				assert.Contains(t, rec.Body.String(), "test-request-id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the middleware chain
			handler := RecoveryMiddleware(tt.handler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)

			// Add request ID for the third test
			if strings.Contains(tt.name, "request ID") {
				ctx := context.WithValue(req.Context(), RequestIDKey, "test-request-id")
				req = req.WithContext(ctx)
			}

			// Create response recorder
			rec := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rec, req)

			// Check results
			assert.Equal(t, tt.expectedStatus, rec.Code)
			tt.checkResponse(t, rec)
		})
	}
}

func TestRecoveryWithLogger(t *testing.T) {
	var loggedMessages []string
	customLogger := func(format string, args ...interface{}) {
		loggedMessages = append(loggedMessages, fmt.Sprintf(format, args...))
	}

	middleware := RecoveryWithLogger(customLogger)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("custom logger test")
	})

	handler := middleware(panicHandler)

	req := httptest.NewRequest("POST", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check response
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal Server Error")

	// Check logged messages
	require.NotEmpty(t, loggedMessages)
	assert.Contains(t, loggedMessages[0], "PANIC recovered: custom logger test")
	assert.Contains(t, loggedMessages[0], "POST /api/test")
	assert.Contains(t, loggedMessages[0], "Stack trace:")
}

func TestRequestIDMiddleware(t *testing.T) {
	tests := []struct {
		name              string
		existingRequestID string
		checkRequestID    func(*testing.T, string)
	}{
		{
			name:              "generates new request ID",
			existingRequestID: "",
			checkRequestID: func(t *testing.T, id string) {
				assert.NotEmpty(t, id)
				assert.True(t, strings.HasPrefix(id, "req_"))
			},
		},
		{
			name:              "uses existing request ID",
			existingRequestID: "existing-id-123",
			checkRequestID: func(t *testing.T, id string) {
				assert.Equal(t, "existing-id-123", id)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRequestID string

			handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequestID = GetRequestIDFromRequest(r)
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.existingRequestID != "" {
				req.Header.Set(RequestIDHeader, tt.existingRequestID)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			// Check that request ID was added to context
			tt.checkRequestID(t, capturedRequestID)

			// Check that request ID was added to response header
			responseID := rec.Header().Get(RequestIDHeader)
			assert.Equal(t, capturedRequestID, responseID)
		})
	}
}

func TestGenerateRequestID(t *testing.T) {
	// Generate multiple IDs and ensure they're unique
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateRequestID()
		assert.True(t, strings.HasPrefix(id, "req_"))
		assert.False(t, ids[id], "Duplicate ID generated: %s", id)
		ids[id] = true
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{
			name:     "with request ID",
			ctx:      context.WithValue(context.Background(), RequestIDKey, "test-id"),
			expected: "test-id",
		},
		{
			name:     "without request ID",
			ctx:      context.Background(),
			expected: "",
		},
		{
			name:     "with wrong type",
			ctx:      context.WithValue(context.Background(), RequestIDKey, 123),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRequestID(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMiddlewareChain(t *testing.T) {
	// Test that both middlewares work together
	var capturedRequestID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = GetRequestIDFromRequest(r)
		// Simulate a panic
		panic("middleware chain test")
	})

	// Chain the middlewares
	withRequestID := RequestIDMiddleware(handler)
	withRecovery := RecoveryMiddleware(withRequestID)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	withRecovery.ServeHTTP(rec, req)

	// Check that request ID was generated
	assert.NotEmpty(t, capturedRequestID)

	// Check that panic was recovered
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Internal Server Error")

	// Check that response includes request ID
	assert.Contains(t, rec.Body.String(), capturedRequestID)
	assert.Equal(t, capturedRequestID, rec.Header().Get(RequestIDHeader))
}

func BenchmarkRequestIDMiddleware(b *testing.B) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

func BenchmarkRecoveryMiddleware(b *testing.B) {
	handler := RecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
