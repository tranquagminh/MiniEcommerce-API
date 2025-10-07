package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"user-service/internal/infrastructure/redis"
)

type RedisRateLimiter struct {
	client *redis.RedisClient
	limit  int
	window time.Duration
}

func NewRedisRateLimiter(client *redis.RedisClient, limit int, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		limit:  limit,
		window: window,
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
			ip := getClientIP(r)

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
