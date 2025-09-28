package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

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

func connectRedis() {
	redisURL := os.Getenv("REDIS_URL")
	rdb = redis.NewClient(&redis.Options{Addr: redisURL})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis.")
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
		log.Fatalf("Failed to create 'plagiarism_reports' table: %v", err)
	}
	log.Println("'plagiarism_reports' table is ready.")
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
	log.Println("Plagiarism worker started...")
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
				log.Printf("High similarity (%.2f) between submission %d and %d", similarity, newSub.ID, otherSub.ID)
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
	connectDB()
	connectRedis()
	createTable()
	defer db.Close()

	go worker()

	http.HandleFunc("/plagiarism/reports", reportsHandler)
	log.Println("Plagiarism Service starting on port 8002")
	if err := http.ListenAndServe(":8002", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
