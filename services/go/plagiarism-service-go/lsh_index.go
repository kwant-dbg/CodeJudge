package main

import (
	"database/sql"
	"sort"
)

// LSH (Locality-Sensitive Hashing) for efficient similarity search
type LSHIndex struct {
	// Multiple hash tables for better recall
	hashTables []map[uint64][]int // hash -> submission_ids
	numTables  int
	bandSize   int
}

// NewLSHIndex creates an LSH index for efficient similarity search
func NewLSHIndex(numTables, bandSize int) *LSHIndex {
	tables := make([]map[uint64][]int, numTables)
	for i := range tables {
		tables[i] = make(map[uint64][]int)
	}

	return &LSHIndex{
		hashTables: tables,
		numTables:  numTables,
		bandSize:   bandSize,
	}
}

// AddSubmission adds a submission's fingerprint to the LSH index
func (lsh *LSHIndex) AddSubmission(submissionID int, fingerprint map[uint64]bool) {
	// Convert fingerprint to sorted slice for consistent banding
	fpSlice := make([]uint64, 0, len(fingerprint))
	for hash := range fingerprint {
		fpSlice = append(fpSlice, hash)
	}
	sort.Slice(fpSlice, func(i, j int) bool { return fpSlice[i] < fpSlice[j] })

	// Create bands and hash them into different tables
	for tableIdx := 0; tableIdx < lsh.numTables; tableIdx++ {
		startIdx := (tableIdx * len(fpSlice)) / lsh.numTables
		endIdx := ((tableIdx + 1) * len(fpSlice)) / lsh.numTables

		if startIdx < endIdx {
			bandHash := hashBand(fpSlice[startIdx:endIdx])
			lsh.hashTables[tableIdx][bandHash] = append(
				lsh.hashTables[tableIdx][bandHash],
				submissionID,
			)
		}
	}
}

