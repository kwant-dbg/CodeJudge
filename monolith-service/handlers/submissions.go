package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"codejudge/common/dbutil"
	"codejudge/common/env"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type Submission struct {
	ID         int    `json:"id"`
	ProblemID  int    `json:"problem_id"`
	SourceCode string `json:"source_code"`
}

type SubmissionsHandler struct {
	logger    *zap.Logger
	dbManager *dbutil.ConnectionManager
	txManager *dbutil.TransactionManager
	rdb       *redis.Client
	ctx       context.Context
}

func NewSubmissionsHandler(logger *zap.Logger, dbManager *dbutil.ConnectionManager, rdb *redis.Client) *SubmissionsHandler {
	txManager := dbutil.NewTransactionManager(dbManager, logger)
	return &SubmissionsHandler{
		logger:    logger,
		dbManager: dbManager,
		txManager: txManager,
		rdb:       rdb,
		ctx:       context.Background(),
	}
}

func (h *SubmissionsHandler) CreateTables() {
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS submissions (
        id SERIAL PRIMARY KEY,
        problem_id INTEGER NOT NULL,
        source_code TEXT NOT NULL,
        verdict VARCHAR(50) DEFAULT 'Pending',
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );`
	if _, err := h.dbManager.GetDB().Exec(createTableSQL); err != nil {
		h.logger.Fatal("Failed to create 'submissions' table", zap.Error(err))
	}
	h.logger.Info("'submissions' table is ready")
}

// SubmissionError represents an error during submission processing
type SubmissionError struct {
	Message string
	Code    int
	Err     error
}

func (se *SubmissionError) Error() string {
	if se.Err != nil {
		return fmt.Sprintf("%s: %v", se.Message, se.Err)
	}
	return se.Message
}

// Sophisticated submission creation using distributed transaction pattern
func (h *SubmissionsHandler) createSubmissionTransactional(s *Submission) error {
	submissionID := fmt.Sprintf("sub_%d_%d", s.ProblemID, time.Now().UnixNano())

	// Create distributed transaction with proper compensation
	dt := dbutil.NewDistributedTransaction(submissionID, h.txManager, h.logger)

	var filePath string

	// Step 1: Database insertion with proper isolation
	dt.AddStep(dbutil.DistributedTransactionStep{
		Name: "database_insert",
		Execute: func(ctx context.Context) error {
			return h.txManager.ExecuteStrictTransaction(ctx, func(tx *sql.Tx) error {
				query := "INSERT INTO submissions (problem_id, source_code) VALUES ($1, $2) RETURNING id"
				return tx.QueryRowContext(ctx, query, s.ProblemID, s.SourceCode).Scan(&s.ID)
			})
		},
		Compensate: func(ctx context.Context) error {
			// Remove from database if other steps fail
			return h.txManager.ExecuteStrictTransaction(ctx, func(tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, "DELETE FROM submissions WHERE id = $1", s.ID)
				return err
			})
		},
	})

	// Step 2: File system storage
	dt.AddStep(dbutil.DistributedTransactionStep{
		Name: "file_storage",
		Execute: func(ctx context.Context) error {
			submissionDir := env.Get("SUBMISSION_STORAGE_PATH", "/app/submissions")
			if err := os.MkdirAll(submissionDir, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create submission directory: %w", err)
			}

			filePath = filepath.Join(submissionDir, fmt.Sprintf("%d.cpp", s.ID))
			if err := os.WriteFile(filePath, []byte(s.SourceCode), 0644); err != nil {
				return fmt.Errorf("failed to write submission file: %w", err)
			}
			return nil
		},
		Compensate: func(ctx context.Context) error {
			// Remove file if it was created
			if filePath != "" {
				return os.Remove(filePath)
			}
			return nil
		},
	})

	// Step 3: Queue operations (atomic)
	dt.AddStep(dbutil.DistributedTransactionStep{
		Name: "queue_operations",
		Execute: func(ctx context.Context) error {
			// Use Redis pipeline for atomicity
			pipe := h.rdb.Pipeline()
			pipe.LPush(ctx, "submission_queue", s.ID)
			pipe.LPush(ctx, "plagiarism_queue", s.ID)

			_, err := pipe.Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to push to processing queues: %w", err)
			}
			return nil
		},
		Compensate: func(ctx context.Context) error {
			// Remove from queues (best effort)
			pipe := h.rdb.Pipeline()
			pipe.LRem(ctx, "submission_queue", 1, s.ID)
			pipe.LRem(ctx, "plagiarism_queue", 1, s.ID)
			_, err := pipe.Exec(ctx)
			return err // Don't fail compensation if Redis cleanup fails
		},
	})

	// Execute the distributed transaction
	if err := dt.Execute(context.Background()); err != nil {
		return &SubmissionError{
			Message: "Failed to create submission",
			Code:    http.StatusInternalServerError,
			Err:     err,
		}
	}

	h.logger.Info("Submission created successfully with distributed transaction",
		zap.Int("submission_id", s.ID),
		zap.Int("problem_id", s.ProblemID),
		zap.String("transaction_id", submissionID))

	return nil
}

func (h *SubmissionsHandler) CreateSubmission(w http.ResponseWriter, r *http.Request) {
	var s Submission
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use transactional submission creation
	if err := h.createSubmissionTransactional(&s); err != nil {
		if submissionErr, ok := err.(*SubmissionError); ok {
			h.logger.Error("Submission creation failed", zap.Error(submissionErr.Err), zap.String("message", submissionErr.Message))
			http.Error(w, submissionErr.Message, submissionErr.Code)
		} else {
			h.logger.Error("Unexpected error during submission creation", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}