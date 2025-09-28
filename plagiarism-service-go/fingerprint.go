package main

import (
	"hash/fnv"
	"regexp"
	"strings"
)

const (
	kGramSize  = 5
	windowSize = 4
)

func normalizeCode(code string) string {
	re := regexp.MustCompile(`\s+`)
	code = re.ReplaceAllString(code, "")
	nonAlphaNum := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	code = nonAlphaNum.ReplaceAllString(code, "")
	return strings.ToLower(code)
}

func getHashes(text string) []uint32 {
	var hashes []uint32
	h := fnv.New32a()
	for i := 0; i <= len(text)-kGramSize; i++ {
		gram := text[i : i+kGramSize]
		h.Reset()
		h.Write([]byte(gram))
		hashes = append(hashes, h.Sum32())
	}
	return hashes
}

func winnow(hashes []uint32) map[uint32]bool {
	fingerprints := make(map[uint32]bool)
	if len(hashes) == 0 {
		return fingerprints
	}

	for i := 0; i <= len(hashes)-windowSize; i++ {
		window := hashes[i : i+windowSize]
		minHash := window[0]
		for _, hash := range window {
			if hash < minHash {
				minHash = hash
			}
		}
		fingerprints[minHash] = true
	}
	return fingerprints
}

func GenerateFingerprint(code string) map[uint32]bool {
	normalized := normalizeCode(code)
	hashes := getHashes(normalized)
	return winnow(hashes)
}

func CalculateJaccard(fpA, fpB map[uint32]bool) float64 {
	intersectionSize := 0
	for hash := range fpA {
		if fpB[hash] {
			intersectionSize++
		}
	}

	unionSize := len(fpA) + len(fpB) - intersectionSize
	if unionSize == 0 {
		return 1.0
	}
	return float64(intersectionSize) / float64(unionSize)
}
