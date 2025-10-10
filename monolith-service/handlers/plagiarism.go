package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"codejudge/common/dbutil"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type PlagiarismSubmission struct {
	ID         int
	ProblemID  int
	SourceCode string
}

type Report struct {
	ID          int       `json:"id"`
	SubmissionA int       `json:"submission_a"`
	SubmissionB int       `json:"submission_b"`
	Similarity  float64   `json:"similarity"`
	CreatedAt   time.Time `json:"created_at"`
}

type PlagiarismHandler struct {
	logger    *zap.Logger
	dbManager *dbutil.ConnectionManager
	rdb       *redis.Client
	ctx       context.Context
}

func NewPlagiarismHandler(logger *zap.Logger, dbManager *dbutil.ConnectionManager, rdb *redis.Client) *PlagiarismHandler {
	return &PlagiarismHandler{
		logger:    logger,
		dbManager: dbManager,
		rdb:       rdb,
		ctx:       context.Background(),
	}
}

func (h *PlagiarismHandler) CreateTables() {
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS plagiarism_reports (
        id SERIAL PRIMARY KEY,
        submission_a INTEGER NOT NULL,
        submission_b INTEGER NOT NULL,
        similarity REAL NOT NULL,
        jaccard_similarity REAL DEFAULT 0.0,
        containment_a_in_b REAL DEFAULT 0.0,
        containment_b_in_a REAL DEFAULT 0.0,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(submission_a, submission_b)
    );`
	if _, err := h.dbManager.GetDB().Exec(createTableSQL); err != nil {
		h.logger.Fatal("Failed to create 'plagiarism_reports' table", zap.Error(err))
	}
	h.logger.Info("'plagiarism_reports' table is ready")
}

func (h *PlagiarismHandler) startWorker() {
	h.logger.Info("Plagiarism worker started")
	go func() {
		for {
			result, err := h.rdb.BLPop(h.ctx, 0, "plagiarism_queue").Result()
			if err != nil {
				h.logger.Error("Redis BLPop error", zap.Error(err))
				time.Sleep(time.Second)
				continue
			}

			submissionID, err := strconv.Atoi(result[1])
			if err != nil {
				h.logger.Error("Invalid submission ID", zap.String("id", result[1]))
				continue
			}

			h.logger.Info("Processing plagiarism check", zap.Int("submission_id", submissionID))
			// For monolith version, we'll implement a basic plagiarism check
			// In a real implementation, this would do sophisticated comparison
		}
	}()
}

func (h *PlagiarismHandler) StartWorker() {
	h.startWorker()
}

func (h *PlagiarismHandler) GetReports(w http.ResponseWriter, r *http.Request) {
	rows, err := h.dbManager.GetDB().Query("SELECT id, submission_a, submission_b, similarity, created_at FROM plagiarism_reports ORDER BY similarity DESC")
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	reports := []Report{}
	for rows.Next() {
		var report Report
		if err := rows.Scan(&report.ID, &report.SubmissionA, &report.SubmissionB, &report.Similarity, &report.CreatedAt); err != nil {
			http.Error(w, "Failed to scan report", http.StatusInternalServerError)
			return
		}
		reports = append(reports, report)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}
