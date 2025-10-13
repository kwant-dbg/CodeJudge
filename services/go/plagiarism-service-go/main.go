package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"codejudge/common/dbutil"
	"codejudge/common/env"
	"codejudge/common/health"
	"codejudge/common/httpx"
	"codejudge/common/redisutil"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

// LRU-based fingerprint cache to prevent memory leaks
type FingerprintCache struct {
	mu      sync.RWMutex
	cache   map[int]map[uint64]bool
	usage   map[int]time.Time
	maxSize int
	ttl     time.Duration
}

func NewFingerprintCache(maxSize int, ttl time.Duration) *FingerprintCache {
	fc := &FingerprintCache{
		cache:   make(map[int]map[uint64]bool),
		usage:   make(map[int]time.Time),
		maxSize: maxSize,
		ttl:     ttl,
	}

	// Start cleanup goroutine
	go fc.cleanupExpired()
	return fc
}

func (fc *FingerprintCache) Get(submissionID int) (map[uint64]bool, bool) {
	fc.mu.RLock()
	fp, exists := fc.cache[submissionID]
	fc.mu.RUnlock()

	if exists {
		// Update usage time
		fc.mu.Lock()
		fc.usage[submissionID] = time.Now()
		fc.mu.Unlock()
	}

	return fp, exists
}

func (fc *FingerprintCache) Set(submissionID int, fingerprint map[uint64]bool) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Check if we need to evict old entries
	if len(fc.cache) >= fc.maxSize {
		fc.evictLRU()
	}

	fc.cache[submissionID] = fingerprint
	fc.usage[submissionID] = time.Now()
}

func (fc *FingerprintCache) evictLRU() {
	// Find oldest entry
	var oldestID int
	var oldestTime time.Time
	first := true

	for id, usageTime := range fc.usage {
		if first || usageTime.Before(oldestTime) {
			oldestID = id
			oldestTime = usageTime
			first = false
		}
	}

	// Remove oldest entry
	delete(fc.cache, oldestID)
	delete(fc.usage, oldestID)
}

func (fc *FingerprintCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()

	for range ticker.C {
		fc.mu.Lock()
		now := time.Now()

		for id, usageTime := range fc.usage {
			if now.Sub(usageTime) > fc.ttl {
				delete(fc.cache, id)
				delete(fc.usage, id)
			}
		}
		fc.mu.Unlock()
	}
}

func (fc *FingerprintCache) Clear() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.cache = make(map[int]map[uint64]bool)
	fc.usage = make(map[int]time.Time)
}

var db *sql.DB
var rdb *redis.Client
var ctx = context.Background()
var fingerprintCache *FingerprintCache
var lshManager *PersistentLSHManager

const similarityThreshold = 0.75 // Lowered from 0.80 due to improved algorithm

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
        jaccard_similarity REAL DEFAULT 0.0,
        containment_a_in_b REAL DEFAULT 0.0,
        containment_b_in_a REAL DEFAULT 0.0,
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

func getSubmissionWithFingerprint(submissionID int) (*Submission, map[uint64]bool, error) {
	// Check cache first
	if fp, exists := fingerprintCache.Get(submissionID); exists {
		submission, err := getSubmission(submissionID)
		return submission, fp, err
	}

	// Get submission from database
	submission, err := getSubmission(submissionID)
	if err != nil {
		return submission, nil, err
	}

	// Generate and cache fingerprint
	fingerprint := GenerateFingerprint(submission.SourceCode)
	fingerprintCache.Set(submissionID, fingerprint)

	return submission, fingerprint, nil
}

