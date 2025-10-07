package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"user-service/internal/application"
	"user-service/internal/config"
	"user-service/internal/infrastructure/auth"
	"user-service/internal/infrastructure/postgres"
	"user-service/internal/infrastructure/redis"
	userhttp "user-service/internal/interfaces/http/handlers"
	"user-service/internal/interfaces/http/middleware"

	_ "github.com/lib/pq"
	"gorm.io/gorm"
)

func main() {
	// Load config
	cfg := config.Load()

	// Setup database connection with advanced config
	dbConfig := &postgres.DBConfig{
		Host:            cfg.DBHost,
		Port:            cfg.DBPort,
		User:            cfg.DBUser,
		Password:        cfg.DBPassword,
		DBName:          cfg.DBName,
		SSLMode:         cfg.DBSSLMode,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		ConnMaxLifeTime: cfg.DBConnMaxLifeTime,
		ConnMaxIdleTime: cfg.DBConnMaxIdleTime,
		RetryAttempts:   cfg.DBRetryAttempts,
		RetryDelay:      cfg.DBRetryDelay,
	}

	// Connect to database
	db, err := postgres.NewConnection(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Get underlying SQL database to ensure closure
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get sql.DB:", err)
	}
	defer sqlDB.Close()

	// Initialize Redis (optional - graceful degradation)
	var redisClient *redis.RedisClient
	redisClient, err = redis.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Printf("WARNING: Failed to connect to Redis: %v", err)
		log.Printf("Continuing without Redis - using in-memory cache and rate limiting")
		redisClient = nil
	} else {
		defer redisClient.Close()
		log.Println("Redis connected successfully")
	}

	// Auto migrate
	if err := db.AutoMigrate(&postgres.UserModel{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	log.Print("Database migrated successfully")

	// Initialize cache
	var userCache application.UserCache
	if redisClient != nil {
		userCache = redis.NewUserCache(redisClient, cfg.CacheUserTTL)
	}

	// Initialize repositories and services
	userRepo := postgres.NewUserRepository(db)
	txManager := postgres.NewTransactionManager(db)
	userService := application.NewUserService(userRepo, txManager, userCache)

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpire)

	// Initialize handlers
	userHandler := userhttp.NewUserHandler(userService, jwtManager)

	// Setup routes with proper configuration
	mux := setupRoutes(userHandler, jwtManager, db, redisClient, cfg)

	// Apply middleware chain
	var handler http.Handler = mux

	// Apply global rate limiting
	if redisClient != nil {
		// Use Redis-based rate limiting for distributed systems
		globalRateLimiter := middleware.NewRedisRateLimiter(
			redisClient,
			int(cfg.RateLimitGlobal),
			time.Minute,
		)
		handler = middleware.RedisRateLimitMiddleware(globalRateLimiter)(handler)
		log.Println("Using Redis-based rate limiting")
	} else {
		// Fallback to in-memory rate limiting
		globalRateLimiter := middleware.NewRateLimiter(
			cfg.RateLimitGlobal,
			cfg.RateLimitGlobalBurst,
			30*time.Minute,
		)
		handler = middleware.RateLimitMiddleware(globalRateLimiter)(handler)
		log.Println("Using in-memory rate limiting")
	}

	// Apply CORS
	handler = middleware.CORS(handler)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":8081",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", srv.Addr)
		log.Printf("Environment: %s", getEnv("ENVIRONMENT", "development"))
		log.Printf("Features enabled:")
		log.Printf("  - Database: PostgreSQL")
		log.Printf("  - Cache: %v", redisClient != nil)
		log.Printf("  - Rate Limiting: %v (Redis: %v)", true, redisClient != nil)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

func setupRoutes(
	handler *userhttp.UserHandler,
	jwtManager *auth.JWTManager,
	db *gorm.DB,
	redisClient *redis.RedisClient,
	cfg *config.Config,
) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check - includes Redis status
	mux.HandleFunc("/health", healthCheck(db, redisClient))

	// Public routes with specific rate limits
	if redisClient != nil {
		// Redis-based rate limiting
		// Register: 5 requests per minute
		mux.Handle("/users/register",
			middleware.CustomRedisRateLimitMiddleware(
				redisClient,
				5,
				time.Minute,
			)(http.HandlerFunc(handler.Register)),
		)

		// Login: 10 requests per minute
		mux.Handle("/users/login",
			middleware.CustomRedisRateLimitMiddleware(
				redisClient,
				10,
				time.Minute,
			)(http.HandlerFunc(handler.Login)),
		)
	} else {
		// In-memory rate limiting fallback
		mux.Handle("/users/register",
			middleware.CustomRateLimitMiddleware(0.083, 1)(
				http.HandlerFunc(handler.Register),
			),
		)

		mux.Handle("/users/login",
			middleware.CustomRateLimitMiddleware(0.167, 2)(
				http.HandlerFunc(handler.Login),
			),
		)
	}

	// Protected routes with authentication
	mux.Handle("/users/me",
		middleware.AuthMiddleware(jwtManager)(
			http.HandlerFunc(handler.GetCurrentUser),
		),
	)

	// Protected routes with auth + user-based rate limiting
	if redisClient != nil {
		// Redis-based user rate limiting
		mux.Handle("/users/update",
			middleware.AuthMiddleware(jwtManager)(
				middleware.RedisUserRateLimitMiddleware(redisClient, 10, time.Minute)(
					http.HandlerFunc(handler.UpdateUser),
				),
			),
		)

		mux.Handle("/users/delete",
			middleware.AuthMiddleware(jwtManager)(
				middleware.RedisUserRateLimitMiddleware(redisClient, 5, time.Minute)(
					http.HandlerFunc(handler.DeleteUser),
				),
			),
		)
	} else {
		// In-memory user rate limiting
		mux.Handle("/users/update",
			middleware.AuthMiddleware(jwtManager)(
				middleware.UserRateLimitMiddleware(2, 5)(
					http.HandlerFunc(handler.UpdateUser),
				),
			),
		)

		mux.Handle("/users/delete",
			middleware.AuthMiddleware(jwtManager)(
				middleware.UserRateLimitMiddleware(1, 2)(
					http.HandlerFunc(handler.DeleteUser),
				),
			),
		)
	}

	// List users - simple auth without extra rate limiting
	mux.Handle("/users",
		middleware.AuthMiddleware(jwtManager)(
			http.HandlerFunc(handler.ListUsers),
		),
	)

	return mux
}

func healthCheck(db *gorm.DB, redisClient *redis.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"services":  make(map[string]interface{}),
		}

		// Check database
		sqlDB, _ := db.DB()
		if err := sqlDB.Ping(); err != nil {
			health["status"] = "unhealthy"
			health["services"].(map[string]interface{})["database"] = map[string]interface{}{
				"status": "down",
				"error":  err.Error(),
			}
		} else {
			health["services"].(map[string]interface{})["database"] = map[string]interface{}{
				"status": "up",
			}
		}

		// Check Redis
		if redisClient != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()

			if err := redisClient.Ping(ctx); err != nil {
				health["services"].(map[string]interface{})["redis"] = map[string]interface{}{
					"status": "down",
					"error":  err.Error(),
				}
			} else {
				health["services"].(map[string]interface{})["redis"] = map[string]interface{}{
					"status": "up",
				}
			}
		} else {
			health["services"].(map[string]interface{})["redis"] = map[string]interface{}{
				"status": "not configured",
			}
		}

		// Determine overall status
		statusCode := http.StatusOK
		if health["status"] == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(health)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
