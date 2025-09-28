package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

type Submission struct {
	ID         int    `json:"id"`
	ProblemID  int    `json:"problem_id"`
	SourceCode string `json:"source_code"`
}

var db *sql.DB
var rdb *redis.Client
var ctx = context.Background()

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
    CREATE TABLE IF NOT EXISTS submissions (
        id SERIAL PRIMARY KEY,
        problem_id INTEGER NOT NULL,
        source_code TEXT NOT NULL,
        verdict VARCHAR(50) DEFAULT 'Pending',
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );`
	if _, err := db.Exec(createTableSQL); err != nil {
		log.Fatalf("Failed to create 'submissions' table: %v", err)
	}
	log.Println("'submissions' table is ready.")
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
	connectDB()
	connectRedis()
	createTable()
	defer db.Close()

	http.HandleFunc("/submissions", submissionHandler)
	log.Println("Submissions Service starting on port 8001")
	if err := http.ListenAndServe(":8001", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
