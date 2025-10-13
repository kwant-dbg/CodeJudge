package main

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
	"codejudge/common/health"
	"codejudge/common/redisutil"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var (
	logger    *zap.Logger
	dbManager *dbutil.ConnectionManager
	txManager *dbutil.TransactionManager
	rdb       *redis.Client
	ctx       = context.Background()
)

type Submission struct {
	ID         int    `json:"id"`
	ProblemID  int    `json:"problem_id"`
	SourceCode string `json:"source_code"`
}

func connectDB() {
	databaseURL := env.Get("DATABASE_URL", "")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL not set")
	}
	dbManager = dbutil.ConnectManagerWithRetry(logger, databaseURL, 5, 2*time.Second)
	txManager = dbutil.NewTransactionManager(dbManager, logger)
}

func connectRedis() {
	redisURL := env.Get("REDIS_URL", "")
	if redisURL == "" {
		logger.Fatal("REDIS_URL not set")
	}
	rdb = redisutil.ConnectWithRetry(ctx, logger, redisURL, 5, 2*time.Second)
}

func createTable() {
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS submissions (
        id SERIAL PRIMARY KEY,
        problem_id INTEGER NOT NULL,
        source_code TEXT NOT NULL,
        verdict VARCHAR(50) DEFAULT 'Pending',
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );`
	if _, err := dbManager.GetDB().Exec(createTableSQL); err != nil {
		logger.Fatal("Failed to create 'submissions' table", zap.Error(err))
	}
	logger.Info("'submissions' table is ready")
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
func createSubmissionTransactional(s *Submission) error {
	submissionID := fmt.Sprintf("sub_%d_%d", s.ProblemID, time.Now().UnixNano())

	// Create distributed transaction with proper compensation
	dt := dbutil.NewDistributedTransaction(submissionID, txManager, logger)

	var filePath string

	// Step 1: Database insertion with proper isolation
	dt.AddStep(dbutil.DistributedTransactionStep{
		Name: "database_insert",
		Execute: func(ctx context.Context) error {
			return txManager.ExecuteStrictTransaction(ctx, func(tx *sql.Tx) error {
				query := "INSERT INTO submissions (problem_id, source_code) VALUES ($1, $2) RETURNING id"
				return tx.QueryRowContext(ctx, query, s.ProblemID, s.SourceCode).Scan(&s.ID)
			})
		},
		Compensate: func(ctx context.Context) error {
			// Remove from database if other steps fail
			return txManager.ExecuteStrictTransaction(ctx, func(tx *sql.Tx) error {
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
			pipe := rdb.Pipeline()
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
			pipe := rdb.Pipeline()
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

	logger.Info("Submission created successfully with distributed transaction",
		zap.Int("submission_id", s.ID),
		zap.Int("problem_id", s.ProblemID),
		zap.String("transaction_id", submissionID))

	return nil
}

func createSubmission(w http.ResponseWriter, r *http.Request) {
	var s Submission
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use transactional submission creation
	if err := createSubmissionTransactional(&s); err != nil {
		if submissionErr, ok := err.(*SubmissionError); ok {
			logger.Error("Submission creation failed", zap.Error(submissionErr.Err), zap.String("message", submissionErr.Message))
			http.Error(w, submissionErr.Message, submissionErr.Code)
		} else {
			logger.Error("Unexpected error during submission creation", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

func submissionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	createSubmission(w, r)
}

func main() {
	logger, _ = zap.NewProduction()
	defer logger.Sync()

	connectDB()
	connectRedis()
	createTable()
	defer dbManager.Close()

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Routes
	r.Post("/submissions", createSubmission)
	r.Get("/health", health.HealthHandler())
	r.Get("/ready", health.ReadyHandler(func(ctx context.Context) error {
		// Check both database and Redis connectivity
		if err := dbManager.GetDB().PingContext(ctx); err != nil {
			return fmt.Errorf("database ping failed: %w", err)
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis ping failed: %w", err)
		}
		return nil
	}))

	port := env.Get("PORT", "8080")
	logger.Info("Server starting", zap.String("port", port))
	http.ListenAndServe(":"+port, r)
}
