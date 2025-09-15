package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"user-service/internal/application"
	"user-service/internal/infrastructure/postgres"
	userhttp "user-service/internal/interfaces/http"

	_ "github.com/lib/pq"
)

func main() {
	connStr := "postgres://admin:admin@postgres:5432/ecommerce?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	repo := postgres.NewUserRepository(db)
	service := application.NewUserService(repo)
	handler := userhttp.NewUserHandler(service)

	http.HandleFunc("/users/register", handler.Register)

	fmt.Println("User Service running at :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
