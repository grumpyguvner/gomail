package middleware

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/grumpyguvner/gomail/internal/errors"
	"github.com/grumpyguvner/gomail/internal/metrics"
)

// TimeoutMiddleware wraps handlers with timeout functionality
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a channel to signal when the handler is done
			done := make(chan struct{})

			// Create a custom response writer to capture the status
			tw := &timeoutWriter{
				ResponseWriter: w,
				written:        false,
			}

			// Run the handler in a goroutine
			go func() {
				defer close(done)
				next.ServeHTTP(tw, r.WithContext(ctx))
			}()

			// Wait for either the handler to complete or timeout
			select {
			case <-done:
				// Handler completed successfully
				if tw.timedOut {
					// Response already sent by timeout handler
					return
				}
			case <-ctx.Done():
				// Timeout occurred
				tw.mu.Lock()
				if !tw.written {
					tw.written = true
					tw.timedOut = true
					tw.mu.Unlock()

					// Record timeout metric
					metrics.IncrementTimeouts(r.URL.Path)

					// Send timeout error response
					SendErrorResponse(w, errors.UnavailableError("Request timeout"))
				} else {
					tw.mu.Unlock()
				}
			}
		})
	}
}

// timeoutWriter wraps http.ResponseWriter to track if response has been written
type timeoutWriter struct {
	http.ResponseWriter
	written  bool
	timedOut bool
	mu       sync.Mutex
}

// WriteHeader intercepts status code writes
func (tw *timeoutWriter) WriteHeader(statusCode int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if !tw.written && !tw.timedOut {
		tw.written = true
		tw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write intercepts body writes
func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if !tw.written && !tw.timedOut {
		tw.written = true
	}

	if tw.timedOut {
		// Discard writes after timeout
		return len(b), nil
	}

	return tw.ResponseWriter.Write(b)
}

// Hijack implements http.Hijacker
func (tw *timeoutWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := tw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, errors.InternalError("ResponseWriter does not support hijacking", fmt.Errorf("hijacker not supported"))
}

// Flush implements http.Flusher
func (tw *timeoutWriter) Flush() {
	if flusher, ok := tw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
