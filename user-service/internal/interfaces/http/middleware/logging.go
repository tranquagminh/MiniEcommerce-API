package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log request details
			duration := time.Since(start)

			event := logger.Info()

			// Add status level based on response code
			if wrapped.statusCode >= 500 {
				event = logger.Error()
			} else if wrapped.statusCode >= 400 {
				event = logger.Warn()
			}

			event.
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Int("status", wrapped.statusCode).
				Int("bytes", wrapped.bytes).
				Dur("duration_ms", duration).
				Msg("HTTP request")
		})
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(logger *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				// Generate a simple request ID (in production, use UUID)
				requestID = generateRequestID()
			}

			// Add request ID to response header
			w.Header().Set("X-Request-ID", requestID)

			// Add request ID to context logger
			ctx := r.Context()
			ctx = logger.With().Str("request_id", requestID).Logger().WithContext(ctx)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// Simple request ID generator (replace with UUID in production)
func generateRequestID() string {
	return time.Now().Format("20060102150405")
}
