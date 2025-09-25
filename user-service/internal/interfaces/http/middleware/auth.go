package middleware

import (
	"context"
	"net/http"
	"strings"
	"user-service/internal/infrastructure/auth"
)

type contextKey string

const userIDKey = contextKey("userID")

// AuthMiddleware nhận vào jwtManager để validate token
func AuthMiddleware(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
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

			// ✅ Gọi method ValidateToken trên jwtManager
			claims, err := jwtManager.ValidateToken(tokenStr)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Inject user_id vào context → handler có thể lấy ra
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID : helper để lấy userID từ context trong handler
func GetUserID(r *http.Request) uint {
	if v := r.Context().Value(userIDKey); v != nil {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}
