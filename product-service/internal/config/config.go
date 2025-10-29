package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	JWTSecret string // For validating tokens from User Service
	
	// Database config
	DBHost            string
	DBPort            int
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	DBMaxIdleConns    int
	DBMaxOpenConns    int
	DBConnMaxLifeTime time.Duration
	DBConnMaxIdleTime time.Duration
	DBRetryAttempts   int
	DBRetryDelay      time.Duration
	
	// Redis config
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	
	// Cache
	CacheProductTTL  time.Duration
	CacheCategoryTTL time.Duration
	
	// Rate limiting config
	RateLimitGlobal      float64
	RateLimitGlobalBurst int
	
	// External services
	UserServiceURL string
}

func Load() *Config {
	_ = godotenv.Load()
	
	port := getEnv("PORT", "8082")
	jwtSecret := getEnv("JWT_SECRET", "your-super-secret-key-change-in-production")
	
	// Database configuration
	dbHost := getEnv("DB_HOST", "postgres")
	dbPort := getEnvAsInt("DB_PORT", 5432)
	dbUser := getEnv("DB_USER", "admin")
	dbPassword := getEnv("DB_PASSWORD", "admin")
	dbName := getEnv("DB_NAME", "ecommerce")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")
	
	dbMaxIdleConns := getEnvAsInt("DB_MAX_IDLE_CONNS", 10)
	dbMaxOpenConns := getEnvAsInt("DB_MAX_OPEN_CONNS", 100)
	
	dbConnMaxLifeTimeStr := getEnv("DB_CONN_MAX_LIFETIME", "5m")
	dbConnMaxLifeTime, _ := time.ParseDuration(dbConnMaxLifeTimeStr)
	
	dbConnMaxIdleTimeStr := getEnv("DB_CONN_MAX_IDLETIME", "1m")
	dbConnMaxIdleTime, _ := time.ParseDuration(dbConnMaxIdleTimeStr)
	
	dbRetryAttempts := getEnvAsInt("DB_RETRY_ATTEMPTS", 5)
	dbRetryDelayStr := getEnv("DB_RETRY_DELAY", "2s")
	dbRetryDelay, _ := time.ParseDuration(dbRetryDelayStr)
	
	// Redis config
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvAsInt("REDIS_DB", 0)
	
	// Cache config
	cacheProductTTLStr := getEnv("CACHE_PRODUCT_TTL", "1h")
	cacheProductTTL, _ := time.ParseDuration(cacheProductTTLStr)
	
	cacheCategoryTTLStr := getEnv("CACHE_CATEGORY_TTL", "24h")
	cacheCategoryTTL, _ := time.ParseDuration(cacheCategoryTTLStr)
	
	// Rate limiting
	rateLimitGlobal := getEnvAsFloat("RATE_LIMIT_GLOBAL", 100.0)
	rateLimitGlobalBurst := getEnvAsInt("RATE_LIMIT_GLOBAL_BURST", 200)
	
	// External services
	userServiceURL := getEnv("USER_SERVICE_URL", "http://user-service:8081")
	
	return &Config{
		Port:                 port,
		JWTSecret:            jwtSecret,
		DBHost:               dbHost,
		DBPort:               dbPort,
		DBUser:               dbUser,
		DBPassword:           dbPassword,
		DBName:               dbName,
		DBSSLMode:            dbSSLMode,
		DBMaxIdleConns:       dbMaxIdleConns,
		DBMaxOpenConns:       dbMaxOpenConns,
		DBConnMaxLifeTime:    dbConnMaxLifeTime,
		DBConnMaxIdleTime:    dbConnMaxIdleTime,
		DBRetryAttempts:      dbRetryAttempts,
		DBRetryDelay:         dbRetryDelay,
		RedisAddr:            redisAddr,
		RedisPassword:        redisPassword,
		RedisDB:              redisDB,
		CacheProductTTL:      cacheProductTTL,
		CacheCategoryTTL:     cacheCategoryTTL,
		RateLimitGlobal:      rateLimitGlobal,
		RateLimitGlobalBurst: rateLimitGlobalBurst,
		UserServiceURL:       userServiceURL,
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}

func getEnvAsFloat(key string, fallback float64) float64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return value
	}
	return fallback
}