func comprehensiveWorker() {
	logger.Info("Comprehensive plagiarism worker with LSH started")
	for {
		result, err := rdb.BLPop(ctx, 0, "plagiarism_queue").Result()
		if err != nil {
			logger.Error("Redis BLPop error", zap.Error(err))
			time.Sleep(time.Second)
			continue
		}

		submissionID, err := strconv.Atoi(result[1])
		if err != nil {
			logger.Error("Invalid submission ID", zap.String("id", result[1]))
			continue
		}

		newSub, newFp, err := getSubmissionWithFingerprint(submissionID)
		if err != nil {
			logger.Error("Failed to get submission", zap.Int("id", submissionID), zap.Error(err))
			continue
		}

		// Add this submission to the LSH index for future comparisons
		lshIndex := lshManager.GetOrCreateIndex(newSub.ProblemID)
		lshIndex.AddSubmission(submissionID, newFp)

		// Use LSH to find potentially similar submissions across ALL historical data
		// This is still O(1) average case due to LSH magic!
		candidateSubs, err := lshManager.FindSimilarSubmissionsForProblem(
			newSub.ProblemID,
			newSub.ID,
			newFp,
		)
		if err != nil {
			logger.Error("Failed to find similar submissions", zap.Error(err))
			continue
		}

		logger.Info("LSH found candidates",
			zap.Int("submission_id", submissionID),
			zap.Int("candidates_found", len(candidateSubs)),
		)

		// Now do detailed comparison only on LSH candidates
		for _, otherSub := range candidateSubs {
			otherFp, exists := fingerprintCache.Get(otherSub.ID)
			if !exists {
				otherFp = GenerateFingerprint(otherSub.SourceCode)
				fingerprintCache.Set(otherSub.ID, otherFp)
			}

			// OPTIMIZATION: Calculate all similarity metrics in one pass
			similarity, jaccard, containmentA, containmentB := calculateAllSimilarityMetrics(newFp, otherFp)

			if similarity >= similarityThreshold {
				logger.Info("High similarity detected",
					zap.Float64("weighted_similarity", similarity),
					zap.Int("submission_a", newSub.ID),
					zap.Int("submission_b", otherSub.ID))

				subA, subB := newSub.ID, otherSub.ID
				if subA > subB {
					subA, subB = subB, subA
				}

				// Store with all metrics (already calculated)
				query := `INSERT INTO plagiarism_reports (submission_a, submission_b, similarity, jaccard_similarity, containment_a_in_b, containment_b_in_a) 
						  VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (submission_a, submission_b) DO UPDATE SET
						  similarity = EXCLUDED.similarity, jaccard_similarity = EXCLUDED.jaccard_similarity,
						  containment_a_in_b = EXCLUDED.containment_a_in_b, containment_b_in_a = EXCLUDED.containment_b_in_a`

				_, err := db.Exec(query, subA, subB, similarity, jaccard, containmentA, containmentB)
				if err != nil {
					logger.Error("Failed to insert plagiarism report", zap.Error(err))
				}
			}
		}
	}
}

// getRecentSubmissionsForProblem gets only recent submissions to reduce comparison load
func getRecentSubmissionsForProblem(problemID, excludeID, limit int) ([]Submission, error) {
	query := `SELECT id, problem_id, source_code FROM submissions 
			  WHERE problem_id = $1 AND id != $2 
			  ORDER BY id DESC LIMIT $3`

	rows, err := db.Query(query, problemID, excludeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []Submission
	for rows.Next() {
		var s Submission
		if err := rows.Scan(&s.ID, &s.ProblemID, &s.SourceCode); err != nil {
			logger.Error("Failed to scan submission", zap.Error(err))
			continue
		}
		submissions = append(submissions, s)
	}
	return submissions, nil
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

	// Initialize optimized fingerprint cache (max 10000 entries, 30 min TTL)
	fingerprintCache = NewFingerprintCache(10000, 30*time.Minute)

	connectDB()
	connectRedis()
	createTable()
	defer db.Close()

	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(httpx.RecoveryMiddleware(logger))

	// Health endpoints
	r.Get("/health", health.HealthHandler())
	r.Get("/ready", health.ReadyHandler(func(ctx context.Context) error {
		if err := db.PingContext(ctx); err != nil {
			return err
		}
		return rdb.Ping(ctx).Err()
	}))

	// API routes
	r.Get("/reports", reportsHandler)

	// Start the improved worker
	go comprehensiveWorker()

	// Create server
	server := &http.Server{
		Addr:    ":8002",
		Handler: r,
	}

	// Start server with graceful shutdown
	httpx.StartServerWithGracefulShutdown(server, logger, 30*time.Second)
}
