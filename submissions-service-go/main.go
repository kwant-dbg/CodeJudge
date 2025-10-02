package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var logger *zap.Logger

type Submission struct {
	ID         int    `json:"id"`
	ProblemID  int    `json:"problem_id"`
	SourceCode string `json:"source_code"`
}

var db *sql.DB
var rdb *redis.Client
var ctx = context.Background()

func buildRedisOptions(redisURL string) (*redis.Options, error) {
	if !strings.Contains(redisURL, "://") {
		redisURL = "redis://" + redisURL
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	if opts.TLSConfig != nil {
		host := opts.Addr
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		opts.TLSConfig.MinVersion = tls.VersionTLS12
		if opts.TLSConfig.ServerName == "" {
			opts.TLSConfig.ServerName = host
		}
	}

	return opts, nil
}

func connectDB() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL not set")
	}

	var err error
	for i := 0; i < 5; i++ {
		conn, openErr := sql.Open("postgres", databaseURL)
		if openErr != nil {
			err = openErr
		} else {
			if pingErr := conn.Ping(); pingErr == nil {
				db = conn
				logger.Info("Successfully connected to the database")
				return
			} else {
				err = pingErr
				conn.Close()
			}
		}
		time.Sleep(2 * time.Second)
	}
	logger.Fatal("Failed to connect to the database", zap.Error(err))
}

func connectRedis() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		logger.Fatal("REDIS_URL not set")
	}

	var err error
	for i := 0; i < 5; i++ {
		opts, parseErr := buildRedisOptions(redisURL)
		if parseErr != nil {
			err = parseErr
			break
		}

		client := redis.NewClient(opts)
		if pingErr := client.Ping(ctx).Err(); pingErr == nil {
			rdb = client
			logger.Info("Successfully connected to Redis")
			return
		} else {
			err = pingErr
			client.Close()
		}
		time.Sleep(2 * time.Second)
	}
	logger.Fatal("Could not connect to Redis", zap.Error(err))
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
	if _, err := db.Exec(createTableSQL); err != nil {
		logger.Fatal("Failed to create 'submissions' table", zap.Error(err))
	}
	logger.Info("'submissions' table is ready")
}

func submissionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var s Submission
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := "INSERT INTO submissions (problem_id, source_code) VALUES ($1, $2) RETURNING id"
	err := db.QueryRow(query, s.ProblemID, s.SourceCode).Scan(&s.ID)
	if err != nil {
		http.Error(w, "Failed to create submission", http.StatusInternalServerError)
		return
	}

	submissionDir := os.Getenv("SUBMISSION_STORAGE_PATH")
	if submissionDir == "" {
		submissionDir = "/app/submissions"
	}

	if err := os.MkdirAll(submissionDir, os.ModePerm); err != nil {
		http.Error(w, "Failed to create submission directory", http.StatusInternalServerError)
		logger.Error("Error creating directory", zap.Error(err))
		return
	}
	filePath := filepath.Join(submissionDir, fmt.Sprintf("%d.cpp", s.ID))
	if err := os.WriteFile(filePath, []byte(s.SourceCode), 0644); err != nil {
		http.Error(w, "Failed to write submission file", http.StatusInternalServerError)
		logger.Error("Error writing file", zap.Error(err))
		return
	}

	if err := rdb.LPush(ctx, "submission_queue", s.ID).Err(); err != nil {
		http.Error(w, "Failed to push to judge queue", http.StatusInternalServerError)
		return
	}

	if err := rdb.LPush(ctx, "plagiarism_queue", s.ID).Err(); err != nil {
		http.Error(w, "Failed to push to plagiarism queue", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

func main() {
	logger, _ = zap.NewProduction()
	defer logger.Sync()

	connectDB()
	connectRedis()
	createTable()
	defer db.Close()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		readyCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(readyCtx); err != nil {
			http.Error(w, "database not ready", http.StatusServiceUnavailable)
			return
		}

		if err := rdb.Ping(readyCtx).Err(); err != nil {
			http.Error(w, "redis not ready", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/submissions", submissionHandler)
	logger.Info("Submissions Service starting on port 8001")
	if err := http.ListenAndServe(":8001", nil); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}
