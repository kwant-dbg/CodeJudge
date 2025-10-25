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
	"codejudge/common/env"

	"go/token"

	"github.com/dgryski/go-farm"
	"github.com/dgryski/go-minhash"
	minhashlsh "github.com/ekzhu/minhash-lsh"
	"github.com/go-redis/redis/v8"
	"github.com/goplus/llcppg/ast"
	"github.com/goplus/llcppg/parser"
	"go.uber.org/zap"
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

	// Add minhash_signature column to submissions table if it doesn't exist
	addSignatureColumnSQL := `
    ALTER TABLE submissions 
    ADD COLUMN IF NOT EXISTS minhash_signature BYTEA;
    
    CREATE INDEX IF NOT EXISTS idx_submissions_problem_signature 
    ON submissions(problem_id) WHERE minhash_signature IS NOT NULL;`
	if _, err := h.dbManager.GetDB().Exec(addSignatureColumnSQL); err != nil {
		h.logger.Error("Failed to add minhash_signature column", zap.Error(err))
	}
	h.logger.Info("MinHash signature storage is ready")
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

// ComputeMinHashSignature generates a MinHash signature for source code
func (h *PlagiarismHandler) ComputeMinHashSignature(sourceCode string) ([]byte, error) {
	// Get AST nodes
	nodes, err := h.getAST(sourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AST: %w", err)
	}

	// Generate shingles
	shingles := getShingles(nodes, 5)

	// Create MinHash
	h1 := farm.Hash64
	h2 := farm.Hash64
	mw := minhash.NewMinWise(h1, h2, 128)
	for _, shingle := range shingles {
		mw.Push([]byte(strings.Join(shingle, "")))
	}

	// Serialize signature
	signature := mw.Signature()
	serialized := make([]byte, len(signature)*8)
	for i, val := range signature {
		// Convert uint64 to bytes (little-endian)
		for j := 0; j < 8; j++ {
			serialized[i*8+j] = byte(val >> (j * 8))
		}
	}

	return serialized, nil
}

// DeserializeSignature converts bytes back to uint64 slice
func DeserializeSignature(data []byte) []uint64 {
	if len(data)%8 != 0 {
		return nil
	}
	signature := make([]uint64, len(data)/8)
	for i := 0; i < len(signature); i++ {
		var val uint64
		for j := 0; j < 8; j++ {
			val |= uint64(data[i*8+j]) << (j * 8)
		}
		signature[i] = val
	}
	return signature
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
			h.processSubmission(submissionID)
		}
	}()
}

