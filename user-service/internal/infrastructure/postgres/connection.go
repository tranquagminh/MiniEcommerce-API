package postgres

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifeTime time.Duration
	ConnMaxIdleTime time.Duration
	RetryAttempts   int
	RetryDelay      time.Duration
}

func NewConnection(cfg *DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("")
}
