package dbutil

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
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
	QueryTimeout    time.Duration // Query timeout
}

// DefaultDatabaseConfig returns sensible defaults for production
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		MaxOpenConns:    25,               // Don't overwhelm the database
		MaxIdleConns:    5,                // Keep some connections ready
		ConnMaxLifetime: 30 * time.Minute, // Rotate connections regularly
		ConnMaxIdleTime: 5 * time.Minute,  // Close idle connections
		ConnTimeout:     10 * time.Second, // Reasonable connection timeout
		QueryTimeout:    30 * time.Second, // Reasonable query timeout
	}
}

// ConnectionManager manages database connections with proper pooling
type ConnectionManager struct {
	db        *sql.DB
	config    DatabaseConfig
	stmtCache map[string]*sql.Stmt
	stmtMutex sync.RWMutex
	logger    *zap.Logger
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
		db:        db,
		config:    config,
		stmtCache: make(map[string]*sql.Stmt),
		logger:    logger,
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

// GetDB returns the underlying database connection (for compatibility)
func (cm *ConnectionManager) GetDB() *sql.DB {
	return cm.db
}

// PrepareStatement prepares and caches a SQL statement
func (cm *ConnectionManager) PrepareStatement(name, query string) error {
	cm.stmtMutex.Lock()
	defer cm.stmtMutex.Unlock()

	if _, exists := cm.stmtCache[name]; exists {
		return nil // Already prepared
	}

	stmt, err := cm.db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement %s: %w", name, err)
	}

	cm.stmtCache[name] = stmt
	cm.logger.Debug("Prepared statement cached", zap.String("name", name))
	return nil
}

// ExecPrepared executes a prepared statement
func (cm *ConnectionManager) ExecPrepared(ctx context.Context, name string, args ...interface{}) (sql.Result, error) {
	cm.stmtMutex.RLock()
	stmt, exists := cm.stmtCache[name]
	cm.stmtMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("prepared statement %s not found", name)
	}

	queryCtx, cancel := context.WithTimeout(ctx, cm.config.QueryTimeout)
	defer cancel()

	return stmt.ExecContext(queryCtx, args...)
}

// QueryPrepared executes a prepared query
func (cm *ConnectionManager) QueryPrepared(ctx context.Context, name string, args ...interface{}) (*sql.Rows, error) {
	cm.stmtMutex.RLock()
	stmt, exists := cm.stmtCache[name]
	cm.stmtMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("prepared statement %s not found", name)
	}

	queryCtx, cancel := context.WithTimeout(ctx, cm.config.QueryTimeout)
	defer cancel()

	return stmt.QueryContext(queryCtx, args...)
}

// QueryRowPrepared executes a prepared query that returns a single row
func (cm *ConnectionManager) QueryRowPrepared(ctx context.Context, name string, args ...interface{}) *sql.Row {
	cm.stmtMutex.RLock()
	stmt, exists := cm.stmtCache[name]
	cm.stmtMutex.RUnlock()

	if !exists {
		// Return a row with an error - this follows sql.DB.QueryRow() pattern
		return cm.db.QueryRowContext(ctx, "SELECT 1 WHERE FALSE") // Always returns sql.ErrNoRows
	}

	queryCtx, cancel := context.WithTimeout(ctx, cm.config.QueryTimeout)
	defer cancel()

	return stmt.QueryRowContext(queryCtx, args...)
}

// BeginTx starts a transaction with proper isolation level
func (cm *ConnectionManager) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	txCtx, cancel := context.WithTimeout(ctx, cm.config.QueryTimeout)
	defer cancel()

	return cm.db.BeginTx(txCtx, opts)
}

// Close gracefully closes the connection manager
func (cm *ConnectionManager) Close() error {
	cm.stmtMutex.Lock()
	defer cm.stmtMutex.Unlock()

	// Close all prepared statements
	for name, stmt := range cm.stmtCache {
		if err := stmt.Close(); err != nil {
			cm.logger.Warn("Failed to close prepared statement",
				zap.String("name", name),
				zap.Error(err))
		}
	}

	// Close database connection
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
