package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"codejudge/common/dbutil"
	"codejudge/common/httpx"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Contest struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	CreatedBy   int       `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"`                // draft, upcoming, active, finished
	FreezeTime  *int      `json:"freeze_time,omitempty"` // Minutes before end to freeze leaderboard
}

type ContestProblem struct {
	ID          int    `json:"id"`
	ContestID   int    `json:"contest_id"`
	ProblemID   int    `json:"problem_id"`
	Points      int    `json:"points"`
	OrderNum    int    `json:"order"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Difficulty  string `json:"difficulty,omitempty"`
}

type ContestParticipant struct {
	ContestID int       `json:"contest_id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	JoinedAt  time.Time `json:"joined_at"`
}

type LeaderboardEntry struct {
	Rank             int       `json:"rank"`
	UserID           int       `json:"user_id"`
	Username         string    `json:"username"`
	TotalScore       int       `json:"total_score"`
	ProblemsSolved   int       `json:"problems_solved"`
	LastSubmissionAt time.Time `json:"last_submission_at"`
	PenaltyMinutes   int       `json:"penalty_minutes"`
}

type ContestSubmission struct {
	ID          int       `json:"id"`
	ContestID   int       `json:"contest_id"`
	ProblemID   int       `json:"problem_id"`
	UserID      int       `json:"user_id"`
	SourceCode  string    `json:"source_code"`
	Language    string    `json:"language"`
	Verdict     string    `json:"verdict"`
	Points      int       `json:"points"`
	SubmittedAt time.Time `json:"submitted_at"`
}

type ContestsHandler struct {
	logger    *zap.Logger
	dbManager *dbutil.ConnectionManager
}

func NewContestsHandler(logger *zap.Logger, dbManager *dbutil.ConnectionManager) *ContestsHandler {
	return &ContestsHandler{
		logger:    logger,
		dbManager: dbManager,
	}
}

func (h *ContestsHandler) CreateTables() {
	// Contests table
	createContestsTableSQL := `
    CREATE TABLE IF NOT EXISTS contests (
        id SERIAL PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        description TEXT,
        start_time TIMESTAMP WITH TIME ZONE NOT NULL,
        end_time TIMESTAMP WITH TIME ZONE NOT NULL,
        created_by INTEGER NOT NULL,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        freeze_time INTEGER DEFAULT 0,
        CONSTRAINT valid_time_range CHECK (end_time > start_time)
    );`

	// Contest problems (many-to-many with problems)
	createContestProblemsSQL := `
    CREATE TABLE IF NOT EXISTS contest_problems (
        id SERIAL PRIMARY KEY,
        contest_id INTEGER NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
        problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
        points INTEGER NOT NULL DEFAULT 100,
        order_num INTEGER NOT NULL,
        UNIQUE(contest_id, problem_id)
    );`

	// Contest participants
	createParticipantsSQL := `
    CREATE TABLE IF NOT EXISTS contest_participants (
        contest_id INTEGER NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
        user_id INTEGER NOT NULL,
        joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        PRIMARY KEY (contest_id, user_id)
    );`

	// Add contest_id to submissions table (if not exists)
	addContestToSubmissionsSQL := `
    ALTER TABLE submissions 
    ADD COLUMN IF NOT EXISTS contest_id INTEGER REFERENCES contests(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS user_id INTEGER,
    ADD COLUMN IF NOT EXISTS points INTEGER DEFAULT 0;`

	// Create indexes for better performance
	createIndexesSQL := `
    CREATE INDEX IF NOT EXISTS idx_contest_problems_contest ON contest_problems(contest_id);
    CREATE INDEX IF NOT EXISTS idx_contest_participants_contest ON contest_participants(contest_id);
    CREATE INDEX IF NOT EXISTS idx_submissions_contest ON submissions(contest_id, user_id);
    CREATE INDEX IF NOT EXISTS idx_contests_time ON contests(start_time, end_time);`

	// Execute in order: contests table first, then related tables, then ALTER, then indexes
	sqlStatements := []struct {
		name string
		sql  string
	}{
		{"contests", createContestsTableSQL},
		{"contest_problems", createContestProblemsSQL},
		{"contest_participants", createParticipantsSQL},
		{"submissions_contest", addContestToSubmissionsSQL},
		{"indexes", createIndexesSQL},
	}

	for _, stmt := range sqlStatements {
		if _, err := h.dbManager.GetDB().Exec(stmt.sql); err != nil {
			h.logger.Fatal("Failed to create table/index", zap.String("name", stmt.name), zap.Error(err))
		}
	}

	h.logger.Info("Contest tables are ready")
}

