package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      string
	JWTSecret string
	JWTExpire time.Duration

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

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Cache
	CacheUserTTL time.Duration

	// Rate limiting config
	RateLimitGlobal        float64
	RateLimitGlobalBurst   int
	RateLimitLogin         float64
	RateLimitLoginBurst    int
	RateLimitRegister      float64
	RateLimitRegisterBurst int
}

func Load() *Config {
	_ = godotenv.Load()

	port := getEnv("PORT", "8081")
	jwtSecret := getEnv("JWT_SECRET", "your-super-secret-key-change-in-production")
	jwtExpireStr := getEnv("JWT_EXPIRE", "24h")

	jwtExpire, err := time.ParseDuration(jwtExpireStr)
	if err != nil {
		log.Fatalf("Invalid JWT_EXPIRE: %v", err)
	}

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
	cacheUserTTLStr := getEnv("CACHE_USER_TTL", "5m")
	cacheUserTTL, _ := time.ParseDuration(cacheUserTTLStr)

	// Rate limiting configuration
	rateLimitGlobal := getEnvAsFloat("RATE_LIMIT_GLOBAL", 100.0)
	rateLimitGlobalBurst := getEnvAsInt("RATE_LIMIT_GLOBAL_BURST", 200)
	rateLimitLogin := getEnvAsFloat("RATE_LIMIT_LOGIN", 0.167) // 10/min
	rateLimitLoginBurst := getEnvAsInt("RATE_LIMIT_LOGIN_BURST", 2)
	rateLimitRegister := getEnvAsFloat("RATE_LIMIT_REGISTER", 0.083) // 5/min
	rateLimitRegisterBurst := getEnvAsInt("RATE_LIMIT_REGISTER_BURST", 1)

	return &Config{
		Port:                   port,
		JWTSecret:              jwtSecret,
		JWTExpire:              jwtExpire,
		DBHost:                 dbHost,
		DBPort:                 dbPort,
		DBUser:                 dbUser,
		DBPassword:             dbPassword,
		DBName:                 dbName,
		DBSSLMode:              dbSSLMode,
		DBMaxIdleConns:         dbMaxIdleConns,
		DBMaxOpenConns:         dbMaxOpenConns,
		DBConnMaxLifeTime:      dbConnMaxLifeTime,
		DBConnMaxIdleTime:      dbConnMaxIdleTime,
		DBRetryAttempts:        dbRetryAttempts,
		DBRetryDelay:           dbRetryDelay,
		RedisAddr:              redisAddr,
		RedisPassword:          redisPassword,
		RedisDB:                redisDB,
		CacheUserTTL:           cacheUserTTL,
		RateLimitGlobal:        rateLimitGlobal,
		RateLimitGlobalBurst:   rateLimitGlobalBurst,
		RateLimitLogin:         rateLimitLogin,
		RateLimitLoginBurst:    rateLimitLoginBurst,
		RateLimitRegister:      rateLimitRegister,
		RateLimitRegisterBurst: rateLimitRegisterBurst,
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
