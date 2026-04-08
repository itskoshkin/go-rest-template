package postgres

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"go-rest-template/internal/models"
	"go-rest-template/internal/utils/text"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	AutoMigrate bool

	LogLevel string
}

func NewInstance(ctx context.Context, cfg Config) (*gorm.DB, error) {
	fmt.Print("Connecting to Postgres...")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Println()
		return nil, fmt.Errorf("GORM: failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		fmt.Println()
		return nil, fmt.Errorf("GORM: failed to get underlying sql.DB: %v", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	if err = sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		fmt.Println()
		return nil, fmt.Errorf("GORM: failed to ping database: %w", err)
	}

	if cfg.AutoMigrate {
		if err = db.AutoMigrate(
			&models.User{},
			&models.Item{},
		); err != nil {
			_ = sqlDB.Close()
			fmt.Println()
			return nil, fmt.Errorf("GORM: failed to auto-migrate: %v", err)
		}
	}

	fmt.Println(text.Green(" Done."))

	return db, nil
}
