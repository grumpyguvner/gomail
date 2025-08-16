package middleware

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/grumpyguvner/gomail/internal/errors"
	"github.com/grumpyguvner/gomail/internal/metrics"
)

// ErrorResponse represents the structure of error responses
type ErrorResponse struct {
	Error   bool        `json:"error"`
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ErrorHandlerMiddleware provides centralized error handling
func ErrorHandlerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a custom response writer to intercept errors
		wrapped := &errorResponseWriter{
			ResponseWriter: w,
			request:        r,
			written:        false,
		}

		// Defer panic recovery
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered in %s %s: %v", r.Method, r.URL.Path, err)

				// Track panic as internal error
				metrics.RecordError(string(errors.ErrorTypeInternal), r.URL.Path)

				// Send internal error response
				SendErrorResponse(w, errors.InternalError("Internal server error", nil))
			}
		}()

		next.ServeHTTP(wrapped, r)
	})
}

// errorResponseWriter wraps http.ResponseWriter to track if response has been written
type errorResponseWriter struct {
	http.ResponseWriter
	request *http.Request
	written bool
}

func (w *errorResponseWriter) WriteHeader(statusCode int) {
	w.written = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *errorResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

// SendErrorResponse sends a structured error response
func SendErrorResponse(w http.ResponseWriter, err error) {
	var response ErrorResponse
	var statusCode int

	// Check if it's an AppError
	if appErr, ok := errors.AsAppError(err); ok {
		response = ErrorResponse{
			Error:   true,
			Type:    string(appErr.Type),
			Message: appErr.Message,
			Details: appErr.Details,
		}
		statusCode = appErr.HTTPStatus()

		// Record error metric
		metrics.RecordError(string(appErr.Type), "")

		// Log based on error type
		if appErr.Type == errors.ErrorTypeInternal || appErr.Type == errors.ErrorTypeStorage {
			log.Printf("ERROR [%s]: %s - Internal: %v", appErr.Type, appErr.Message, appErr.Internal)
		} else {
			log.Printf("INFO [%s]: %s", appErr.Type, appErr.Message)
		}
	} else {
		// Handle non-AppError errors
		response = ErrorResponse{
			Error:   true,
			Type:    string(errors.ErrorTypeInternal),
			Message: "An error occurred",
		}
		statusCode = http.StatusInternalServerError

		// Record as internal error
		metrics.RecordError(string(errors.ErrorTypeInternal), "")

		log.Printf("ERROR [INTERNAL]: Unhandled error: %v", err)
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Encode response
	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		log.Printf("Failed to encode error response: %v", encodeErr)
	}
}

// HandleError is a helper function to handle errors in handlers
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	// If it's already an AppError, send it directly
	if _, ok := errors.AsAppError(err); ok {
		SendErrorResponse(w, err)
		return
	}

	// Otherwise wrap it as an internal error
	SendErrorResponse(w, errors.InternalError("An error occurred processing your request", err))
}
