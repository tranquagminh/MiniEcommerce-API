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

	redisClient, err := redis.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Printf("Failed to connect to Redis: %v. Continuing without Redis...", err)
		// Continue without Redis - graceful degradation
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
	var userCache *redis.UserCache
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

	// Initialize global rate limiter: 100 requests per second, burst of 200
	globalRateLimiter := middleware.NewRateLimiter(100, 200, 30*time.Minute)

	// Setup routes
	mux := setupRoutes(userHandler, jwtManager, db)

	// Apply rate limiting TRƯỚC CORS
	handler := middleware.RateLimitMiddleware(globalRateLimiter)(
		middleware.CORS(mux),
	)

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

func setupRoutes(handler *userhttp.UserHandler, jwtManager *auth.JWTManager, db *gorm.DB) *http.ServeMux {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", healthCheck(db))

	// Register với rate limit chặt: 5 requests/minute (0.083/s)
	mux.Handle("/users/register",
		middleware.CustomRateLimitMiddleware(0.083, 1)(
			http.HandlerFunc(handler.Register),
		),
	)

	// Login với rate limit vừa: 10 requests/minute (0.167/s)
	mux.Handle("/users/login",
		middleware.CustomRateLimitMiddleware(0.167, 2)(
			http.HandlerFunc(handler.Login),
		),
	)

	// Protected routes
	mux.Handle("/users/me", middleware.AuthMiddleware(jwtManager)(
		http.HandlerFunc(handler.GetCurrentUser),
	))

	mux.Handle("/users/update",
		middleware.AuthMiddleware(jwtManager)(
			middleware.UserRateLimitMiddleware(2, 5)( // 2 req/s per user
				http.HandlerFunc(handler.UpdateUser),
			),
		),
	)

	mux.Handle("/users", middleware.AuthMiddleware(jwtManager)(
		http.HandlerFunc(handler.ListUsers),
	))

	mux.Handle("/users/delete", middleware.AuthMiddleware(jwtManager)(
		http.HandlerFunc(handler.DeleteUser),
	))

	return mux
}

func healthCheck(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sqlDB, _ := db.DB()
		if err := sqlDB.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "unhealthy",
				"error":  "database connection failed",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}
}
