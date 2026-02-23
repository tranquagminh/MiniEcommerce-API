package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"user-service/internal/infrastructure/redis"
)

type RedisRateLimiter struct {
	client           *redis.RedisClient
	limit            int
	window           time.Duration
	trustProxies     bool
	trustedProxyNets []*net.IPNet
}

func NewRedisRateLimiter(client *redis.RedisClient, limit int, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:           client,
		limit:            limit,
		window:           window,
		trustProxies:     false,
		trustedProxyNets: nil,
	}
}

// NewRedisRateLimiterWithProxyConfig creates a Redis rate limiter with proxy configuration
func NewRedisRateLimiterWithProxyConfig(client *redis.RedisClient, limit int, window time.Duration, trustProxies bool, trustedCIDRs []string) *RedisRateLimiter {
	var trustedNets []*net.IPNet

	if trustProxies && len(trustedCIDRs) > 0 {
		for _, cidr := range trustedCIDRs {
			_, ipNet, err := net.ParseCIDR(cidr)
			if err != nil {
				fmt.Printf("Warning: Invalid CIDR %s: %v\n", cidr, err)
				continue
			}
			trustedNets = append(trustedNets, ipNet)
		}
	}

	return &RedisRateLimiter{
		client:           client,
		limit:            limit,
		window:           window,
		trustProxies:     trustProxies,
		trustedProxyNets: trustedNets,
	}
}

func (rl *RedisRateLimiter) Allow(ctx context.Context, identifier string) (bool, error) {
	key := fmt.Sprintf("rate_limit:%s", identifier)

	// Use pipeline for atomic operations
	pipe := rl.client.Pipeline()

	// Increment counter
	incr := pipe.Incr(ctx, key)
	// Set expiration only if key doesn't exist
	pipe.Expire(ctx, key, rl.window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("redis pipeline error: %w", err)
	}

	count, err := incr.Result()
	if err != nil {
		return false, fmt.Errorf("failed to get incr result: %w", err)
	}

	return count <= int64(rl.limit), nil
}

// RedisRateLimitMiddleware using Redis
func RedisRateLimitMiddleware(rl *RedisRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ip := rl.getClientIP(r)

			allowed, err := rl.Allow(ctx, ip)
			if err != nil {
				// Fallback to allow request if Redis is down
				// Log error for monitoring
				fmt.Printf("Redis rate limit error: %v\n", err)
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "rate_limit_exceeded",
					"message": "Too many requests. Please try again later.",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the real client IP from the request with security checks
func (rl *RedisRateLimiter) getClientIP(r *http.Request) string {
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
func (rl *RedisRateLimiter) isTrustedProxy(ip net.IP) bool {
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

// Custom Redis rate limiter for different endpoints
func CustomRedisRateLimitMiddleware(client *redis.RedisClient, limit int, window time.Duration) func(http.Handler) http.Handler {
	rl := NewRedisRateLimiter(client, limit, window)
	return RedisRateLimitMiddleware(rl)
}

// RedisUserRateLimitMiddleware - rate limit based on authenticated user ID
func RedisUserRateLimitMiddleware(client *redis.RedisClient, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context
			userID := GetUserID(r)
			if userID == 0 {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Create rate limiter with user-specific key
			rl := NewRedisRateLimiter(client, limit, window)
			identifier := fmt.Sprintf("user:%d:%s", userID, r.URL.Path)

			ctx := r.Context()
			allowed, err := rl.Allow(ctx, identifier)
			if err != nil {
				// Log error but allow request
				log.Printf("Redis rate limit error for user %d: %v", userID, err)
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "rate_limit_exceeded",
					"message": "Too many requests. Please try again later.",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
