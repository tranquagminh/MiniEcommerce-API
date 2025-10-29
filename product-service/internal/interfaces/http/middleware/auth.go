package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey = contextKey("userID")

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT tokens
// Uses the same JWT_SECRET as user-service for token validation
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}

			tokenStr := parts[1]

			// Parse and validate token
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Check if token is expired
			if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
				http.Error(w, "token expired", http.StatusUnauthorized)
				return
			}

			// Inject user_id into context
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID retrieves userID from context
func GetUserID(r *http.Request) uint {
	if v := r.Context().Value(userIDKey); v != nil {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// OptionalAuthMiddleware validates JWT if present, but doesn't require it
func OptionalAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No auth header, continue without user ID
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				// Invalid format, continue without user ID
				next.ServeHTTP(w, r)
				return
			}

			tokenStr := parts[1]

			// Parse and validate token
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				// Invalid token, continue without user ID
				next.ServeHTTP(w, r)
				return
			}

			// Valid token, inject user_id into context
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
