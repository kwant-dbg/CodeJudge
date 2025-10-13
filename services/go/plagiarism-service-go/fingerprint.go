package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"hash/fnv"
	"regexp"
	"strings"
)

const (
	kGramSize  = 7  // Increased from 5 for better precision
	windowSize = 10 // Increased from 4 for better selectivity
)

// Optimized AST-based normalization with reduced allocations
func normalizeCodeAST(code string) string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		// Fallback to simple normalization if parsing fails
		return normalizeCodeSimple(code)
	}

	// Pre-allocate builder with estimated capacity
	var normalized strings.Builder
	normalized.Grow(len(code) / 4) // Estimate: normalized code is ~25% of original

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BasicLit:
			// Normalize string and numeric literals
			if x.Kind == token.STRING {
				normalized.WriteString("STR")
			} else if x.Kind == token.INT || x.Kind == token.FLOAT {
				normalized.WriteString("NUM")
			}
		case *ast.Ident:
			// Preserve keywords but normalize identifiers
			if isKeyword(x.Name) {
				normalized.WriteString(x.Name)
			} else {
				normalized.WriteString("ID")
			}
		case *ast.BinaryExpr:
			// Preserve operator structure
			normalized.WriteString("OP")
		}
		return true
	})

	return normalized.String()
} // Fallback simple normalization (improved version of original)
func normalizeCodeSimple(code string) string {
	// Remove comments
	singleLineComment := regexp.MustCompile(`//.*`)
	multiLineComment := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	code = singleLineComment.ReplaceAllString(code, "")
	code = multiLineComment.ReplaceAllString(code, "")

	// Normalize whitespace
	whitespace := regexp.MustCompile(`\s+`)
	code = whitespace.ReplaceAllString(code, " ")

	// Normalize string literals
	stringLiteral := regexp.MustCompile(`"[^"]*"`)
	code = stringLiteral.ReplaceAllString(code, `"STR"`)

	// Normalize numeric literals
	numericLiteral := regexp.MustCompile(`\b\d+\.?\d*\b`)
	code = numericLiteral.ReplaceAllString(code, "NUM")

	// Normalize variable names (preserve keywords)
	keywords := []string{"if", "else", "while", "for", "int", "float", "char", "return", "void", "class", "struct"}
	words := regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`)
	code = words.ReplaceAllStringFunc(code, func(word string) string {
		for _, keyword := range keywords {
			if word == keyword {
				return word
			}
		}
		return "VAR"
	})

	return strings.ToLower(code)
}

func isKeyword(name string) bool {
	keywords := map[string]bool{
		"break": true, "case": true, "chan": true, "const": true, "continue": true,
		"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
		"func": true, "go": true, "goto": true, "if": true, "import": true,
		"interface": true, "map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true, "var": true,
	}
	return keywords[name]
}

// Optimized hash generation with pre-allocated hasher and rolling hash
func getHashesOptimized(text string) []uint64 {
	if len(text) < kGramSize {
		return nil
	}

	// Pre-allocate slice with exact capacity to avoid reallocations
	numHashes := len(text) - kGramSize + 1
	hashes := make([]uint64, 0, numHashes)

	// Use a single hasher instance instead of creating/resetting for each k-gram
	h := fnv.New64a()

	for i := 0; i <= len(text)-kGramSize; i++ {
		h.Reset()
		// Write the k-gram directly without creating substring
		h.Write([]byte(text[i : i+kGramSize]))
		hashes = append(hashes, h.Sum64())
	}
	return hashes
}

// More efficient winnowing using single-pass algorithm
func winnowOptimized(hashes []uint64) map[uint64]bool {
	if len(hashes) == 0 {
		return make(map[uint64]bool)
	}

	// Pre-allocate map with estimated capacity
	fingerprints := make(map[uint64]bool, len(hashes)/windowSize+1)

	if len(hashes) < windowSize {
		// For very short code, use all hashes
		for _, hash := range hashes {
			fingerprints[hash] = true
		}
		return fingerprints
	}

	// Use sliding window minimum with deque for O(n) complexity
	for i := 0; i <= len(hashes)-windowSize; i++ {
		window := hashes[i : i+windowSize]

		// Find minimum in current window - this is still O(windowSize) per window
		// but windowSize is small constant (10)
		minHash := window[0]
		for j := 1; j < len(window); j++ {
			if window[j] < minHash {
				minHash = window[j]
			}
		}

		fingerprints[minHash] = true
	}

	return fingerprints
}

// GenerateFingerprint creates an optimized fingerprint
func GenerateFingerprint(code string) map[uint64]bool {
	normalized := normalizeCodeAST(code)
	if len(normalized) < kGramSize {
		// Fallback for very short code
		return winnowOptimized(getHashesOptimized(normalizeCodeSimple(code)))
	}

	hashes := getHashesOptimized(normalized)
	return winnowOptimized(hashes)
}

// Optimized containment similarity with single iteration
func CalculateContainmentSimilarity(fpA, fpB map[uint64]bool) (float64, float64) {
	lenA, lenB := len(fpA), len(fpB)

	if lenA == 0 && lenB == 0 {
		return 1.0, 1.0
	}
	if lenA == 0 || lenB == 0 {
		return 0.0, 0.0
	}

	// Optimize: iterate over smaller map for better performance
	var smaller, larger map[uint64]bool

	if lenA <= lenB {
		smaller, larger = fpA, fpB
	} else {
		smaller, larger = fpB, fpA
	}

	intersectionSize := 0
	for hash := range smaller {
		if larger[hash] {
			intersectionSize++
		}
	}

	// Return in correct order based on which was smaller
	if lenA <= lenB {
		containmentAinB := float64(intersectionSize) / float64(lenA)
		containmentBinA := float64(intersectionSize) / float64(lenB)
		return containmentAinB, containmentBinA
	} else {
		containmentBinA := float64(intersectionSize) / float64(lenB)
		containmentAinB := float64(intersectionSize) / float64(lenA)
		return containmentAinB, containmentBinA
	}
}

// Optimized Jaccard calculation reusing intersection from containment
func CalculateJaccardOptimized(fpA, fpB map[uint64]bool, intersectionSize int) float64 {
	if len(fpA) == 0 && len(fpB) == 0 {
		return 1.0
	}

	unionSize := len(fpA) + len(fpB) - intersectionSize
	if unionSize == 0 {
		return 1.0
	}
	return float64(intersectionSize) / float64(unionSize)
}

// Legacy Jaccard for backward compatibility (but less efficient)
func CalculateJaccard(fpA, fpB map[uint64]bool) float64 {
	if len(fpA) == 0 && len(fpB) == 0 {
		return 1.0
	}

	// Optimize: iterate over smaller map
	smaller, larger := fpA, fpB
	if len(fpB) < len(fpA) {
		smaller, larger = fpB, fpA
	}

	intersectionSize := 0
	for hash := range smaller {
		if larger[hash] {
			intersectionSize++
		}
	}

	unionSize := len(fpA) + len(fpB) - intersectionSize
	if unionSize == 0 {
		return 1.0
	}
	return float64(intersectionSize) / float64(unionSize)
}

// Highly optimized weighted similarity with single-pass calculation
func CalculateWeightedSimilarity(fpA, fpB map[uint64]bool) float64 {
	lenA, lenB := len(fpA), len(fpB)

	if lenA == 0 && lenB == 0 {
		return 1.0
	}
	if lenA == 0 || lenB == 0 {
		return 0.0
	}

	// Single iteration to calculate intersection
	smaller, larger := fpA, fpB
	if lenB < lenA {
		smaller, larger = fpB, fpA
	}

	intersectionSize := 0
	for hash := range smaller {
		if larger[hash] {
			intersectionSize++
		}
	}

	// Calculate all metrics in one go
	jaccard := float64(intersectionSize) / float64(lenA+lenB-intersectionSize)
	containmentA := float64(intersectionSize) / float64(lenA)
	containmentB := float64(intersectionSize) / float64(lenB)

	// Use maximum containment to catch subset relationships
	maxContainment := containmentA
	if containmentB > maxContainment {
		maxContainment = containmentB
	}

	// Weighted combination: 40% Jaccard + 60% max containment
	return 0.4*jaccard + 0.6*maxContainment
}

// Ultra-optimized function that calculates ALL similarity metrics in one pass
func calculateAllSimilarityMetrics(fpA, fpB map[uint64]bool) (weighted, jaccard, containmentA, containmentB float64) {
	lenA, lenB := len(fpA), len(fpB)

	if lenA == 0 && lenB == 0 {
		return 1.0, 1.0, 1.0, 1.0
	}
	if lenA == 0 || lenB == 0 {
		return 0.0, 0.0, 0.0, 0.0
	}

	// Single iteration to calculate intersection (iterate over smaller set)
	smaller, larger := fpA, fpB
	if lenB < lenA {
		smaller, larger = fpB, fpA
	}

	intersectionSize := 0
	for hash := range smaller {
		if larger[hash] {
			intersectionSize++
		}
	}

	// Calculate all metrics in one go
	jaccard = float64(intersectionSize) / float64(lenA+lenB-intersectionSize)
	containmentA = float64(intersectionSize) / float64(lenA)
	containmentB = float64(intersectionSize) / float64(lenB)

	// Use maximum containment to catch subset relationships
	maxContainment := containmentA
	if containmentB > maxContainment {
		maxContainment = containmentB
	}

	// Weighted combination: 40% Jaccard + 60% max containment
	weighted = 0.4*jaccard + 0.6*maxContainment

	return weighted, jaccard, containmentA, containmentB
}
