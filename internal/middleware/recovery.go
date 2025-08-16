package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

// RecoveryMiddleware handles panics and recovers gracefully
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				stack := debug.Stack()
				log.Printf("PANIC recovered: %v\nRequest: %s %s\nStack trace:\n%s",
					err, r.Method, r.URL.Path, stack)

				// Get request ID if available
				requestID := r.Context().Value(RequestIDKey)
				if requestID != nil {
					log.Printf("Request ID: %v", requestID)
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
					log.Printf("Failed to write error response: %v", writeErr)
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
