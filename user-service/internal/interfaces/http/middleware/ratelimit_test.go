// internal/interfaces/http/middleware/ratelimit_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	// Create rate limiter: 2 requests per second
	rl := NewRateLimiter(2, 2, time.Minute)

	handler := RateLimitMiddleware(rl)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	// Test 5 requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// First 2 should pass
		if i < 2 {
			if rr.Code != http.StatusOK {
				t.Errorf("Request %d: expected 200, got %d", i+1, rr.Code)
			}
		} else {
			// Rest should be rate limited
			if rr.Code != http.StatusTooManyRequests {
				t.Errorf("Request %d: expected 429, got %d", i+1, rr.Code)
			}
		}
	}
}
