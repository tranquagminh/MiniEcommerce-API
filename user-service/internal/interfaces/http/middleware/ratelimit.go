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
	visitors          map[string]*visitor
	mu                sync.RWMutex
	limit             rate.Limit
	burst             int
	ttl               time.Duration
	trustProxies      bool
	trustedProxyNets  []*net.IPNet
}

// visitor holds the rate limiter and last seen time for each visitor
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond float64, burst int, ttl time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors:         make(map[string]*visitor),
		limit:            rate.Limit(requestsPerSecond),
		burst:            burst,
		ttl:              ttl,
		trustProxies:     false,
		trustedProxyNets: nil,
	}

	// Cleanup goroutine để xóa các visitors cũ
	go rl.cleanupVisitors()

	return rl
}

// NewRateLimiterWithProxyConfig creates a new rate limiter with proxy configuration
func NewRateLimiterWithProxyConfig(requestsPerSecond float64, burst int, ttl time.Duration, trustProxies bool, trustedCIDRs []string) *RateLimiter {
	var trustedNets []*net.IPNet

	if trustProxies && len(trustedCIDRs) > 0 {
		for _, cidr := range trustedCIDRs {
			_, ipNet, err := net.ParseCIDR(cidr)
			if err != nil {
				// Log error but continue - invalid CIDR will be skipped
				fmt.Printf("Warning: Invalid CIDR %s: %v\n", cidr, err)
				continue
			}
			trustedNets = append(trustedNets, ipNet)
		}
	}

	rl := &RateLimiter{
		visitors:         make(map[string]*visitor),
		limit:            rate.Limit(requestsPerSecond),
		burst:            burst,
		ttl:              ttl,
		trustProxies:     trustProxies,
		trustedProxyNets: trustedNets,
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
			ip := limiter.getClientIP(r)

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
			ip := limiter.getClientIP(r)
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

// getClientIP extracts the real client IP from the request with security checks
func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Extract the direct connection IP (from RemoteAddr)
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If parsing fails, use RemoteAddr as-is
		remoteIP = r.RemoteAddr
	}

	// Parse the remote IP
	directIP := net.ParseIP(remoteIP)
	if directIP == nil {
		// Invalid IP, return as-is for rate limiting
		return remoteIP
	}

	// Only trust proxy headers if explicitly configured AND the request comes from a trusted proxy
	if rl.trustProxies && rl.isTrustedProxy(directIP) {
		// Check X-Forwarded-For header
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			ips := strings.Split(xff, ",")
			// Take the first IP in the chain (the original client)
			if len(ips) > 0 {
				clientIP := strings.TrimSpace(ips[0])
				// Validate it's a proper IP
				if net.ParseIP(clientIP) != nil {
					return clientIP
				}
			}
		}

		// Check X-Real-IP header as fallback
		xri := r.Header.Get("X-Real-IP")
		if xri != "" && net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Fall back to direct connection IP (safe default)
	return remoteIP
}

// isTrustedProxy checks if the given IP is in the trusted proxy list
func (rl *RateLimiter) isTrustedProxy(ip net.IP) bool {
	if len(rl.trustedProxyNets) == 0 {
		return false
	}

	for _, ipNet := range rl.trustedProxyNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
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
