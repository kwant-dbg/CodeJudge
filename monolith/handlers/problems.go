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
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Difficulty   string `json:"difficulty"`
	InputFormat  string `json:"input_format"`
	OutputFormat string `json:"output_format"`
}

type TestCase struct {
	ID        int    `json:"id"`
	ProblemID int    `json:"problem_id"`
	Input     string `json:"input"`
	Output    string `json:"output"`
	Sample    bool   `json:"sample"`
}

type ProblemWithTestCases struct {
	Problem
	TestCases []TestCase `json:"test_cases"`
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

func (h *ProblemsHandler) CreateTables() {
	createProblemsTableSQL := `
    CREATE TABLE IF NOT EXISTS problems (
        id SERIAL PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        description TEXT,
        difficulty VARCHAR(50),
        input_format TEXT,
        output_format TEXT
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
		sample BOOLEAN NOT NULL DEFAULT FALSE,
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
	db := h.dbManager.GetDB()
	query := `SELECT id, title, description, difficulty, input_format, output_format FROM problems ORDER BY id`
	
	rows, err := db.QueryContext(ctx, query)
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
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Difficulty, &p.InputFormat, &p.OutputFormat); err != nil {
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
	db := h.dbManager.GetDB()
	query := `SELECT id, title, description, difficulty, input_format, output_format FROM problems WHERE id = $1`
	
	row := db.QueryRowContext(ctx, query, id)
	err = row.Scan(&p.ID, &p.Title, &p.Description, &p.Difficulty, &p.InputFormat, &p.OutputFormat)
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

	// Fetch test cases for the problem
	testCasesQuery := `SELECT id, problem_id, input, output, sample FROM test_cases WHERE problem_id = $1 ORDER BY id`
	rows, err := db.QueryContext(ctx, testCasesQuery, id)
	if err != nil {
		serviceErr := httpx.NewServiceError(
			"Failed to retrieve test cases",
			"DATABASE_ERROR",
			http.StatusInternalServerError,
			err,
		)
		httpx.ErrorWithDetails(w, serviceErr, h.logger)
		return
	}
	defer rows.Close()

	testCases := []TestCase{}
	for rows.Next() {
		var tc TestCase
		if err := rows.Scan(&tc.ID, &tc.ProblemID, &tc.Input, &tc.Output, &tc.Sample); err != nil {
			serviceErr := httpx.NewServiceError(
				"Failed to process test case data",
				"DATA_PROCESSING_ERROR",
				http.StatusInternalServerError,
				err,
			)
			httpx.ErrorWithDetails(w, serviceErr, h.logger)
			return
		}
		testCases = append(testCases, tc)
	}

	response := ProblemWithTestCases{
		Problem:   p,
		TestCases: testCases,
	}

	httpx.JSON(w, http.StatusOK, response)
}

func (h *ProblemsHandler) CreateProblem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title        string     `json:"title"`
		Description  string     `json:"description"`
		Difficulty   string     `json:"difficulty"`
		InputFormat  string     `json:"input_format"`
		OutputFormat string     `json:"output_format"`
		TestCases    []TestCase `json:"test_cases"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	if req.Title == "" {
		serviceErr := httpx.NewServiceError(
			"Problem title is required",
			"VALIDATION_ERROR",
			http.StatusBadRequest,
			nil,
		)
		httpx.ErrorWithDetails(w, serviceErr, h.logger)
		return
	}

	if len(req.TestCases) == 0 {
		serviceErr := httpx.NewServiceError(
			"At least one test case is required",
			"VALIDATION_ERROR",
			http.StatusBadRequest,
			nil,
		)
		httpx.ErrorWithDetails(w, serviceErr, h.logger)
		return
	}

	ctx := r.Context()
	db := h.dbManager.GetDB()

	// Create the problem
	var problemID int
	createProblemQuery := `INSERT INTO problems (title, description, difficulty, input_format, output_format) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	row := db.QueryRowContext(ctx, createProblemQuery, req.Title, req.Description, req.Difficulty, req.InputFormat, req.OutputFormat)
	err := row.Scan(&problemID)
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

	// Create all test cases
	createTestCaseQuery := `INSERT INTO test_cases (problem_id, input, output, sample) VALUES ($1, $2, $3, $4) RETURNING id`
	for _, tc := range req.TestCases {
		var testCaseID int
		tcRow := db.QueryRowContext(ctx, createTestCaseQuery, problemID, tc.Input, tc.Output, tc.Sample)
		if err := tcRow.Scan(&testCaseID); err != nil {
			h.logger.Error("Failed to create test case", zap.Error(err), zap.Int("problemID", problemID))
			// Continue creating other test cases even if one fails
		}
	}

	h.logger.Info("Problem created with test cases",
		zap.Int("problemID", problemID),
		zap.Int("testCaseCount", len(req.TestCases)))

	response := Problem{
		ID:           problemID,
		Title:        req.Title,
		Description:  req.Description,
		Difficulty:   req.Difficulty,
		InputFormat:  req.InputFormat,
		OutputFormat: req.OutputFormat,
	}

	httpx.JSON(w, http.StatusCreated, response)
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
	db := h.dbManager.GetDB()
	query := `INSERT INTO test_cases (problem_id, input, output, sample) VALUES ($1, $2, $3, $4) RETURNING id`
	row := db.QueryRowContext(ctx, query, tc.ProblemID, tc.Input, tc.Output, tc.Sample)
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