package dbutil

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// TransactionOptions defines options for transaction handling
type TransactionOptions struct {
	IsolationLevel sql.IsolationLevel
	ReadOnly       bool
	Timeout        time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
}

// DefaultTransactionOptions returns sensible defaults for most use cases
func DefaultTransactionOptions() TransactionOptions {
	return TransactionOptions{
		IsolationLevel: sql.LevelReadCommitted, // Safe default
		ReadOnly:       false,
		Timeout:        30 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     100 * time.Millisecond,
	}
}

// StrictTransactionOptions returns options for sensitive operations
func StrictTransactionOptions() TransactionOptions {
	return TransactionOptions{
		IsolationLevel: sql.LevelSerializable, // Strictest isolation
		ReadOnly:       false,
		Timeout:        10 * time.Second,
		RetryAttempts:  5, // More retries for serialization failures
		RetryDelay:     50 * time.Millisecond,
	}
}

// TransactionFunc represents a function to execute within a transaction
type TransactionFunc func(tx *sql.Tx) error

// TransactionManager handles advanced transaction patterns
type TransactionManager struct {
	cm     *ConnectionManager
	logger *zap.Logger
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(cm *ConnectionManager, logger *zap.Logger) *TransactionManager {
	return &TransactionManager{
		cm:     cm,
		logger: logger,
	}
}

// ExecuteTransaction runs a function within a transaction with automatic retry logic
func (tm *TransactionManager) ExecuteTransaction(ctx context.Context, opts TransactionOptions, fn TransactionFunc) error {
	var lastErr error

	for attempt := 0; attempt <= opts.RetryAttempts; attempt++ {
		err := tm.executeTransactionOnce(ctx, opts, fn)
		if err == nil {
			if attempt > 0 {
				tm.logger.Info("Transaction succeeded after retry",
					zap.Int("attempt", attempt+1),
					zap.Duration("total_retry_time", time.Duration(attempt)*opts.RetryDelay))
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable (serialization failure, deadlock, etc.)
		if !isRetryableError(err) {
			tm.logger.Error("Non-retryable transaction error", zap.Error(err))
			return err
		}

		if attempt < opts.RetryAttempts {
			tm.logger.Warn("Transaction failed, retrying",
				zap.Int("attempt", attempt+1),
				zap.Int("max_attempts", opts.RetryAttempts+1),
				zap.Error(err))

			// Exponential backoff with jitter
			delay := opts.RetryDelay * time.Duration(1<<attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	tm.logger.Error("Transaction failed after all retry attempts",
		zap.Int("total_attempts", opts.RetryAttempts+1),
		zap.Error(lastErr))

	return lastErr
}

// executeTransactionOnce executes a transaction once
func (tm *TransactionManager) executeTransactionOnce(ctx context.Context, opts TransactionOptions, fn TransactionFunc) error {
	// Create context with timeout
	txCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Start transaction with proper isolation level
	txOpts := &sql.TxOptions{
		Isolation: opts.IsolationLevel,
		ReadOnly:  opts.ReadOnly,
	}

	tx, err := tm.cm.BeginTx(txCtx, txOpts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Ensure rollback on any error
	committed := false
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				tm.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			}
		}
	}()

	// Execute the transaction function
	if err := fn(tx); err != nil {
		return fmt.Errorf("transaction function failed: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	committed = true
	return nil
}

// ExecuteReadOnlyTransaction is optimized for read-only operations
func (tm *TransactionManager) ExecuteReadOnlyTransaction(ctx context.Context, fn TransactionFunc) error {
	opts := TransactionOptions{
		IsolationLevel: sql.LevelRepeatableRead, // Good for consistent reads
		ReadOnly:       true,
		Timeout:        15 * time.Second,
		RetryAttempts:  2, // Fewer retries for read-only
		RetryDelay:     50 * time.Millisecond,
	}

	return tm.ExecuteTransaction(ctx, opts, fn)
}

// ExecuteStrictTransaction is for operations requiring strict consistency
func (tm *TransactionManager) ExecuteStrictTransaction(ctx context.Context, fn TransactionFunc) error {
	return tm.ExecuteTransaction(ctx, StrictTransactionOptions(), fn)
}

// Distributed transaction coordinator for cross-service operations
type DistributedTransaction struct {
	id          string
	steps       []DistributedTransactionStep
	tm          *TransactionManager
	logger      *zap.Logger
	compensated bool
}

// DistributedTransactionStep represents a step in a distributed transaction
type DistributedTransactionStep struct {
	Name       string
	Execute    func(ctx context.Context) error
	Compensate func(ctx context.Context) error
	executed   bool
}

// NewDistributedTransaction creates a new distributed transaction
func NewDistributedTransaction(id string, tm *TransactionManager, logger *zap.Logger) *DistributedTransaction {
	return &DistributedTransaction{
		id:     id,
		tm:     tm,
		logger: logger,
		steps:  make([]DistributedTransactionStep, 0),
	}
}

// AddStep adds a step to the distributed transaction
func (dt *DistributedTransaction) AddStep(step DistributedTransactionStep) {
	dt.steps = append(dt.steps, step)
}

// Execute runs all steps of the distributed transaction
func (dt *DistributedTransaction) Execute(ctx context.Context) error {
	dt.logger.Info("Starting distributed transaction", zap.String("transaction_id", dt.id))

	for i := range dt.steps {
		step := &dt.steps[i]

		dt.logger.Debug("Executing transaction step",
			zap.String("transaction_id", dt.id),
			zap.String("step_name", step.Name),
			zap.Int("step_index", i))

		if err := step.Execute(ctx); err != nil {
			dt.logger.Error("Transaction step failed",
				zap.String("transaction_id", dt.id),
				zap.String("step_name", step.Name),
				zap.Error(err))

			// Compensate all executed steps in reverse order
			if compensateErr := dt.compensate(ctx, i); compensateErr != nil {
				dt.logger.Error("Compensation failed",
					zap.String("transaction_id", dt.id),
					zap.Error(compensateErr))
			}

			return fmt.Errorf("distributed transaction step '%s' failed: %w", step.Name, err)
		}

		step.executed = true
	}

	dt.logger.Info("Distributed transaction completed successfully", zap.String("transaction_id", dt.id))
	return nil
}

// compensate runs compensation logic for executed steps in reverse order
func (dt *DistributedTransaction) compensate(ctx context.Context, failedStepIndex int) error {
	if dt.compensated {
		return nil // Already compensated
	}

	dt.compensated = true
	dt.logger.Info("Starting transaction compensation", zap.String("transaction_id", dt.id))

	// Compensate in reverse order
	for i := failedStepIndex - 1; i >= 0; i-- {
		step := &dt.steps[i]
		if !step.executed {
			continue
		}

		dt.logger.Debug("Compensating transaction step",
			zap.String("transaction_id", dt.id),
			zap.String("step_name", step.Name))

		if step.Compensate != nil {
			if err := step.Compensate(ctx); err != nil {
				dt.logger.Error("Step compensation failed",
					zap.String("transaction_id", dt.id),
					zap.String("step_name", step.Name),
					zap.Error(err))
				// Continue compensating other steps even if one fails
			}
		}
	}

	return nil
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// PostgreSQL specific error codes that are retryable
	retryableErrors := []string{
		"40001", // serialization_failure
		"40P01", // deadlock_detected
		"53300", // too_many_connections
		"57014", // query_canceled
		"connection refused",
		"connection reset",
		"connection timed out",
	}

	for _, retryableErr := range retryableErrors {
		if contains(errStr, retryableErr) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(str, substr string) bool {
	return len(str) >= len(substr) &&
		(substr == "" ||
			str == substr ||
			(len(str) > len(substr) &&
				(str[:len(substr)] == substr ||
					str[len(str)-len(substr):] == substr ||
					indexContains(str, substr) >= 0)))
}

func indexContains(str, substr string) int {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
