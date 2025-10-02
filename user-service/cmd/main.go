package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"user-service/internal/application"
	"user-service/internal/config"
	"user-service/internal/domain"
	"user-service/internal/infrastructure/auth"
	"user-service/internal/infrastructure/postgres"
	userhttp "user-service/internal/interfaces/http"
	"user-service/internal/interfaces/http/middleware"

	_ "github.com/lib/pq"
)

func main() {
	// Load config
	cfg := config.Load()

	// Setup database connection with advanced config
	dbConfig := &postgres.DBConfig{
		Host:            "postgres", // from docker-compose
		Port:            5432,
		User:            "admin",
		Password:        "admin",
		DBName:          "ecommerce",
		SSLMode:         "disable",
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifeTime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		RetryAttempts:   5,
		RetryDelay:      2 * time.Second,
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

	// Auto migrate
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Initialize repositories and services
	userRepo := postgres.NewUserRepository(db)
	txManager := postgres.NewTransactionManager(db)
	userService := application.NewUserService(userRepo, txManager)

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpire)

	// Initialize handlers
	userHandler := userhttp.NewUserHandler(userService, jwtManager)

	// Setup routes
	mux := setupRoutes(userHandler, jwtManager)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":8081",
		Handler:      mux,
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

func setupRoutes(handler *userhttp.UserHandler, jwtManager *auth.JWTManager) *http.ServeMux {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/health", healthCheck)
	mux.HandleFunc("/users/register", handler.Register)
	mux.HandleFunc("/users/login", handler.Login)

	// Protected routes
	mux.Handle("/users/me", middleware.AuthMiddleware(jwtManager)(
		http.HandlerFunc(handler.GetCurrentUser),
	))

	mux.Handle("/users/update", middleware.AuthMiddleware(jwtManager)(
		http.HandlerFunc(handler.UpdateUser),
	))

	mux.Handle("/users", middleware.AuthMiddleware(jwtManager)(
		http.HandlerFunc(handler.ListUsers),
	))

	mux.Handle("/users/delete", middleware.AuthMiddleware(jwtManager)(
		http.HandlerFunc(handler.DeleteUser),
	))

	return mux
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}
