package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorType represents the category of error
type ErrorType string

const (
	// ErrorTypeValidation indicates a validation error (400)
	ErrorTypeValidation ErrorType = "VALIDATION_ERROR"
	// ErrorTypeAuth indicates an authentication/authorization error (401)
	ErrorTypeAuth ErrorType = "AUTH_ERROR"
	// ErrorTypeRateLimit indicates rate limiting (429)
	ErrorTypeRateLimit ErrorType = "RATE_LIMIT_ERROR"
	// ErrorTypeStorage indicates a storage operation error (500)
	ErrorTypeStorage ErrorType = "STORAGE_ERROR"
	// ErrorTypeNetwork indicates a network error (502)
	ErrorTypeNetwork ErrorType = "NETWORK_ERROR"
	// ErrorTypeInternal indicates an internal server error (500)
	ErrorTypeInternal ErrorType = "INTERNAL_ERROR"
	// ErrorTypeNotFound indicates a resource not found (404)
	ErrorTypeNotFound ErrorType = "NOT_FOUND"
	// ErrorTypeBadRequest indicates a bad request (400)
	ErrorTypeBadRequest ErrorType = "BAD_REQUEST"
	// ErrorTypeConflict indicates a conflict (409)
	ErrorTypeConflict ErrorType = "CONFLICT"
	// ErrorTypeUnavailable indicates service unavailable (503)
	ErrorTypeUnavailable ErrorType = "SERVICE_UNAVAILABLE"
)

// AppError represents a categorized application error
type AppError struct {
	Type       ErrorType   `json:"type"`
	Message    string      `json:"message"`
	Details    interface{} `json:"details,omitempty"`
	StatusCode int         `json:"-"`
	Internal   error       `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %s (internal: %v)", e.Type, e.Message, e.Internal)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the internal error
func (e *AppError) Unwrap() error {
	return e.Internal
}

// HTTPStatus returns the HTTP status code for this error
func (e *AppError) HTTPStatus() int {
	if e.StatusCode > 0 {
		return e.StatusCode
	}
	return GetStatusCode(e.Type)
}

// ToJSON converts the error to JSON
func (e *AppError) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// errorStatusCodes maps error types to HTTP status codes
var errorStatusCodes = map[ErrorType]int{
	ErrorTypeValidation:  http.StatusBadRequest,
	ErrorTypeAuth:        http.StatusUnauthorized,
	ErrorTypeRateLimit:   http.StatusTooManyRequests,
	ErrorTypeStorage:     http.StatusInternalServerError,
	ErrorTypeNetwork:     http.StatusBadGateway,
	ErrorTypeInternal:    http.StatusInternalServerError,
	ErrorTypeNotFound:    http.StatusNotFound,
	ErrorTypeBadRequest:  http.StatusBadRequest,
	ErrorTypeConflict:    http.StatusConflict,
	ErrorTypeUnavailable: http.StatusServiceUnavailable,
}

// GetStatusCode returns the HTTP status code for an error type
func GetStatusCode(errorType ErrorType) int {
	if code, ok := errorStatusCodes[errorType]; ok {
		return code
	}
	return http.StatusInternalServerError
}

// New creates a new AppError
func New(errorType ErrorType, message string) *AppError {
	return &AppError{
		Type:       errorType,
		Message:    message,
		StatusCode: GetStatusCode(errorType),
	}
}

// NewWithDetails creates a new AppError with additional details
func NewWithDetails(errorType ErrorType, message string, details interface{}) *AppError {
	return &AppError{
		Type:       errorType,
		Message:    message,
		Details:    details,
		StatusCode: GetStatusCode(errorType),
	}
}

// Wrap creates a new AppError wrapping an existing error
func Wrap(errorType ErrorType, message string, err error) *AppError {
	return &AppError{
		Type:       errorType,
		Message:    message,
		StatusCode: GetStatusCode(errorType),
		Internal:   err,
	}
}

// WrapWithDetails creates a new AppError wrapping an existing error with details
func WrapWithDetails(errorType ErrorType, message string, err error, details interface{}) *AppError {
	return &AppError{
		Type:       errorType,
		Message:    message,
		Details:    details,
		StatusCode: GetStatusCode(errorType),
		Internal:   err,
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError attempts to cast an error to AppError
func AsAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
}

// ValidationError creates a validation error
func ValidationError(message string, details interface{}) *AppError {
	return NewWithDetails(ErrorTypeValidation, message, details)
}

// AuthError creates an authentication error
func AuthError(message string) *AppError {
	return New(ErrorTypeAuth, message)
}

// RateLimitError creates a rate limit error
func RateLimitError(message string, details interface{}) *AppError {
	return NewWithDetails(ErrorTypeRateLimit, message, details)
}

// StorageError creates a storage error
func StorageError(message string, err error) *AppError {
	return Wrap(ErrorTypeStorage, message, err)
}

// NetworkError creates a network error
func NetworkError(message string, err error) *AppError {
	return Wrap(ErrorTypeNetwork, message, err)
}

// InternalError creates an internal error
func InternalError(message string, err error) *AppError {
	return Wrap(ErrorTypeInternal, message, err)
}

// NotFoundError creates a not found error
func NotFoundError(message string) *AppError {
	return New(ErrorTypeNotFound, message)
}

// BadRequestError creates a bad request error
func BadRequestError(message string) *AppError {
	return New(ErrorTypeBadRequest, message)
}

// ConflictError creates a conflict error
func ConflictError(message string) *AppError {
	return New(ErrorTypeConflict, message)
}

// UnavailableError creates a service unavailable error
func UnavailableError(message string) *AppError {
	return New(ErrorTypeUnavailable, message)
}
