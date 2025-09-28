package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
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

var db *sql.DB

func connectDB() {
	databaseURL := os.Getenv("DATABASE_URL")
	var err error
	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", databaseURL)
		if err == nil {
			if err = db.Ping(); err == nil {
				log.Println("Successfully connected to the database.")
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("Failed to connect to the database: %v", err)
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
		log.Fatalf("Failed to create 'problems' table: %v", err)
	}
	log.Println("'problems' table is ready.")

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
		log.Fatalf("Failed to create 'test_cases' table: %v", err)
	}
	log.Println("'test_cases' table is ready.")
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

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	problemID, err := strconv.Atoi(parts[2])
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
		log.Printf("Error creating test case: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tc)
}

func main() {
	connectDB()
	createTable()
	defer db.Close()

	http.HandleFunc("/problems/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/testcases") {
			testCaseHandler(w, r)
		} else {
			problemsHandler(w, r)
		}
	})

	log.Println("Problems Service starting on port 8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
