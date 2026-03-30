package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
		// New tables for orders, reviews, Q&A, coupons
		&postgres.OrderModel{},
		&postgres.OrderItemModel{},
		&postgres.ReviewModel{},
		&postgres.ProductQAModel{},
		&postgres.CouponModel{},
		&postgres.CouponUsageModel{},
	); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	log.Print("Database migrated successfully")

	// Initialize repositories
	productRepo := postgres.NewProductRepository(db)
	categoryRepo := postgres.NewCategoryRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	reviewRepo := postgres.NewReviewRepository(db)
	qaRepo := postgres.NewQARepository(db)
	txManager := postgres.NewTransactionManager(db)

	// Initialize services (without cache for now)
	productService := application.NewProductService(productRepo, txManager, nil)
	categoryService := application.NewCategoryService(categoryRepo, txManager, nil)

	// Initialize handlers
	productHandler := handlers.NewProductHandler(productService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	orderHandler := handlers.NewOrderHandler(orderRepo)
	reviewHandler := handlers.NewReviewHandler(reviewRepo)
	qaHandler := handlers.NewQAHandler(qaRepo)

	// Setup routes
	mux := setupRoutes(productHandler, categoryHandler, orderHandler, reviewHandler, qaHandler, cfg, db)

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
		log.Printf("  - Orders API: enabled")
		log.Printf("  - Reviews API: enabled")
		log.Printf("  - Q&A API: enabled")

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
	orderHandler *handlers.OrderHandler,
	reviewHandler *handlers.ReviewHandler,
	qaHandler *handlers.QAHandler,
	cfg *config.Config,
	db *gorm.DB,
) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", healthCheck(db))

	// ==================== PRODUCT ROUTES ====================
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			productHandler.ListProducts(w, r)
		case http.MethodPost:
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(productHandler.CreateProduct),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Handle /products/{id}/reviews
		if strings.HasSuffix(path, "/reviews") {
			reviewHandler.GetProductReviews(w, r)
			return
		}

		// Handle /products/{id}/qa
		if strings.HasSuffix(path, "/qa") {
			qaHandler.GetProductQA(w, r)
			return
		}

		// Handle /products/{id}/images/{imageId}  (DELETE)
		if strings.Contains(path, "/images/") {
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(productHandler.DeleteProductImage),
			).ServeHTTP(w, r)
			return
		}

		// Handle /products/{id}/images  (POST)
		if strings.HasSuffix(path, "/images") {
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(productHandler.AddProductImage),
			).ServeHTTP(w, r)
			return
		}

		// Handle /products/{id}
		switch r.Method {
		case http.MethodGet:
			productHandler.GetProduct(w, r)
		case http.MethodPut:
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(productHandler.UpdateProduct),
			).ServeHTTP(w, r)
		case http.MethodDelete:
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(productHandler.DeleteProduct),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ==================== CATEGORY ROUTES ====================
	mux.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			categoryHandler.ListCategories(w, r)
		case http.MethodPost:
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(categoryHandler.CreateCategory),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/categories/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/categories/root" {
			categoryHandler.GetRootCategories(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			categoryHandler.GetCategory(w, r)
		case http.MethodPut:
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(categoryHandler.UpdateCategory),
			).ServeHTTP(w, r)
		case http.MethodDelete:
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(categoryHandler.DeleteCategory),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ==================== ORDER ROUTES ====================
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Admin: list all orders
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(orderHandler.ListOrders),
			).ServeHTTP(w, r)
		case http.MethodPost:
			// Customer: create order
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(orderHandler.CreateOrder),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/orders/stats", func(w http.ResponseWriter, r *http.Request) {
		middleware.AuthMiddleware(cfg.JWTSecret)(
			http.HandlerFunc(orderHandler.GetOrderStats),
		).ServeHTTP(w, r)
	})

	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Handle /orders/{id}/status
		if strings.HasSuffix(path, "/status") {
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(orderHandler.UpdateOrderStatus),
			).ServeHTTP(w, r)
			return
		}

		// Handle /orders/{id}
		switch r.Method {
		case http.MethodGet:
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(orderHandler.GetOrder),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// User orders
	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/orders") {
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(orderHandler.GetUserOrders),
			).ServeHTTP(w, r)
			return
		}
		http.Error(w, "Not found", http.StatusNotFound)
	})

	// ==================== REVIEW ROUTES ====================
	mux.HandleFunc("/reviews", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Admin: list all reviews
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(reviewHandler.ListReviews),
			).ServeHTTP(w, r)
		case http.MethodPost:
			// Customer: create review
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(reviewHandler.CreateReview),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/reviews/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Handle /reviews/{id}/status
		if strings.HasSuffix(path, "/status") {
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(reviewHandler.UpdateReviewStatus),
			).ServeHTTP(w, r)
			return
		}

		// Handle /reviews/{id}/reply
		if strings.HasSuffix(path, "/reply") {
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(reviewHandler.AddAdminReply),
			).ServeHTTP(w, r)
			return
		}

		// Handle /reviews/{id}
		switch r.Method {
		case http.MethodGet:
			reviewHandler.GetReview(w, r)
		case http.MethodDelete:
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(reviewHandler.DeleteReview),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ==================== Q&A ROUTES ====================
	mux.HandleFunc("/qa", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Admin: list all Q&A
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(qaHandler.ListQA),
			).ServeHTTP(w, r)
		case http.MethodPost:
			// Customer: create question
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(qaHandler.CreateQuestion),
			).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/qa/pending-count", func(w http.ResponseWriter, r *http.Request) {
		middleware.AuthMiddleware(cfg.JWTSecret)(
			http.HandlerFunc(qaHandler.GetPendingCount),
		).ServeHTTP(w, r)
	})

	mux.HandleFunc("/qa/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Handle /qa/{id}/answer
		if strings.HasSuffix(path, "/answer") {
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(qaHandler.AnswerQuestion),
			).ServeHTTP(w, r)
			return
		}

		// Handle /qa/{id}
		switch r.Method {
		case http.MethodGet:
			qaHandler.GetQA(w, r)
		case http.MethodDelete:
			middleware.AuthMiddleware(cfg.JWTSecret)(
				http.HandlerFunc(qaHandler.DeleteQA),
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
