package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"codejudge/common/dbutil"
	"codejudge/common/env"
	"codejudge/common/health"
	"codejudge/common/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

var dbManager *dbutil.ConnectionManager

func connectDB() {
	databaseURL := env.Get("DATABASE_URL", "")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL not set")
	}
	dbManager = dbutil.ConnectManagerWithRetry(logger, databaseURL, 5, 2*time.Second)

	// Prepare commonly used statements for better performance
	prepareStatements()
}

func prepareStatements() {
	statements := map[string]string{
		"list_problems":    `SELECT id, title, description, difficulty FROM problems ORDER BY id`,
		"get_problem":      `SELECT id, title, description, difficulty FROM problems WHERE id = $1`,
		"create_problem":   `INSERT INTO problems (title, description, difficulty) VALUES ($1, $2, $3) RETURNING id`,
		"get_test_cases":   `SELECT id, problem_id, input, output FROM test_cases WHERE problem_id = $1 ORDER BY id`,
		"create_test_case": `INSERT INTO test_cases (problem_id, input, output) VALUES ($1, $2, $3) RETURNING id`,
	}

	for name, query := range statements {
		if err := dbManager.PrepareStatement(name, query); err != nil {
			logger.Fatal("Failed to prepare statement", zap.String("name", name), zap.Error(err))
		}
	}

	logger.Info("All SQL statements prepared successfully")
}

func createTable() {
	createProblemsTableSQL := `
    CREATE TABLE IF NOT EXISTS problems (
        id SERIAL PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        description TEXT,
        difficulty VARCHAR(50)
    );`
	_, err := dbManager.GetDB().Exec(createProblemsTableSQL)
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
	_, err = dbManager.GetDB().Exec(createTestCasesTableSQL)
	if err != nil {
		logger.Fatal("Failed to create 'test_cases' table", zap.Error(err))
	}
	logger.Info("'test_cases' table is ready")
}

func getProblems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := dbManager.QueryPrepared(ctx, "list_problems")
	if err != nil {
		serviceErr := httpx.NewServiceError(
			"Failed to retrieve problems",
			"DATABASE_ERROR",
			http.StatusInternalServerError,
			err,
		)
		httpx.ErrorWithDetails(w, serviceErr, logger)
		return
	}
	defer rows.Close()

	problems := []Problem{}
	for rows.Next() {
		var p Problem
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Difficulty); err != nil {
			serviceErr := httpx.NewServiceError(
				"Failed to process problem data",
				"DATA_PROCESSING_ERROR",
				http.StatusInternalServerError,
				err,
			)
			httpx.ErrorWithDetails(w, serviceErr, logger)
			return
		}
		problems = append(problems, p)
	}

	httpx.JSON(w, http.StatusOK, problems)
}

func getProblem(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		serviceErr := httpx.NewServiceError(
			"Invalid problem ID format",
			"INVALID_PARAMETER",
			http.StatusBadRequest,
			err,
		)
		httpx.ErrorWithDetails(w, serviceErr, logger)
		return
	}

	var p Problem
	ctx := r.Context()
	row := dbManager.QueryRowPrepared(ctx, "get_problem", id)
	err = row.Scan(&p.ID, &p.Title, &p.Description, &p.Difficulty)
	if err != nil {
		if err == sql.ErrNoRows {
			serviceErr := httpx.NewServiceError(
				"Problem not found",
				"NOT_FOUND",
				http.StatusNotFound,
				nil,
			)
			httpx.ErrorWithDetails(w, serviceErr, logger)
		} else {
			serviceErr := httpx.NewServiceError(
				"Failed to retrieve problem",
				"DATABASE_ERROR",
				http.StatusInternalServerError,
				err,
			)
			httpx.ErrorWithDetails(w, serviceErr, logger)
		}
		return
	}

	httpx.JSON(w, http.StatusOK, p)
}

func createProblem(w http.ResponseWriter, r *http.Request) {
	var p Problem
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		serviceErr := httpx.NewServiceError(
			"Invalid request body format",
			"INVALID_REQUEST_BODY",
			http.StatusBadRequest,
			err,
		)
		httpx.ErrorWithDetails(w, serviceErr, logger)
		return
	}

	// Basic validation
	if p.Title == "" {
		serviceErr := httpx.NewServiceError(
			"Problem title is required",
			"VALIDATION_ERROR",
			http.StatusBadRequest,
			nil,
		)
		httpx.ErrorWithDetails(w, serviceErr, logger)
		return
	}

	ctx := r.Context()
	row := dbManager.QueryRowPrepared(ctx, "create_problem", p.Title, p.Description, p.Difficulty)
	err := row.Scan(&p.ID)
	if err != nil {
		serviceErr := httpx.NewServiceError(
			"Failed to create problem",
			"DATABASE_ERROR",
			http.StatusInternalServerError,
			err,
		)
		httpx.ErrorWithDetails(w, serviceErr, logger)
		return
	}

	httpx.JSON(w, http.StatusCreated, p)
}

func createTestCase(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	problemID, err := strconv.Atoi(idStr)
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

	ctx := r.Context()
	row := dbManager.QueryRowPrepared(ctx, "create_test_case", tc.ProblemID, tc.Input, tc.Output)
	err = row.Scan(&tc.ID)
	if err != nil {
		logger.Error("Error creating test case", zap.Error(err))
		http.Error(w, "Failed to create test case", http.StatusInternalServerError)
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
	defer dbManager.Close()

	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(httpx.RecoveryMiddleware(logger))

	// Health endpoints
	r.Get("/health", health.HealthHandler())
	r.Get("/ready", health.ReadyHandler(func(ctx context.Context) error { return dbManager.GetDB().PingContext(ctx) }))

	// API routes
	r.Route("/problems", func(r chi.Router) {
		r.Get("/", getProblems)
		r.Post("/", createProblem)
		r.Get("/{id}", getProblem)
		r.Post("/{id}/testcases", createTestCase)
	})

	// Create server
	server := &http.Server{
		Addr:    ":8000",
		Handler: r,
	}

	// Start server with graceful shutdown
	httpx.StartServerWithGracefulShutdown(server, logger, 30*time.Second)
}
