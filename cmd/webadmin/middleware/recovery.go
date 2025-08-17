package middleware

import (
	"net/http"
	"runtime"

	"github.com/grumpyguvner/gomail/cmd/webadmin/logging"
)

func Recovery(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Get stack trace
					buf := make([]byte, 4096)
					n := runtime.Stack(buf, false)
					stack := string(buf[:n])

					logger.Error("Panic recovered",
						"error", err,
						"path", r.URL.Path,
						"method", r.Method,
						"stack", stack,
					)

					// Return 500 error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}