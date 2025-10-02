package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

type DBConfig struct {
	Host            string
	Port            int
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
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	var db *gorm.DB
	var err error
	//Retry logic for connection
	for i := 0; i < cfg.RetryAttempts; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
			NowFunc: func() time.Time {
				return time.Now().UTC()
			},
			PrepareStmt: true,
		})
		if err == nil {
			break
		}
		log.Printf("Failed to connect to database (attempt %d/%d): %v",
			i+1, cfg.RetryAttempts, err)

		if i < cfg.RetryAttempts-1 {
			time.Sleep(cfg.RetryDelay)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect after %d attempts: %w",
			cfg.RetryAttempts, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifeTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Database connection established successfully")

	return db, nil
}

// SetupReadWriteSplit - for read/write spiltting
func SetupReadWriteSplit(db *gorm.DB, readDSN []string) error {
	resolveConfig := dbresolver.Config{
		Sources:  []gorm.Dialector{}, // Write database (uses main connection)
		Replicas: []gorm.Dialector{}, // read databases
		Policy:   dbresolver.RandomPolicy{},
	}

	for _, dsn := range readDSN {
		resolveConfig.Replicas = append(
			resolveConfig.Replicas,
			postgres.Open(dsn),
		)
	}

	err := db.Use(
		dbresolver.Register(resolveConfig).
			SetConnMaxIdleTime(time.Hour).
			SetConnMaxLifetime((24 * time.Hour)).
			SetMaxIdleConns(10).
			SetMaxOpenConns(100),
	)

	return err
}
