package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"codejudge/common/dbutil"
	"codejudge/common/env"
	"codejudge/common/health"
	"codejudge/common/redisutil"
	"database/sql"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var logger *zap.Logger

type Submission struct {
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

var db *sql.DB
var rdb *redis.Client
var ctx = context.Background()

const similarityThreshold = 0.80

func connectDB() {
	databaseURL := env.Get("DATABASE_URL", "")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL not set")
	}
	db = dbutil.ConnectWithRetry(logger, databaseURL, 5, 2*time.Second)
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
    CREATE TABLE IF NOT EXISTS plagiarism_reports (
        id SERIAL PRIMARY KEY,
        submission_a INTEGER NOT NULL,
        submission_b INTEGER NOT NULL,
        similarity REAL NOT NULL,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(submission_a, submission_b)
    );`
	if _, err := db.Exec(createTableSQL); err != nil {
		logger.Fatal("Failed to create 'plagiarism_reports' table", zap.Error(err))
	}
	logger.Info("'plagiarism_reports' table is ready")
}

func getSubmission(id int) (*Submission, error) {
	s := &Submission{}
	query := "SELECT id, problem_id, source_code FROM submissions WHERE id = $1"
	err := db.QueryRow(query, id).Scan(&s.ID, &s.ProblemID, &s.SourceCode)
	return s, err
}

func getSubmissionsForProblem(problemId, excludeId int) ([]Submission, error) {
	subs := []Submission{}
	query := "SELECT id, problem_id, source_code FROM submissions WHERE problem_id = $1 AND id != $2"
	rows, err := db.Query(query, problemId, excludeId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s Submission
		if err := rows.Scan(&s.ID, &s.ProblemID, &s.SourceCode); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func worker() {
	logger.Info("Plagiarism worker started")
	for {
		result, err := rdb.BLPop(ctx, 0, "plagiarism_queue").Result()
		if err != nil {
			continue
		}

		submissionID, _ := strconv.Atoi(result[1])
		newSub, err := getSubmission(submissionID)
		if err != nil {
			continue
		}

		otherSubs, err := getSubmissionsForProblem(newSub.ProblemID, newSub.ID)
		if err != nil {
			continue
		}

		fpA := GenerateFingerprint(newSub.SourceCode)
		for _, otherSub := range otherSubs {
			fpB := GenerateFingerprint(otherSub.SourceCode)
			similarity := CalculateJaccard(fpA, fpB)

			if similarity >= similarityThreshold {
				logger.Info("High similarity detected",
					zap.Float64("similarity", similarity),
					zap.Int("submission_a", newSub.ID),
					zap.Int("submission_b", otherSub.ID))
				subA, subB := newSub.ID, otherSub.ID
				if subA > subB {
					subA, subB = subB, subA
				}
				query := `INSERT INTO plagiarism_reports (submission_a, submission_b, similarity) 
								VALUES ($1, $2, $3) ON CONFLICT (submission_a, submission_b) DO NOTHING`
				db.Exec(query, subA, subB, similarity)
			}
		}
	}
}

func reportsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, submission_a, submission_b, similarity, created_at FROM plagiarism_reports ORDER BY similarity DESC")
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

func main() {
	logger, _ = zap.NewProduction()
	defer logger.Sync()

	connectDB()
	connectRedis()
	createTable()
	defer db.Close()

	http.HandleFunc("/health", health.HealthHandler())
	http.HandleFunc("/ready", health.ReadyHandler(func(ctx context.Context) error {
		if err := db.PingContext(ctx); err != nil {
			return err
		}
		return rdb.Ping(ctx).Err()
	}))

	go worker()

	http.HandleFunc("/plagiarism/reports", reportsHandler)
	logger.Info("Plagiarism Service starting on port 8002")
	if err := http.ListenAndServe(":8002", nil); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}
