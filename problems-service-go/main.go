package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var logger *zap.Logger

type Problem struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Difficulty  string `json:"difficulty"`
}

type TestCase struct {
	ID        int    `json:"id"`
	ProblemID int    `json:"problem_id"`
	Input     string `json:"input"`
	Output    string `json:"output"`
}

var db *sql.DB

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

func createTable() {
	createProblemsTableSQL := `
    CREATE TABLE IF NOT EXISTS problems (
        id SERIAL PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        description TEXT,
        difficulty VARCHAR(50)
    );`
	_, err := db.Exec(createProblemsTableSQL)
	if err != nil {
		logger.Fatal("Failed to create 'problems' table", zap.Error(err))
	}
	logger.Info("'problems' table is ready")

	createTestCasesTableSQL := `
	CREATE TABLE IF NOT EXISTS test_cases (
		id SERIAL PRIMARY KEY,
		problem_id INTEGER NOT NULL,
		input TEXT NOT NULL,
		output TEXT NOT NULL,
		FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE
	);`
	_, err = db.Exec(createTestCasesTableSQL)
	if err != nil {
		logger.Fatal("Failed to create 'test_cases' table", zap.Error(err))
	}
	logger.Info("'test_cases' table is ready")
}

func problemsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getProblems(w, r)
	case http.MethodPost:
		createProblem(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getProblems(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, description, difficulty FROM problems")
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	problems := []Problem{}
	for rows.Next() {
		var p Problem
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Difficulty); err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}
		problems = append(problems, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(problems)
}

func createProblem(w http.ResponseWriter, r *http.Request) {
	var p Problem
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := "INSERT INTO problems (title, description, difficulty) VALUES ($1, $2, $3) RETURNING id"
	err := db.QueryRow(query, p.Title, p.Description, p.Difficulty).Scan(&p.ID)
	if err != nil {
		http.Error(w, "Failed to create problem", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func testCaseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// URL path is expected to be /problems/:id/testcases
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL format. Expected /problems/:id/testcases", http.StatusBadRequest)
		return
	}
	problemID, err := strconv.Atoi(parts[1])
	if err != nil {
		http.Error(w, "Invalid problem ID", http.StatusBadRequest)
		return
	}

	var tc TestCase
	if err := json.NewDecoder(r.Body).Decode(&tc); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	tc.ProblemID = problemID

	query := "INSERT INTO test_cases (problem_id, input, output) VALUES ($1, $2, $3) RETURNING id"
	err = db.QueryRow(query, tc.ProblemID, tc.Input, tc.Output).Scan(&tc.ID)
	if err != nil {
		http.Error(w, "Failed to create test case", http.StatusInternalServerError)
		logger.Error("Error creating test case", zap.Error(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tc)
}

func main() {
	logger, _ = zap.NewProduction()
	defer logger.Sync()

	connectDB()
	createTable()
	defer db.Close()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			http.Error(w, "database not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/problems/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/testcases") {
			testCaseHandler(w, r)
		} else {
			problemsHandler(w, r)
		}
	})

	logger.Info("Problems Service starting on port 8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		logger.Fatal("Server failed to start", zap.Error(err))
	}
}
