package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"codejudge/common/dbutil"

	"github.com/dgryski/go-farm"
	"github.com/dgryski/go-minhash"
	minhashlsh "github.com/ekzhu/minhash-lsh"
	"github.com/go-redis/redis/v8"
	"github.com/goplus/llcppg/ast"
	"github.com/goplus/llcppg/parser"
	"go.uber.org/zap"
	"go/token"
)

type PlagiarismSubmission struct {
	ID         int
	ProblemID  int
	SourceCode string
	Language   string
}

type Report struct {
	ID          int       `json:"id"`
	SubmissionA int       `json:"submission_a"`
	SubmissionB int       `json:"submission_b"`
	Similarity  float64   `json:"similarity"`
	CreatedAt   time.Time `json:"created_at"`
}

type PlagiarismHandler struct {
	logger    *zap.Logger
	dbManager *dbutil.ConnectionManager
	rdb       *redis.Client
	ctx       context.Context
}

func NewPlagiarismHandler(logger *zap.Logger, dbManager *dbutil.ConnectionManager, rdb *redis.Client) *PlagiarismHandler {
	return &PlagiarismHandler{
		logger:    logger,
		dbManager: dbManager,
		rdb:       rdb,
		ctx:       context.Background(),
	}
}

func (h *PlagiarismHandler) CreateTables() {
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
	if _, err := h.dbManager.GetDB().Exec(createTableSQL); err != nil {
		h.logger.Fatal("Failed to create 'plagiarism_reports' table", zap.Error(err))
	}
	h.logger.Info("'plagiarism_reports' table is ready")
}

func walk(node ast.Node, f func(ast.Node)) {
	if node == nil {
		return
	}
	f(node)

	switch n := node.(type) {
	case *ast.File:
		for _, decl := range n.Decls {
			walk(decl, f)
		}
	}
}

func (h *PlagiarismHandler) getAST(source string) ([]string, error) {
	fset := token.NewFileSet()
	tmpfile, err := os.CreateTemp("", "example.cpp")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(source); err != nil {
		return nil, err
	}
	if err := tmpfile.Close(); err != nil {
		return nil, err
	}

	astFile, err := parser.ParseFile(fset, tmpfile.Name(), "", nil)
	if err != nil {
		return nil, err
	}

	var nodes []string
	walk(astFile, func(n ast.Node) {
		if n != nil {
			nodes = append(nodes, fmt.Sprintf("%T", n))
		}
	})
	return nodes, nil
}

func getShingles(nodes []string, k int) [][]string {
	if len(nodes) < k {
		return [][]string{nodes}
	}
	shingles := make([][]string, len(nodes)-k+1)
	for i := 0; i < len(nodes)-k+1; i++ {
		shingles[i] = nodes[i : i+k]
	}
	return shingles
}

func (h *PlagiarismHandler) startWorker() {
	h.logger.Info("Plagiarism worker started")
	go func() {
		for {
			result, err := h.rdb.BLPop(h.ctx, 0, "plagiarism_queue").Result()
			if err != nil {
				h.logger.Error("Redis BLPop error", zap.Error(err))
				time.Sleep(time.Second)
				continue
			}

			submissionID, err := strconv.Atoi(result[1])
			if err != nil {
				h.logger.Error("Invalid submission ID", zap.String("id", result[1]))
				continue
			}

			h.logger.Info("Processing plagiarism check", zap.Int("submission_id", submissionID))

			var currentSubmission PlagiarismSubmission
			err = h.dbManager.GetDB().QueryRow("SELECT id, problem_id, source_code, language FROM submissions WHERE id = $1", submissionID).Scan(&currentSubmission.ID, &currentSubmission.ProblemID, &currentSubmission.SourceCode, &currentSubmission.Language)
			if err != nil {
				h.logger.Error("Failed to get submission", zap.Error(err))
				continue
			}

			if currentSubmission.Language != "C++" {
				h.logger.Info("Skipping plagiarism check for non-C++ submission", zap.Int("submission_id", submissionID))
				continue
			}

			// 1. Get all submissions for the same problem
			rows, err := h.dbManager.GetDB().Query("SELECT id, source_code FROM submissions WHERE problem_id = $1 AND language = 'C++'", currentSubmission.ProblemID)
			if err != nil {
				h.logger.Error("Failed to get other submissions", zap.Error(err))
				continue
			}
			defer rows.Close()

			// 2. Create a map of submission ID to shingles
			submissionShingles := make(map[int][][]string)
			for rows.Next() {
				var submission PlagiarismSubmission
				if err := rows.Scan(&submission.ID, &submission.SourceCode); err != nil {
					h.logger.Error("Failed to scan other submission", zap.Error(err))
					continue
				}
				nodes, err := h.getAST(submission.SourceCode)
				if err != nil {
					h.logger.Error("Failed to get AST", zap.Error(err), zap.Int("submission_id", submission.ID))
					continue
				}
				submissionShingles[submission.ID] = getShingles(nodes, 5)
			}

			// 3. Create MinHash for each submission
			minhashes := make(map[int]*minhash.MinWise)
			for id, shingles := range submissionShingles {
				h1 := farm.Hash64
				h2 := farm.Hash64
				mw := minhash.NewMinWise(h1, h2, 128)
				for _, shingle := range shingles {
					mw.Push([]byte(strings.Join(shingle, "")))
				}
				minhashes[id] = mw
			}

			// 4. Use LSH to find candidate pairs
			l := minhashlsh.NewMinhashLSH(128, 0.8, 1)

			for id, mh := range minhashes {
				l.Add(fmt.Sprintf("%d", id), mh.Signature())
			}

			// 5. For each submission, query for candidates
			for id, mh := range minhashes {
				candidates := l.Query(mh.Signature())
				for _, candidate := range candidates {
					candidateID, _ := strconv.Atoi(candidate.(string))
					if id >= candidateID {
						continue
					}

					// 6. Calculate Jaccard similarity for candidate pairs
					jaccard := minhashes[id].Similarity(minhashes[candidateID])

					if jaccard > 0.8 { // Threshold
						_, err := h.dbManager.GetDB().Exec("INSERT INTO plagiarism_reports (submission_a, submission_b, similarity) VALUES ($1, $2, $3) ON CONFLICT (submission_a, submission_b) DO NOTHING", id, candidateID, jaccard)
						if err != nil {
							h.logger.Error("Failed to insert plagiarism report", zap.Error(err))
						}
					}
				}
			}
		}
	}()
}

func (h *PlagiarismHandler) StartWorker() {
	h.startWorker()
}

func (h *PlagiarismHandler) GetReports(w http.ResponseWriter, r *http.Request) {
	rows, err := h.dbManager.GetDB().Query("SELECT id, submission_a, submission_b, similarity, created_at FROM plagiarism_reports ORDER BY similarity DESC")
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
