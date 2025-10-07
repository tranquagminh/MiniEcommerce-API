package middleware

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter stores the rate limiters for each visitor
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	limit    rate.Limit
	burst    int
	ttl      time.Duration
}

// visitor holds the rate limiter and last seen time for each visitor
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond float64, burst int, ttl time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    rate.Limit(requestsPerSecond),
		burst:    burst,
		ttl:      ttl,
	}

	// Cleanup goroutine để xóa các visitors cũ
	go rl.cleanupVisitors()

	return rl
}

// getVisitor returns the rate limiter for the given IP
func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.limit, rl.burst)
		rl.visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update last seen time
	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors removes old entries from the visitors map
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C

		// Collect expired IPs first
		rl.mu.RLock()
		var expiredIPs []string
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.ttl {
				expiredIPs = append(expiredIPs, ip)
			}
		}
		rl.mu.RUnlock()

		// Delete with write lock
		if len(expiredIPs) > 0 {
			rl.mu.Lock()
			for _, ip := range expiredIPs {
				delete(rl.visitors, ip)
			}
			rl.mu.Unlock()
		}
	}
}

// RateLimitMiddleware creates a new rate limiting middleware
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			// Get the rate limiter for this IP
			l := limiter.getVisitor(ip)

			if !l.Allow() {
				rateLimitExceededResponse(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Per-route rate limiting với config khác nhau
func CustomRateLimitMiddleware(requestsPerSecond float64, burst int) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(requestsPerSecond, burst, 30*time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			l := limiter.getVisitor(ip)

			if !l.Allow() {
				rateLimitExceededResponse(w)
				return
			}

			// Add rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", requestsPerSecond))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", int(l.Tokens())))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		// Take the first IP in the chain
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

// rateLimitExceededResponse sends a 429 Too Many Requests response
func rateLimitExceededResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	response := map[string]interface{}{
		"error":   "rate_limit_exceeded",
		"message": "Too many requests. Please try again later.",
	}

	json.NewEncoder(w).Encode(response)
}

// UserRateLimitMiddleware limits requests per authenticated user
func UserRateLimitMiddleware(requestsPerSecond float64, burst int) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(requestsPerSecond, burst, 30*time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context (set by AuthMiddleware)
			userID := GetUserID(r)
			if userID == 0 {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Use user ID as key instead of IP
			key := fmt.Sprintf("user:%d", userID)
			l := limiter.getVisitor(key)

			if !l.Allow() {
				rateLimitExceededResponse(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
