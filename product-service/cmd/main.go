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

	"product-service/internal/application"
	"product-service/internal/config"
	"product-service/internal/infrastructure/postgres"
	handlers "product-service/internal/interfaces/http/handlers"
	"product-service/internal/interfaces/http/middleware"

	_ "github.com/lib/pq"
	"gorm.io/gorm"
)

func main() {
	// Load config
	cfg := config.Load()

	// Setup database connection
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

	// Get underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get sql.DB:", err)
	}
	defer sqlDB.Close()

	// Auto migrate
	if err := db.AutoMigrate(
		&postgres.CategoryModel{},
		&postgres.TagModel{},
		&postgres.ProductModel{},
		&postgres.ProductImageModel{},
		&postgres.ProductVariantModel{},
	); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	log.Print("Database migrated successfully")

	// Initialize repositories
	productRepo := postgres.NewProductRepository(db)
	categoryRepo := postgres.NewCategoryRepository(db)
	txManager := postgres.NewTransactionManager(db)

	// Initialize services (without cache for now)
	productService := application.NewProductService(productRepo, txManager, nil)
	categoryService := application.NewCategoryService(categoryRepo, txManager, nil)

	// Initialize handlers
	productHandler := handlers.NewProductHandler(productService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)

	// Setup routes
	mux := setupRoutes(productHandler, categoryHandler, cfg, db)

	// Apply middleware chain
	var handler http.Handler = mux
	handler = middleware.CORS(handler)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":8082",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Product Service starting on port %s", srv.Addr)
		log.Printf("Environment: %s", getEnv("ENVIRONMENT", "development"))
		log.Printf("Features enabled:")
		log.Printf("  - Database: PostgreSQL")
		log.Printf("  - Cache: disabled")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

func setupRoutes(
	productHandler *handlers.ProductHandler,
	categoryHandler *handlers.CategoryHandler,
	cfg *config.Config,
	db *gorm.DB,
) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", healthCheck(db))

	// Public routes - Products (no auth required for reading)
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// List products - public
			productHandler.ListProducts(w, r)
		case http.MethodPost:
			// Create product - requires auth
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(productHandler.CreateProduct),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Product by ID
	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Get product - public
			productHandler.GetProduct(w, r)
		case http.MethodPut:
			// Update product - requires auth
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(productHandler.UpdateProduct),
			).ServeHTTP(w, r)
		case http.MethodDelete:
			// Delete product - requires auth
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(productHandler.DeleteProduct),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Categories routes
	mux.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// List categories - public
			categoryHandler.ListCategories(w, r)
		case http.MethodPost:
			// Create category - requires auth
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(categoryHandler.CreateCategory),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Category by ID
	mux.HandleFunc("/categories/", func(w http.ResponseWriter, r *http.Request) {
		// Check if it's /categories/root
		if r.URL.Path == "/categories/root" {
			categoryHandler.GetRootCategories(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			// Get category - public
			categoryHandler.GetCategory(w, r)
		case http.MethodPut:
			// Update category - requires auth
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(categoryHandler.UpdateCategory),
			).ServeHTTP(w, r)
		case http.MethodDelete:
			// Delete category - requires auth
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(categoryHandler.DeleteCategory),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}

func healthCheck(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := map[string]interface{}{
			"status":    "healthy",
			"service":   "product-service",
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