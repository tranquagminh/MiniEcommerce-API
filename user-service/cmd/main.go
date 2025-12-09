package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"user-service/internal/application"
	"user-service/internal/config"
	"user-service/internal/infrastructure/auth"
	"user-service/internal/infrastructure/logger"
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

	// Initialize logger
	environment := getEnv("ENVIRONMENT", "development")
	log := logger.New("user-service", environment)

	// Initialize Prometheus metrics
	middleware.InitMetrics()
	log.Info().Msg("Prometheus metrics initialized")

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
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	// Get underlying SQL database to ensure closure
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get sql.DB")
	}
	defer sqlDB.Close()

	// Initialize Redis (optional - graceful degradation)
	var redisClient *redis.RedisClient
	redisClient, err = redis.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to connect to Redis")
		log.Info().Msg("Continuing without Redis - using in-memory cache and rate limiting")
		redisClient = nil
	} else {
		defer redisClient.Close()
		log.Info().Msg("Redis connected successfully")
	}

	// Run database migrations
	migrationsPath := getEnv("MIGRATIONS_PATH", "./migrations")
	if err := postgres.RunMigrations(sqlDB, migrationsPath); err != nil {
		log.Fatal().Err(err).Msg("Failed to run database migrations")
	}
	log.Info().Msg("Database migrations completed successfully")

	// Log current migration version
	version, dirty, err := postgres.GetMigrationVersion(sqlDB, migrationsPath)
	if err == nil {
		log.Info().Uint("version", version).Bool("dirty", dirty).Msg("Current migration version")
	}

	// Initialize cache
	var userCache application.UserCache
	if redisClient != nil {
		userCache = redis.NewUserCache(redisClient, cfg.CacheUserTTL)
	}

	// Initialize repositories and services
	userRepo := postgres.NewUserRepository(db)
	txManager := postgres.NewTransactionManager(db)
	passwordValidator := auth.DefaultPasswordValidator()
	userService := application.NewUserService(userRepo, txManager, userCache, passwordValidator)

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
		log.Info().Msg("Using Redis-based rate limiting")
	} else {
		// Fallback to in-memory rate limiting
		globalRateLimiter := middleware.NewRateLimiter(
			cfg.RateLimitGlobal,
			cfg.RateLimitGlobalBurst,
			30*time.Minute,
		)
		handler = middleware.RateLimitMiddleware(globalRateLimiter)(handler)
		log.Info().Msg("Using in-memory rate limiting")
	}

	// Apply Prometheus metrics middleware
	handler = middleware.PrometheusMiddleware(handler)

	// Apply logging middleware
	handler = middleware.LoggingMiddleware(log.GetZerolog())(handler)

	// Apply security headers
	handler = middleware.SecurityHeaders(handler)

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
		log.Info().
			Str("address", srv.Addr).
			Str("environment", environment).
			Bool("database_enabled", true).
			Bool("cache_enabled", redisClient != nil).
			Bool("redis_rate_limiting", redisClient != nil).
			Msg("Server starting")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited gracefully")
}

func setupRoutes(
	handler *userhttp.UserHandler,
	jwtManager *auth.JWTManager,
	db *gorm.DB,
	redisClient *redis.RedisClient,
	_ *config.Config,
) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check (non-versioned)
	mux.HandleFunc("/health", healthCheck(db, redisClient))

	// Prometheus metrics endpoint (non-versioned)
	mux.Handle("/metrics", middleware.MetricsHandler())

	// API v1 routes
	// Public routes - Register
	if redisClient != nil {
		mux.Handle("/api/v1/auth/register",
			middleware.CustomRedisRateLimitMiddleware(redisClient, 5, time.Minute)(
				http.HandlerFunc(handler.Register),
			),
		)

		mux.Handle("/api/v1/auth/login",
			middleware.CustomRedisRateLimitMiddleware(redisClient, 10, time.Minute)(
				http.HandlerFunc(handler.Login),
			),
		)
	} else {
		mux.Handle("/api/v1/auth/register",
			middleware.CustomRateLimitMiddleware(0.083, 1)(
				http.HandlerFunc(handler.Register),
			),
		)

		mux.Handle("/api/v1/auth/login",
			middleware.CustomRateLimitMiddleware(0.167, 2)(
				http.HandlerFunc(handler.Login),
			),
		)
	}

	// Protected routes - User profile
	mux.Handle("/api/v1/users/me",
		middleware.AuthMiddleware(jwtManager)(
			http.HandlerFunc(handler.GetCurrentUser),
		),
	)

	// Update Profile with rate limiting
	if redisClient != nil {
		mux.Handle("/api/v1/users/profile",
			middleware.AuthMiddleware(jwtManager)(
				middleware.RedisUserRateLimitMiddleware(redisClient, 10, time.Minute)(
					http.HandlerFunc(handler.UpdateProfile),
				),
			),
		)

		mux.Handle("/api/v1/users/change-password",
			middleware.AuthMiddleware(jwtManager)(
				middleware.RedisUserRateLimitMiddleware(redisClient, 5, time.Minute)(
					http.HandlerFunc(handler.ChangePassword),
				),
			),
		)

		mux.Handle("/api/v1/users",
			middleware.AuthMiddleware(jwtManager)(
				http.HandlerFunc(handler.ListUsers),
			),
		)

		mux.Handle("/api/v1/users/delete",
			middleware.AuthMiddleware(jwtManager)(
				middleware.RedisUserRateLimitMiddleware(redisClient, 5, time.Minute)(
					http.HandlerFunc(handler.DeleteUser),
				),
			),
		)
	} else {
		mux.Handle("/api/v1/users/profile",
			middleware.AuthMiddleware(jwtManager)(
				middleware.UserRateLimitMiddleware(2, 5)(
					http.HandlerFunc(handler.UpdateProfile),
				),
			),
		)

		mux.Handle("/api/v1/users/change-password",
			middleware.AuthMiddleware(jwtManager)(
				middleware.UserRateLimitMiddleware(1, 2)(
					http.HandlerFunc(handler.ChangePassword),
				),
			),
		)

		mux.Handle("/api/v1/users",
			middleware.AuthMiddleware(jwtManager)(
				http.HandlerFunc(handler.ListUsers),
			),
		)

		mux.Handle("/api/v1/users/delete",
			middleware.AuthMiddleware(jwtManager)(
				middleware.UserRateLimitMiddleware(1, 2)(
					http.HandlerFunc(handler.DeleteUser),
				),
			),
		)
	}

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