func (h *PlagiarismHandler) processSubmission(submissionID int) {
	// Get current submission
	var currentSubmission PlagiarismSubmission
	var currentSignature []byte
	err := h.dbManager.GetDB().QueryRow(
		"SELECT id, problem_id, source_code, language, minhash_signature FROM submissions WHERE id = $1",
		submissionID,
	).Scan(&currentSubmission.ID, &currentSubmission.ProblemID, &currentSubmission.SourceCode, &currentSubmission.Language, &currentSignature)

	if err != nil {
		h.logger.Error("Failed to get submission", zap.Error(err))
		return
	}

	if currentSubmission.Language != "C++" {
		h.logger.Info("Skipping plagiarism check for non-C++ submission", zap.Int("submission_id", submissionID))
		return
	}

	// Compute signature if not exists
	if currentSignature == nil {
		h.logger.Info("Computing MinHash signature for submission", zap.Int("submission_id", submissionID))
		sig, err := h.ComputeMinHashSignature(currentSubmission.SourceCode)
		if err != nil {
			h.logger.Error("Failed to compute signature", zap.Error(err), zap.Int("submission_id", submissionID))
			return
		}
		currentSignature = sig

		// Store the signature
		_, err = h.dbManager.GetDB().Exec(
			"UPDATE submissions SET minhash_signature = $1 WHERE id = $2",
			currentSignature, submissionID,
		)
		if err != nil {
			h.logger.Error("Failed to store signature", zap.Error(err))
		}
	}

	// Get max comparisons from environment (default 1000)
	maxComparisons := 1000
	if maxStr := env.Get("PLAGIARISM_MAX_COMPARISONS", "1000"); maxStr != "" {
		if val, err := strconv.Atoi(maxStr); err == nil && val > 0 {
			maxComparisons = val
		}
	}

	// Get similarity threshold from environment (default 0.8)
	similarityThreshold := 0.8
	if thresholdStr := env.Get("PLAGIARISM_SIMILARITY_THRESHOLD", "0.8"); thresholdStr != "" {
		if val, err := strconv.ParseFloat(thresholdStr, 64); err == nil && val > 0 && val <= 1.0 {
			similarityThreshold = val
		}
	}

	// Get other submissions with pre-computed signatures
	// Limit to recent submissions to avoid memory issues
	rows, err := h.dbManager.GetDB().Query(`
		SELECT id, minhash_signature 
		FROM submissions 
		WHERE problem_id = $1 
		  AND id != $2
		  AND language = 'C++'
		  AND minhash_signature IS NOT NULL
		ORDER BY created_at DESC
		LIMIT $3`,
		currentSubmission.ProblemID,
		submissionID,
		maxComparisons,
	)
	if err != nil {
		h.logger.Error("Failed to get other submissions", zap.Error(err))
		return
	}
	defer rows.Close()

	// Load signatures from database
	type SignatureData struct {
		ID        int
		Signature []uint64
	}
	signatures := make([]SignatureData, 0, maxComparisons)

	for rows.Next() {
		var id int
		var sigBytes []byte
		if err := rows.Scan(&id, &sigBytes); err != nil {
			h.logger.Error("Failed to scan signature", zap.Error(err))
			continue
		}

		sig := DeserializeSignature(sigBytes)
		if sig != nil {
			signatures = append(signatures, SignatureData{ID: id, Signature: sig})
		}
	}
	rows.Close() // Close early to free resources

	h.logger.Info("Loaded signatures for comparison",
		zap.Int("submission_id", submissionID),
		zap.Int("comparison_count", len(signatures)))

	// Create MinHash from current signature
	currentSig := DeserializeSignature(currentSignature)
	if currentSig == nil {
		h.logger.Error("Failed to deserialize current signature")
		return
	}

	// Use LSH to find candidate pairs
	lsh := minhashlsh.NewMinhashLSH(128, 0.8, 1)
	lsh.Add(fmt.Sprintf("%d", submissionID), currentSig)

	// Add all other signatures to LSH
	signaturesMap := make(map[int][]uint64)
	for _, sigData := range signatures {
		signaturesMap[sigData.ID] = sigData.Signature
		lsh.Add(fmt.Sprintf("%d", sigData.ID), sigData.Signature)
	}

	// Query for candidates
	candidates := lsh.Query(currentSig)
	h.logger.Info("Found candidates",
		zap.Int("submission_id", submissionID),
		zap.Int("candidate_count", len(candidates)))

	// Calculate similarity for candidates
	for _, candidate := range candidates {
		candidateID, err := strconv.Atoi(candidate.(string))
		if err != nil || candidateID == submissionID {
			continue
		}

		candidateSig, exists := signaturesMap[candidateID]
		if !exists {
			continue
		}

		// Calculate Jaccard similarity using signature comparison
		jaccard := calculateJaccardSimilarity(currentSig, candidateSig)
		h.logger.Info("Calculated similarity",
			zap.Int("submission_a", submissionID),
			zap.Int("submission_b", candidateID),
			zap.Float64("jaccard_similarity", jaccard))

		if jaccard > similarityThreshold {
			// Ensure smaller ID is first for consistency
			idA, idB := submissionID, candidateID
			if idA > idB {
				idA, idB = idB, idA
			}

			_, err := h.dbManager.GetDB().Exec(`
				INSERT INTO plagiarism_reports (submission_a, submission_b, similarity) 
				VALUES ($1, $2, $3) 
				ON CONFLICT (submission_a, submission_b) DO NOTHING`,
				idA, idB, jaccard)
			if err != nil {
				h.logger.Error("Failed to insert plagiarism report", zap.Error(err))
			}
		}
	}
}

// calculateJaccardSimilarity estimates Jaccard similarity from MinHash signatures
func calculateJaccardSimilarity(sig1, sig2 []uint64) float64 {
	if len(sig1) != len(sig2) {
		return 0.0
	}
	matches := 0
	for i := range sig1 {
		if sig1[i] == sig2[i] {
			matches++
		}
	}
	return float64(matches) / float64(len(sig1))
}

func (h *PlagiarismHandler) StartWorker() {
	h.startWorker()
}

func (h *PlagiarismHandler) GetReports(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	userRole, ok := r.Context().Value("role").(string)
	if !ok || userRole != "admin" {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

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
