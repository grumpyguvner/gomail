package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grumpyguvner/gomail/internal/errors"
)

func TestSendErrorResponse(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedType   string
	}{
		{
			name:           "ValidationError",
			err:            errors.ValidationError("Invalid input", nil),
			expectedStatus: http.StatusBadRequest,
			expectedType:   "VALIDATION_ERROR",
		},
		{
			name:           "AuthError",
			err:            errors.AuthError("Unauthorized"),
			expectedStatus: http.StatusUnauthorized,
			expectedType:   "AUTH_ERROR",
		},
		{
			name:           "RateLimitError",
			err:            errors.RateLimitError("Too many requests", nil),
			expectedStatus: http.StatusTooManyRequests,
			expectedType:   "RATE_LIMIT_ERROR",
		},
		{
			name:           "NotFoundError",
			err:            errors.NotFoundError("Resource not found"),
			expectedStatus: http.StatusNotFound,
			expectedType:   "NOT_FOUND",
		},
		{
			name:           "StandardError",
			err:            http.ErrBodyNotAllowed,
			expectedStatus: http.StatusInternalServerError,
			expectedType:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			SendErrorResponse(w, tt.err)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Status code = %d; want %d", w.Code, tt.expectedStatus)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %s; want application/json", contentType)
			}

			// Parse response
			var response ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check error flag
			if !response.Error {
				t.Error("Error flag should be true")
			}

			// Check error type
			if response.Type != tt.expectedType {
				t.Errorf("Error type = %s; want %s", response.Type, tt.expectedType)
			}

			// Check message exists
			if response.Message == "" {
				t.Error("Error message should not be empty")
			}
		})
	}
}

func TestErrorHandlerMiddleware(t *testing.T) {
	// Create a handler that returns success
	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	// Wrap with error handler middleware
	wrapped := ErrorHandlerMiddleware(successHandler)

	// Test successful request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d; want %d", w.Code, http.StatusOK)
	}

	if w.Body.String() != "success" {
		t.Errorf("Response body = %s; want 'success'", w.Body.String())
	}
}

func TestErrorHandlerMiddlewarePanic(t *testing.T) {
	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Wrap with error handler middleware
	wrapped := ErrorHandlerMiddleware(panicHandler)

	// Test request that causes panic
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// This should not panic due to recovery
	wrapped.ServeHTTP(w, req)

	// Should return internal server error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d; want %d", w.Code, http.StatusInternalServerError)
	}

	// Parse response
	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Type != "INTERNAL_ERROR" {
		t.Errorf("Error type = %s; want INTERNAL_ERROR", response.Type)
	}
}

func TestHandleError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		shouldRespond  bool
	}{
		{
			name:          "NilError",
			err:           nil,
			shouldRespond: false,
		},
		{
			name:           "AppError",
			err:            errors.ValidationError("test", nil),
			expectedStatus: http.StatusBadRequest,
			shouldRespond:  true,
		},
		{
			name:           "StandardError",
			err:            http.ErrBodyNotAllowed,
			expectedStatus: http.StatusInternalServerError,
			shouldRespond:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)

			HandleError(w, req, tt.err)

			if tt.shouldRespond {
				if w.Code != tt.expectedStatus {
					t.Errorf("Status code = %d; want %d", w.Code, tt.expectedStatus)
				}

				// Check that response has content
				if w.Body.Len() == 0 {
					t.Error("Response body should not be empty")
				}
			} else {
				// For nil error, nothing should be written
				if w.Code != 200 { // Default for httptest.NewRecorder
					t.Error("No response should be written for nil error")
				}
			}
		})
	}
}

func TestErrorResponseWriterWriteHeader(t *testing.T) {
	base := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)

	wrapper := &errorResponseWriter{
		ResponseWriter: base,
		request:        req,
		written:        false,
	}

	// Test WriteHeader
	wrapper.WriteHeader(http.StatusCreated)

	if !wrapper.written {
		t.Error("written flag should be true after WriteHeader")
	}

	if base.Code != http.StatusCreated {
		t.Errorf("Status code = %d; want %d", base.Code, http.StatusCreated)
	}
}

func TestErrorResponseWriterWrite(t *testing.T) {
	base := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)

	wrapper := &errorResponseWriter{
		ResponseWriter: base,
		request:        req,
		written:        false,
	}

	// Test Write
	data := []byte("test data")
	n, err := wrapper.Write(data)

	if err != nil {
		t.Errorf("Write error = %v; want nil", err)
	}

	if n != len(data) {
		t.Errorf("Bytes written = %d; want %d", n, len(data))
	}

	if !wrapper.written {
		t.Error("written flag should be true after Write")
	}

	if base.Body.String() != "test data" {
		t.Errorf("Response body = %s; want 'test data'", base.Body.String())
	}
}
