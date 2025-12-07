package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/monitoring-engine/monitoring-tool/internal/config"
	"github.com/monitoring-engine/monitoring-tool/internal/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var postgresInstance *gorm.DB

// GetPostgresInstance initializes and returns the PostgreSQL connection
func GetPostgresInstance(cfg config.PostgresConfig) (*gorm.DB, error) {
	if postgresInstance != nil {
		return postgresInstance, nil
	}

	dsn := cfg.GetDSN()

	// Log connection attempt (without password)
	logger.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("user", cfg.User).
		Str("database", cfg.Database).
		Str("sslmode", cfg.SSLMode).
		Msg("Connecting to PostgreSQL database...")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		logger.Error().
			Err(err).
			Str("host", cfg.Host).
			Int("port", cfg.Port).
			Str("database", cfg.Database).
			Msg("Failed to connect to PostgreSQL")
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(cfg.MaxConnections())
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConnections())
	sqlDB.SetConnMaxLifetime(cfg.ConnectionLifetime())

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		logger.Error().
			Err(err).
			Str("host", cfg.Host).
			Int("port", cfg.Port).
			Msg("Failed to ping PostgreSQL")
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	// Log successful connection
	logger.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Int("max_connections", cfg.MaxConnections()).
		Int("max_idle_connections", cfg.MaxIdleConnections()).
		Dur("connection_lifetime", cfg.ConnectionLifetime()).
		Msg("Successfully connected to PostgreSQL")

	postgresInstance = db
	return postgresInstance, nil
}

// HealthCheck checks if the database connection is healthy
func HealthCheck(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database instance is nil")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// Close closes the database connection
func Close(db *gorm.DB) {
	if db == nil {
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		return
	}

	sqlDB.Close()
}
