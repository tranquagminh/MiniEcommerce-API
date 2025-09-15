package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	DBUrl     string
	JWTSecret string
	JWTExpire time.Duration
}

func Load() *Config {
	_ = godotenv.Load()

	port := getEnv("PORT", "8081")
	dbUrl := getEnv("DB_URL", "postgres://admin:admin@postgres:5432/ecommerce?sslmode=disable")
	jwtSecret := getEnv("JWT_SECRET", "changeme")
	jwtExpireStr := getEnv("JWT_EXPIRE", "24h")

	jwtExpire, err := time.ParseDuration(jwtExpireStr)
	if err != nil {
		log.Fatalf("Invalid JWT_EXPIRE: %v", err)
	}

	return &Config{
		Port:      port,
		DBUrl:     dbUrl,
		JWTSecret: jwtSecret,
		JWTExpire: jwtExpire,
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
