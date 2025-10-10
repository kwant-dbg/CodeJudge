package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"codejudge/common/dbutil"
	"codejudge/common/env"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type Submission struct {
	ID         int    `json:"id"`
	ProblemID  int    `json:"problem_id"`
	SourceCode string `json:"source_code"`
	Language   string `json:"language,omitempty"`
	Verdict    string `json:"verdict,omitempty"`
	Status     string `json:"status,omitempty"`
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

// Simplified submission creation with proper error handling
func (h *SubmissionsHandler) createSubmissionTransactional(s *Submission) error {
	// Use a simple context without external cancellation
	ctx := context.Background()

	// Step 1: Insert into database with direct connection
	db := h.dbManager.GetDB()

	query := "INSERT INTO submissions (problem_id, source_code) VALUES ($1, $2) RETURNING id"
	err := db.QueryRowContext(ctx, query, s.ProblemID, s.SourceCode).Scan(&s.ID)
	if err != nil {
		return &SubmissionError{
			Message: "Failed to insert submission into database",
			Code:    http.StatusInternalServerError,
			Err:     err,
		}
	}

	// Step 2: Write to filesystem (best effort)
	submissionDir := env.Get("SUBMISSION_STORAGE_PATH", "/app/submissions")
	if err := os.MkdirAll(submissionDir, os.ModePerm); err != nil {
		h.logger.Warn("Failed to create submission directory", zap.Error(err))
	} else {
		filePath := filepath.Join(submissionDir, fmt.Sprintf("%d.cpp", s.ID))
		if err := os.WriteFile(filePath, []byte(s.SourceCode), 0644); err != nil {
			h.logger.Warn("Failed to write submission file", zap.Error(err))
		}
	}

	// Step 3: Push to queues (best effort)
	pipe := h.rdb.Pipeline()
	pipe.LPush(ctx, "submission_queue", s.ID)
	pipe.LPush(ctx, "plagiarism_queue", s.ID)

	if _, err := pipe.Exec(ctx); err != nil {
		h.logger.Warn("Failed to push to processing queues", zap.Error(err))
	}

	h.logger.Info("Submission created successfully",
		zap.Int("submission_id", s.ID),
		zap.Int("problem_id", s.ProblemID))

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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(submissionErr.Code)
			json.NewEncoder(w).Encode(map[string]string{"error": submissionErr.Message})
		} else {
			h.logger.Error("Unexpected error during submission creation", zap.Error(err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Internal server error"})
		}
		return
	}

	// Set default status for new submissions
	s.Status = "Pending"
	s.Verdict = ""

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

func (h *SubmissionsHandler) GetSubmission(w http.ResponseWriter, r *http.Request) {
	submissionID := chi.URLParam(r, "id")

	db := h.dbManager.GetDB()
	query := "SELECT id, problem_id, source_code, verdict FROM submissions WHERE id = $1"

	var s Submission
	var verdict sql.NullString

	err := db.QueryRowContext(r.Context(), query, submissionID).Scan(&s.ID, &s.ProblemID, &s.SourceCode, &verdict)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Submission not found", http.StatusNotFound)
			return
		}
		h.logger.Error("Failed to get submission", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create response with verdict
	response := map[string]interface{}{
		"id":          s.ID,
		"problem_id":  s.ProblemID,
		"source_code": s.SourceCode,
		"verdict":     nil,
		"status":      "Pending",
	}

	if verdict.Valid && verdict.String != "" {
		response["verdict"] = verdict.String
		response["status"] = "Completed"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