// GetContestStatus determines the current status of a contest
func (h *ContestsHandler) GetContestStatus(startTime, endTime time.Time) string {
	now := time.Now()
	if now.Before(startTime) {
		return "upcoming"
	} else if now.After(endTime) {
		return "finished"
	}
	return "active"
}

// CreateContest creates a new contest (admin only)
func (h *ContestsHandler) CreateContest(w http.ResponseWriter, r *http.Request) {
	var contest Contest
	if err := json.NewDecoder(r.Body).Decode(&contest); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate
	if contest.Title == "" {
		httpx.Error(w, http.StatusBadRequest, "Title is required")
		return
	}
	if contest.EndTime.Before(contest.StartTime) {
		httpx.Error(w, http.StatusBadRequest, "End time must be after start time")
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "User ID not found")
		return
	}

	query := `INSERT INTO contests (title, description, start_time, end_time, created_by, freeze_time)
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`

	err := h.dbManager.GetDB().QueryRow(query,
		contest.Title, contest.Description, contest.StartTime, contest.EndTime, userID, contest.FreezeTime).
		Scan(&contest.ID, &contest.CreatedAt)

	if err != nil {
		h.logger.Error("Failed to create contest", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "Failed to create contest")
		return
	}

	contest.CreatedBy = userID
	contest.Status = h.GetContestStatus(contest.StartTime, contest.EndTime)
	httpx.JSON(w, http.StatusCreated, contest)
}

// ListContests lists all contests with optional status filter
func (h *ContestsHandler) ListContests(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status") // upcoming, active, finished

	query := `SELECT id, title, description, start_time, end_time, created_by, created_at, freeze_time
	          FROM contests ORDER BY start_time DESC`

	rows, err := h.dbManager.GetDB().Query(query)
	if err != nil {
		h.logger.Error("Failed to list contests", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "Failed to list contests")
		return
	}
	defer rows.Close()

	contests := []Contest{}
	for rows.Next() {
		var c Contest
		if err := rows.Scan(&c.ID, &c.Title, &c.Description, &c.StartTime, &c.EndTime,
			&c.CreatedBy, &c.CreatedAt, &c.FreezeTime); err != nil {
			continue
		}
		c.Status = h.GetContestStatus(c.StartTime, c.EndTime)

		// Apply status filter if provided
		if statusFilter == "" || c.Status == statusFilter {
			contests = append(contests, c)
		}
	}

	httpx.JSON(w, http.StatusOK, contests)
}

// GetContest gets a single contest by ID
func (h *ContestsHandler) GetContest(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "Invalid contest ID")
		return
	}

	query := `SELECT id, title, description, start_time, end_time, created_by, created_at, freeze_time
	          FROM contests WHERE id = $1`

	var contest Contest
	err = h.dbManager.GetDB().QueryRow(query, id).Scan(
		&contest.ID, &contest.Title, &contest.Description, &contest.StartTime, &contest.EndTime,
		&contest.CreatedBy, &contest.CreatedAt, &contest.FreezeTime)

	if err == sql.ErrNoRows {
		httpx.Error(w, http.StatusNotFound, "Contest not found")
		return
	} else if err != nil {
		h.logger.Error("Failed to get contest", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "Failed to get contest")
		return
	}

	contest.Status = h.GetContestStatus(contest.StartTime, contest.EndTime)
	httpx.JSON(w, http.StatusOK, contest)
}

// AddProblemToContest adds a problem to a contest
func (h *ContestsHandler) AddProblemToContest(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	contestID, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "Invalid contest ID")
		return
	}

	var cp ContestProblem
	if err := json.NewDecoder(r.Body).Decode(&cp); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	cp.ContestID = contestID
	query := `INSERT INTO contest_problems (contest_id, problem_id, points, order_num)
	          VALUES ($1, $2, $3, $4) RETURNING id`

	err = h.dbManager.GetDB().QueryRow(query, cp.ContestID, cp.ProblemID, cp.Points, cp.OrderNum).Scan(&cp.ID)
	if err != nil {
		h.logger.Error("Failed to add problem to contest", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "Failed to add problem")
		return
	}

	httpx.JSON(w, http.StatusCreated, cp)
}

// GetContestProblems gets all problems in a contest
func (h *ContestsHandler) GetContestProblems(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	contestID, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "Invalid contest ID")
		return
	}

	query := `SELECT cp.id, cp.contest_id, cp.problem_id, cp.points, cp.order_num,
	                 p.title, p.description, p.difficulty
	          FROM contest_problems cp
	          JOIN problems p ON cp.problem_id = p.id
	          WHERE cp.contest_id = $1
	          ORDER BY cp.order_num`

	rows, err := h.dbManager.GetDB().Query(query, contestID)
	if err != nil {
		h.logger.Error("Failed to get contest problems", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "Failed to get problems")
		return
	}
	defer rows.Close()

	problems := []ContestProblem{}
	for rows.Next() {
		var cp ContestProblem
		if err := rows.Scan(&cp.ID, &cp.ContestID, &cp.ProblemID, &cp.Points, &cp.OrderNum,
			&cp.Title, &cp.Description, &cp.Difficulty); err != nil {
			continue
		}
		problems = append(problems, cp)
	}

	httpx.JSON(w, http.StatusOK, problems)
}

