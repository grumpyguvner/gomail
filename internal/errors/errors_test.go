package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name         string
		errorType    ErrorType
		expectedCode int
	}{
		{"ValidationError", ErrorTypeValidation, http.StatusBadRequest},
		{"AuthError", ErrorTypeAuth, http.StatusUnauthorized},
		{"RateLimitError", ErrorTypeRateLimit, http.StatusTooManyRequests},
		{"StorageError", ErrorTypeStorage, http.StatusInternalServerError},
		{"NetworkError", ErrorTypeNetwork, http.StatusBadGateway},
		{"InternalError", ErrorTypeInternal, http.StatusInternalServerError},
		{"NotFoundError", ErrorTypeNotFound, http.StatusNotFound},
		{"BadRequestError", ErrorTypeBadRequest, http.StatusBadRequest},
		{"ConflictError", ErrorTypeConflict, http.StatusConflict},
		{"UnavailableError", ErrorTypeUnavailable, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GetStatusCode(tt.errorType)
			if code != tt.expectedCode {
				t.Errorf("GetStatusCode(%s) = %d; want %d", tt.errorType, code, tt.expectedCode)
			}
		})
	}
}

func TestNew(t *testing.T) {
	err := New(ErrorTypeValidation, "validation failed")

	if err.Type != ErrorTypeValidation {
		t.Errorf("err.Type = %s; want %s", err.Type, ErrorTypeValidation)
	}

	if err.Message != "validation failed" {
		t.Errorf("err.Message = %s; want 'validation failed'", err.Message)
	}

	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("err.StatusCode = %d; want %d", err.StatusCode, http.StatusBadRequest)
	}

	if err.Error() != "VALIDATION_ERROR: validation failed" {
		t.Errorf("err.Error() = %s; want 'VALIDATION_ERROR: validation failed'", err.Error())
	}
}

func TestNewWithDetails(t *testing.T) {
	details := map[string]string{"field": "email", "reason": "invalid format"}
	err := NewWithDetails(ErrorTypeValidation, "validation failed", details)

	if err.Details == nil {
		t.Error("err.Details should not be nil")
	}

	detailsMap, ok := err.Details.(map[string]string)
	if !ok {
		t.Error("err.Details should be map[string]string")
	}

	if detailsMap["field"] != "email" {
		t.Errorf("details[field] = %s; want 'email'", detailsMap["field"])
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("database connection failed")
	err := Wrap(ErrorTypeStorage, "storage operation failed", originalErr)

	if err.Internal != originalErr {
		t.Error("err.Internal should be the original error")
	}

	if err.Unwrap() != originalErr {
		t.Error("Unwrap() should return the original error")
	}

	expectedMsg := "STORAGE_ERROR: storage operation failed (internal: database connection failed)"
	if err.Error() != expectedMsg {
		t.Errorf("err.Error() = %s; want %s", err.Error(), expectedMsg)
	}
}

func TestHTTPStatus(t *testing.T) {
	// Test default status code
	err := New(ErrorTypeValidation, "test")
	if err.HTTPStatus() != http.StatusBadRequest {
		t.Errorf("HTTPStatus() = %d; want %d", err.HTTPStatus(), http.StatusBadRequest)
	}

	// Test custom status code
	err.StatusCode = http.StatusTeapot
	if err.HTTPStatus() != http.StatusTeapot {
		t.Errorf("HTTPStatus() with custom code = %d; want %d", err.HTTPStatus(), http.StatusTeapot)
	}
}

func TestToJSON(t *testing.T) {
	details := map[string]interface{}{
		"field": "email",
		"value": "invalid",
	}
	err := NewWithDetails(ErrorTypeValidation, "validation failed", details)

	jsonData, jsonErr := err.ToJSON()
	if jsonErr != nil {
		t.Fatalf("ToJSON() error = %v", jsonErr)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(jsonData, &result); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", unmarshalErr)
	}

	if result["type"] != string(ErrorTypeValidation) {
		t.Errorf("JSON type = %s; want %s", result["type"], ErrorTypeValidation)
	}

	if result["message"] != "validation failed" {
		t.Errorf("JSON message = %s; want 'validation failed'", result["message"])
	}

	if result["details"] == nil {
		t.Error("JSON details should not be nil")
	}
}

func TestIsAppError(t *testing.T) {
	appErr := New(ErrorTypeValidation, "test")
	stdErr := errors.New("standard error")

	if !IsAppError(appErr) {
		t.Error("IsAppError should return true for AppError")
	}

	if IsAppError(stdErr) {
		t.Error("IsAppError should return false for standard error")
	}
}

func TestAsAppError(t *testing.T) {
	appErr := New(ErrorTypeValidation, "test")
	stdErr := errors.New("standard error")

	// Test with AppError
	result, ok := AsAppError(appErr)
	if !ok {
		t.Error("AsAppError should return true for AppError")
	}
	if result != appErr {
		t.Error("AsAppError should return the same AppError instance")
	}

	// Test with standard error
	result, ok = AsAppError(stdErr)
	if ok {
		t.Error("AsAppError should return false for standard error")
	}
	if result != nil {
		t.Error("AsAppError should return nil for standard error")
	}
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() *AppError
		expected ErrorType
	}{
		{
			name:     "ValidationError",
			fn:       func() *AppError { return ValidationError("test", nil) },
			expected: ErrorTypeValidation,
		},
		{
			name:     "AuthError",
			fn:       func() *AppError { return AuthError("test") },
			expected: ErrorTypeAuth,
		},
		{
			name:     "RateLimitError",
			fn:       func() *AppError { return RateLimitError("test", nil) },
			expected: ErrorTypeRateLimit,
		},
		{
			name:     "StorageError",
			fn:       func() *AppError { return StorageError("test", nil) },
			expected: ErrorTypeStorage,
		},
		{
			name:     "NetworkError",
			fn:       func() *AppError { return NetworkError("test", nil) },
			expected: ErrorTypeNetwork,
		},
		{
			name:     "InternalError",
			fn:       func() *AppError { return InternalError("test", nil) },
			expected: ErrorTypeInternal,
		},
		{
			name:     "NotFoundError",
			fn:       func() *AppError { return NotFoundError("test") },
			expected: ErrorTypeNotFound,
		},
		{
			name:     "BadRequestError",
			fn:       func() *AppError { return BadRequestError("test") },
			expected: ErrorTypeBadRequest,
		},
		{
			name:     "ConflictError",
			fn:       func() *AppError { return ConflictError("test") },
			expected: ErrorTypeConflict,
		},
		{
			name:     "UnavailableError",
			fn:       func() *AppError { return UnavailableError("test") },
			expected: ErrorTypeUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err.Type != tt.expected {
				t.Errorf("%s created error with type %s; want %s", tt.name, err.Type, tt.expected)
			}
		})
	}
}

func TestGetStatusCodeUnknown(t *testing.T) {
	// Test with an unknown error type
	unknownType := ErrorType("UNKNOWN_ERROR")
	code := GetStatusCode(unknownType)
	if code != http.StatusInternalServerError {
		t.Errorf("GetStatusCode(unknown) = %d; want %d", code, http.StatusInternalServerError)
	}
}
