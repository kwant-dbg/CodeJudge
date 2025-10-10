package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"codejudge/common/dbutil"
	"codejudge/common/httpx"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

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

type ProblemsHandler struct {
	logger    *zap.Logger
	dbManager *dbutil.ConnectionManager
}

func NewProblemsHandler(logger *zap.Logger, dbManager *dbutil.ConnectionManager) *ProblemsHandler {
	return &ProblemsHandler{
		logger:    logger,
		dbManager: dbManager,
	}
}

func (h *ProblemsHandler) PrepareStatements() {
	statements := map[string]string{
		"list_problems":    `SELECT id, title, description, difficulty FROM problems ORDER BY id`,
		"get_problem":      `SELECT id, title, description, difficulty FROM problems WHERE id = $1`,
		"create_problem":   `INSERT INTO problems (title, description, difficulty) VALUES ($1, $2, $3) RETURNING id`,
		"get_test_cases":   `SELECT id, problem_id, input, output FROM test_cases WHERE problem_id = $1 ORDER BY id`,
		"create_test_case": `INSERT INTO test_cases (problem_id, input, output) VALUES ($1, $2, $3) RETURNING id`,
	}

	for name, query := range statements {
		if err := h.dbManager.PrepareStatement(name, query); err != nil {
			h.logger.Fatal("Failed to prepare statement", zap.String("name", name), zap.Error(err))
		}
	}

	h.logger.Info("Problems SQL statements prepared successfully")
}

func (h *ProblemsHandler) CreateTables() {
	createProblemsTableSQL := `
    CREATE TABLE IF NOT EXISTS problems (
        id SERIAL PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        description TEXT,
        difficulty VARCHAR(50)
    );`
	_, err := h.dbManager.GetDB().Exec(createProblemsTableSQL)
	if err != nil {
		h.logger.Fatal("Failed to create 'problems' table", zap.Error(err))
	}
	h.logger.Info("'problems' table is ready")

	createTestCasesTableSQL := `
	CREATE TABLE IF NOT EXISTS test_cases (
		id SERIAL PRIMARY KEY,
		problem_id INTEGER NOT NULL,
		input TEXT NOT NULL,
		output TEXT NOT NULL,
		FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE
	);`
	_, err = h.dbManager.GetDB().Exec(createTestCasesTableSQL)
	if err != nil {
		h.logger.Fatal("Failed to create 'test_cases' table", zap.Error(err))
	}
	h.logger.Info("'test_cases' table is ready")
}

func (h *ProblemsHandler) GetProblems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.dbManager.QueryPrepared(ctx, "list_problems")
	if err != nil {
		serviceErr := httpx.NewServiceError(
			"Failed to retrieve problems",
			"DATABASE_ERROR",
			http.StatusInternalServerError,
			err,
		)
		httpx.ErrorWithDetails(w, serviceErr, h.logger)
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
			httpx.ErrorWithDetails(w, serviceErr, h.logger)
			return
		}
		problems = append(problems, p)
	}

	httpx.JSON(w, http.StatusOK, problems)
}

func (h *ProblemsHandler) GetProblem(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		serviceErr := httpx.NewServiceError(
			"Invalid problem ID format",
			"INVALID_PARAMETER",
			http.StatusBadRequest,
			err,
		)
		httpx.ErrorWithDetails(w, serviceErr, h.logger)
		return
	}

	var p Problem
	ctx := r.Context()
	row := h.dbManager.QueryRowPrepared(ctx, "get_problem", id)
	err = row.Scan(&p.ID, &p.Title, &p.Description, &p.Difficulty)
	if err != nil {
		if err == sql.ErrNoRows {
			serviceErr := httpx.NewServiceError(
				"Problem not found",
				"NOT_FOUND",
				http.StatusNotFound,
				nil,
			)
			httpx.ErrorWithDetails(w, serviceErr, h.logger)
		} else {
			serviceErr := httpx.NewServiceError(
				"Failed to retrieve problem",
				"DATABASE_ERROR",
				http.StatusInternalServerError,
				err,
			)
			httpx.ErrorWithDetails(w, serviceErr, h.logger)
		}
		return
	}

	httpx.JSON(w, http.StatusOK, p)
}

func (h *ProblemsHandler) CreateProblem(w http.ResponseWriter, r *http.Request) {
	var p Problem
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		serviceErr := httpx.NewServiceError(
			"Invalid request body format",
			"INVALID_REQUEST_BODY",
			http.StatusBadRequest,
			err,
		)
		httpx.ErrorWithDetails(w, serviceErr, h.logger)
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
		httpx.ErrorWithDetails(w, serviceErr, h.logger)
		return
	}

	ctx := r.Context()
	row := h.dbManager.QueryRowPrepared(ctx, "create_problem", p.Title, p.Description, p.Difficulty)
	err := row.Scan(&p.ID)
	if err != nil {
		serviceErr := httpx.NewServiceError(
			"Failed to create problem",
			"DATABASE_ERROR",
			http.StatusInternalServerError,
			err,
		)
		httpx.ErrorWithDetails(w, serviceErr, h.logger)
		return
	}

	httpx.JSON(w, http.StatusCreated, p)
}

func (h *ProblemsHandler) CreateTestCase(w http.ResponseWriter, r *http.Request) {
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
	row := h.dbManager.QueryRowPrepared(ctx, "create_test_case", tc.ProblemID, tc.Input, tc.Output)
	err = row.Scan(&tc.ID)
	if err != nil {
		h.logger.Error("Error creating test case", zap.Error(err))
		http.Error(w, "Failed to create test case", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tc)
}