// RegisterForContest registers a user for a contest
func (h *ContestsHandler) RegisterForContest(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	contestID, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "Invalid contest ID")
		return
	}

	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "User ID not found")
		return
	}

	// Check if contest exists and hasn't ended
	var endTime time.Time
	err = h.dbManager.GetDB().QueryRow("SELECT end_time FROM contests WHERE id = $1", contestID).Scan(&endTime)
	if err == sql.ErrNoRows {
		httpx.Error(w, http.StatusNotFound, "Contest not found")
		return
	} else if err != nil {
		h.logger.Error("Failed to check contest", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "Failed to register")
		return
	}

	if time.Now().After(endTime) {
		httpx.Error(w, http.StatusBadRequest, "Contest has ended")
		return
	}

	// Register user
	query := `INSERT INTO contest_participants (contest_id, user_id) VALUES ($1, $2)
	          ON CONFLICT (contest_id, user_id) DO NOTHING`

	_, err = h.dbManager.GetDB().Exec(query, contestID, userID)
	if err != nil {
		h.logger.Error("Failed to register for contest", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "Failed to register")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Successfully registered"})
}

// GetLeaderboard gets the leaderboard for a contest
func (h *ContestsHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	contestID, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "Invalid contest ID")
		return
	}

	// Check if leaderboard is frozen
	var startTime, endTime time.Time
	var freezeTime *int
	err = h.dbManager.GetDB().QueryRow(
		"SELECT start_time, end_time, freeze_time FROM contests WHERE id = $1", contestID).
		Scan(&startTime, &endTime, &freezeTime)

	if err != nil {
		h.logger.Error("Failed to get contest info", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "Failed to get leaderboard")
		return
	}

	// Calculate if we should show frozen leaderboard
	now := time.Now()
	isFrozen := false
	var cutoffTime time.Time
	if freezeTime != nil && *freezeTime > 0 && now.Before(endTime) {
		cutoffTime = endTime.Add(-time.Duration(*freezeTime) * time.Minute)
		isFrozen = now.After(cutoffTime)
	}

	// Build leaderboard query
	timeFilter := ""
	if isFrozen {
		timeFilter = " AND s.created_at < $2"
	}

	query := `
	WITH user_scores AS (
		SELECT 
			s.user_id,
			u.username,
			COALESCE(SUM(CASE WHEN s.verdict = 'Accepted' THEN s.points ELSE 0 END), 0) as total_score,
			COUNT(DISTINCT CASE WHEN s.verdict = 'Accepted' THEN s.problem_id END) as problems_solved,
			MAX(s.created_at) as last_submission_at,
			EXTRACT(EPOCH FROM (MAX(s.created_at) - $1))::INTEGER / 60 as penalty_minutes
		FROM submissions s
		JOIN users u ON s.user_id = u.id
		WHERE s.contest_id = ` + strconv.Itoa(contestID) + timeFilter + `
		GROUP BY s.user_id, u.username
	)
	SELECT 
		ROW_NUMBER() OVER (ORDER BY total_score DESC, problems_solved DESC, last_submission_at ASC) as rank,
		user_id, username, total_score, problems_solved, last_submission_at, penalty_minutes
	FROM user_scores
	ORDER BY total_score DESC, problems_solved DESC, last_submission_at ASC`

	var rows *sql.Rows
	if isFrozen {
		rows, err = h.dbManager.GetDB().Query(query, startTime, cutoffTime)
	} else {
		rows, err = h.dbManager.GetDB().Query(query, startTime)
	}

	if err != nil {
		h.logger.Error("Failed to get leaderboard", zap.Error(err))
		httpx.Error(w, http.StatusInternalServerError, "Failed to get leaderboard")
		return
	}
	defer rows.Close()

	leaderboard := []LeaderboardEntry{}
	for rows.Next() {
		var entry LeaderboardEntry
		if err := rows.Scan(&entry.Rank, &entry.UserID, &entry.Username, &entry.TotalScore,
			&entry.ProblemsSolved, &entry.LastSubmissionAt, &entry.PenaltyMinutes); err != nil {
			continue
		}
		leaderboard = append(leaderboard, entry)
	}

	response := map[string]interface{}{
		"leaderboard": leaderboard,
		"is_frozen":   isFrozen,
		"contest_id":  contestID,
	}

	httpx.JSON(w, http.StatusOK, response)
}
