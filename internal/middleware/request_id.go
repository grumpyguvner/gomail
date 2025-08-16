package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// ContextKey is a type for context keys
type ContextKey string

// RequestIDKey is the context key for request IDs
const RequestIDKey ContextKey = "request_id"

// RequestIDHeader is the HTTP header for request IDs
const RequestIDHeader = "X-Request-ID"

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request already has an ID (from client or proxy)
		requestID := r.Header.Get(RequestIDHeader)

		// Generate new ID if not present
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Add to request context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		r = r.WithContext(ctx)

		// Add to response header
		w.Header().Set(RequestIDHeader, requestID)

		next.ServeHTTP(w, r)
	})
}

// generateRequestID creates a new unique request ID
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if random fails
		return "req_fallback"
	}
	return "req_" + hex.EncodeToString(bytes)
}

// GetRequestID extracts the request ID from the context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetRequestIDFromRequest extracts the request ID from an HTTP request
func GetRequestIDFromRequest(r *http.Request) string {
	return GetRequestID(r.Context())
}
