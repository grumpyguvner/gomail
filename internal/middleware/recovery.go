package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/grumpyguvner/gomail/internal/logging"
)

// RecoveryMiddleware handles panics and recovers gracefully
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				stack := debug.Stack()
				logger := logging.Get()

				// Try to get request ID from response header first (set by RequestIDMiddleware)
				// This works even when RecoveryMiddleware wraps RequestIDMiddleware
				var reqIDStr string
				if reqID := w.Header().Get(RequestIDHeader); reqID != "" {
					reqIDStr = reqID
					logger = logging.WithRequestID(reqIDStr)
				} else {
					// Fallback to context (for when RecoveryMiddleware is inside RequestIDMiddleware)
					if requestID := r.Context().Value(RequestIDKey); requestID != nil {
						if s, ok := requestID.(string); ok {
							reqIDStr = s
							logger = logging.WithRequestID(reqIDStr)
						}
					}
				}

				logger.Errorw("PANIC recovered",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
					"stack_trace", string(stack))

				// Return a generic error response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				response := `{"error":"Internal Server Error","message":"An unexpected error occurred"}`
				if reqIDStr != "" {
					response = fmt.Sprintf(`{"error":"Internal Server Error","message":"An unexpected error occurred","request_id":"%s"}`, reqIDStr)
				}

				_, writeErr := w.Write([]byte(response))
				if writeErr != nil {
					logger.Errorf("Failed to write error response: %v", writeErr)
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RecoveryWithLogger creates a recovery middleware with a custom logger
func RecoveryWithLogger(logger func(format string, args ...interface{})) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log the panic with stack trace
					stack := debug.Stack()
					logger("PANIC recovered: %v\nRequest: %s %s\nStack trace:\n%s",
						err, r.Method, r.URL.Path, stack)

					// Get request ID if available
					requestID := r.Context().Value(RequestIDKey)
					if requestID != nil {
						logger("Request ID: %v", requestID)
					}

					// Return a generic error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)

					response := `{"error":"Internal Server Error","message":"An unexpected error occurred"}`
					if requestID != nil {
						response = fmt.Sprintf(`{"error":"Internal Server Error","message":"An unexpected error occurred","request_id":"%v"}`, requestID)
					}

					_, writeErr := w.Write([]byte(response))
					if writeErr != nil {
						logger("Failed to write error response: %v", writeErr)
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