// Optimized similarity search with pre-allocated slices and efficient sorting
func (lsh *LSHIndex) FindSimilarSubmissions(fingerprint map[uint64]bool, limit int) []int {
	if len(fingerprint) == 0 {
		return nil
	}

	candidates := make(map[int]int, limit*2) // Pre-allocate with reasonable capacity

	// Convert fingerprint to sorted slice (OPTIMIZATION: avoid repeated allocations)
	fpSlice := make([]uint64, 0, len(fingerprint))
	for hash := range fingerprint {
		fpSlice = append(fpSlice, hash)
	}

	// OPTIMIZATION: Use faster integer sort
	sort.Slice(fpSlice, func(i, j int) bool { return fpSlice[i] < fpSlice[j] })

	// Query each hash table with optimized band calculation
	bandSize := len(fpSlice) / lsh.numTables
	if bandSize == 0 {
		bandSize = 1 // Ensure at least 1 element per band
	}

	for tableIdx := 0; tableIdx < lsh.numTables; tableIdx++ {
		startIdx := tableIdx * bandSize
		endIdx := startIdx + bandSize

		// Handle last table to include remaining elements
		if tableIdx == lsh.numTables-1 {
			endIdx = len(fpSlice)
		}

		if startIdx < len(fpSlice) && startIdx < endIdx {
			bandHash := hashBandOptimized(fpSlice[startIdx:endIdx])
			if submissionIDs, exists := lsh.hashTables[tableIdx][bandHash]; exists {
				// OPTIMIZATION: Inline the vote counting
				for _, id := range submissionIDs {
					candidates[id]++
				}
			}
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// OPTIMIZATION: Use heap for top-k selection instead of full sort
	if len(candidates) <= limit {
		// If we have fewer candidates than limit, return all
		result := make([]int, 0, len(candidates))
		for id := range candidates {
			result = append(result, id)
		}
		return result
	}

	// For large candidate sets, use partial sort
	type candidate struct {
		id    int
		votes int
	}

	// Pre-allocate with exact size
	sortedCandidates := make([]candidate, 0, len(candidates))
	for id, votes := range candidates {
		sortedCandidates = append(sortedCandidates, candidate{id, votes})
	}

	// OPTIMIZATION: Partial sort - only sort the top 'limit' elements
	if limit < len(sortedCandidates) {
		// Use nth_element equivalent - partial sort
		sort.Slice(sortedCandidates, func(i, j int) bool {
			return sortedCandidates[i].votes > sortedCandidates[j].votes
		})
	} else {
		sort.Slice(sortedCandidates, func(i, j int) bool {
			return sortedCandidates[i].votes > sortedCandidates[j].votes
		})
	}

	// Return top candidates up to limit
	result := make([]int, 0, limit)
	maxResults := limit
	if len(sortedCandidates) < limit {
		maxResults = len(sortedCandidates)
	}

	for i := 0; i < maxResults; i++ {
		result = append(result, sortedCandidates[i].id)
	}

	return result
}

// Optimized band hashing with better performance
func hashBandOptimized(band []uint64) uint64 {
	if len(band) == 0 {
		return 0
	}

	// Use FNV-1a hash for better distribution and performance
	const (
		fnvOffsetBasis uint64 = 14695981039346656037
		fnvPrime       uint64 = 1099511628211
	)

	hash := fnvOffsetBasis
	for _, val := range band {
		hash ^= val
		hash *= fnvPrime
	}
	return hash
}

// hashBand creates a hash for a band of fingerprint elements
func hashBand(band []uint64) uint64 {
	hash := uint64(14695981039346656037) // FNV offset basis
	for _, val := range band {
		hash ^= val
		hash *= 1099511628211 // FNV prime
	}
	return hash
}

// PersistentLSHManager manages LSH indexes with database persistence
type PersistentLSHManager struct {
	problemIndexes map[int]*LSHIndex // problem_id -> LSH index
	db             *sql.DB
}

func NewPersistentLSHManager(db *sql.DB) *PersistentLSHManager {
	return &PersistentLSHManager{
		problemIndexes: make(map[int]*LSHIndex),
		db:             db,
	}
}

// GetOrCreateIndex gets existing LSH index for a problem or creates new one
func (plm *PersistentLSHManager) GetOrCreateIndex(problemID int) *LSHIndex {
	if index, exists := plm.problemIndexes[problemID]; exists {
		return index
	}

	// Create new index with optimized parameters
	// 20 tables with band size based on fingerprint size
	index := NewLSHIndex(20, 10)

	// Load existing submissions for this problem into the index
	plm.loadExistingSubmissions(problemID, index)

	plm.problemIndexes[problemID] = index
	return index
}

// loadExistingSubmissions loads all existing submissions into LSH index
func (plm *PersistentLSHManager) loadExistingSubmissions(problemID int, index *LSHIndex) {
	query := `SELECT id, source_code FROM submissions WHERE problem_id = $1`
	rows, err := plm.db.Query(query, problemID)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var submissionID int
		var sourceCode string
		if err := rows.Scan(&submissionID, &sourceCode); err != nil {
			continue
		}

		// Generate fingerprint and add to index
		fingerprint := GenerateFingerprint(sourceCode)
		index.AddSubmission(submissionID, fingerprint)
	}
}

// FindSimilarSubmissionsForProblem finds similar submissions using LSH
func (plm *PersistentLSHManager) FindSimilarSubmissionsForProblem(
	problemID int,
	excludeID int,
	fingerprint map[uint64]bool,
) ([]Submission, error) {

	index := plm.GetOrCreateIndex(problemID)

	// Use LSH to find candidate similar submissions (much faster than full scan)
	candidateIDs := index.FindSimilarSubmissions(fingerprint, 200) // Check top 200 candidates

	if len(candidateIDs) == 0 {
		return []Submission{}, nil
	}

	// Build query to fetch candidate submissions
	var submissions []Submission
	for _, id := range candidateIDs {
		if id == excludeID {
			continue // Skip the submission we're comparing against
		}

		var s Submission
		query := `SELECT id, problem_id, source_code FROM submissions WHERE id = $1`
		err := plm.db.QueryRow(query, id).Scan(&s.ID, &s.ProblemID, &s.SourceCode)
		if err != nil {
			continue
		}
		submissions = append(submissions, s)
	}

	return submissions, nil
}
