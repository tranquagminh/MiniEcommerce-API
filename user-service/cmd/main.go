package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"user-service/internal/application"
	"user-service/internal/config"
	"user-service/internal/infrastructure/auth"
	"user-service/internal/infrastructure/postgres"
	userhttp "user-service/internal/interfaces/http"
	"user-service/internal/interfaces/http/middleware"

	_ "github.com/lib/pq"
)

func main() {

	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DBUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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
