package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"user-service/internal/application"
	"user-service/internal/config"
	"user-service/internal/domain"
	"user-service/internal/infrastructure/auth"
	"user-service/internal/infrastructure/postgres"
	userhttp "user-service/internal/interfaces/http"
	"user-service/internal/interfaces/http/middleware"

	_ "github.com/lib/pq"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {

	cfg := config.Load()
	var db *gorm.DB
	var err error

	for i := 0; i < 5; i++ {
		db, err = gorm.Open(gormPostgres.Open(cfg.DBUrl), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Retrying DB connection (%d/5): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatal("Could not connect to DB: ", err)
	}

	sqlDB, err := db.DB()
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	if err := db.AutoMigrate(&domain.User{}); err != nil {
		log.Fatal("Could not migrate DB: ", err)
	}

	repo := postgres.NewUserRepository(db)
	service := application.NewUserService(repo)

	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpire)

	handler := userhttp.NewUserHandler(service, jwtManager)

	mux := http.NewServeMux()

	mux.HandleFunc("/users/register", handler.Register)
	mux.HandleFunc("/users/login", handler.Login)

	mux.Handle("/users/me", middleware.AuthMiddleware(jwtManager)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r)
		w.Write([]byte(fmt.Sprintf("Hello user %d", userID)))
	})))

	http.ListenAndServe(":8081", mux)
}
