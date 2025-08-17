package middleware

import (
	"net/http"
	"strings"
)

func Auth(bearerToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Check Bearer token format
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}

			// Extract token
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != bearerToken {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Token is valid, continue
			next.ServeHTTP(w, r)
		})
	}
}
