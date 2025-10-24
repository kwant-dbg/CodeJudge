package dbutil

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	MaxOpenConns    int           // Maximum number of open connections
	MaxIdleConns    int           // Maximum number of idle connections
	ConnMaxLifetime time.Duration // Maximum lifetime of a connection
	ConnMaxIdleTime time.Duration // Maximum idle time of a connection
	ConnTimeout     time.Duration // Connection timeout
}

// DefaultDatabaseConfig returns sensible defaults for production
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		MaxOpenConns:    25,               // Don't overwhelm the database
		MaxIdleConns:    5,                // Keep some connections ready
		ConnMaxLifetime: 30 * time.Minute, // Rotate connections regularly
		ConnMaxIdleTime: 5 * time.Minute,  // Close idle connections
		ConnTimeout:     10 * time.Second, // Reasonable connection timeout
	}
}

// ConnectionManager manages database connections with proper pooling
type ConnectionManager struct {
	db     *sql.DB
	config DatabaseConfig
	logger *zap.Logger
}

// NewConnectionManager creates a new database connection manager
func NewConnectionManager(logger *zap.Logger, databaseURL string, config DatabaseConfig) (*ConnectionManager, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	manager := &ConnectionManager{
		db:     db,
		config: config,
		logger: logger,
	}

	logger.Info("Database connection manager initialized",
		zap.Int("max_open_conns", config.MaxOpenConns),
		zap.Int("max_idle_conns", config.MaxIdleConns),
		zap.Duration("conn_max_lifetime", config.ConnMaxLifetime),
	)

	return manager, nil
}

// ConnectManagerWithRetry creates a connection manager with retry logic
func ConnectManagerWithRetry(logger *zap.Logger, databaseURL string, attempts int, delay time.Duration) *ConnectionManager {
	config := DefaultDatabaseConfig()

	var manager *ConnectionManager
	var err error

	for i := 0; i < attempts; i++ {
		manager, err = NewConnectionManager(logger, databaseURL, config)
		if err == nil {
			return manager
		}

		logger.Warn("Failed to connect to database, retrying...",
			zap.Int("attempt", i+1),
			zap.Int("max_attempts", attempts),
			zap.Error(err))

		if i < attempts-1 {
			time.Sleep(delay)
		}
	}

	logger.Fatal("Failed to connect to database after all attempts", zap.Error(err))
	return nil
}

// GetDB returns the underlying database connection
func (cm *ConnectionManager) GetDB() *sql.DB {
	return cm.db
}

// Close gracefully closes the connection manager
func (cm *ConnectionManager) Close() error {
	if err := cm.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	cm.logger.Info("Database connection manager closed")
	return nil
}

// Stats returns database connection statistics
func (cm *ConnectionManager) Stats() sql.DBStats {
	return cm.db.Stats()
}
