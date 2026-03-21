package embedding

import (
	"math"
	"strings"
	"sync"
)

// SimpleEncoder implements a TF-IDF based encoder.
// This is a lightweight implementation that doesn't require external models.
type SimpleEncoder struct {
	vocab     map[string]int
	idf       map[string]float64
	dimension int
	mu        sync.RWMutex
}

// NewSimpleEncoder creates a new simple encoder with the given vocabulary size.
func NewSimpleEncoder(vocabSize int) *SimpleEncoder {
	return &SimpleEncoder{
		vocab:     make(map[string]int),
		idf:       make(map[string]float64),
		dimension: vocabSize,
	}
}

// Encode converts text to a TF-IDF vector.
func (e *SimpleEncoder) Encode(text string) ([]float32, error) {
	// Tokenize
	tokens := e.tokenize(text)

	// Calculate term frequency
	tf := make(map[string]float64)
	for _, token := range tokens {
		tf[token]++
	}

	// Normalize TF
	maxFreq := 0.0
	for _, freq := range tf {
		if freq > maxFreq {
			maxFreq = freq
		}
	}
	if maxFreq > 0 {
		for token := range tf {
			tf[token] /= maxFreq
		}
	}

	// Build vector
	vector := make([]float32, e.dimension)

	// Use hash-based vocabulary mapping
	for token, freq := range tf {
		idx := e.hashToken(token) % e.dimension
		idf := e.getIDF(token)
		vector[idx] += float32(freq * idf)
	}

	// Normalize vector
	e.normalize(vector)

	return vector, nil
}

// Dimension returns the embedding dimension.
func (e *SimpleEncoder) Dimension() int {
	return e.dimension
}

// Close releases resources (no-op for SimpleEncoder).
func (e *SimpleEncoder) Close() error {
	return nil
}

// tokenize splits text into tokens.
func (e *SimpleEncoder) tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Split by non-alphanumeric characters
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127 {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// hashToken creates a hash for a token.
func (e *SimpleEncoder) hashToken(token string) int {
	h := 0
	for _, c := range token {
		h = 31*h + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

// getIDF returns the IDF value for a token.
func (e *SimpleEncoder) getIDF(token string) float64 {
	e.mu.RLock()
	idf, ok := e.idf[token]
	e.mu.RUnlock()

	if ok {
		return idf
	}

	// Default IDF
	return 1.0
}

// UpdateIDF updates IDF values based on a corpus.
func (e *SimpleEncoder) UpdateIDF(documents []string) {
	docCount := make(map[string]int)
	totalDocs := len(documents)

	for _, doc := range documents {
		tokens := e.tokenize(doc)
		seen := make(map[string]bool)
		for _, token := range tokens {
			if !seen[token] {
				docCount[token]++
				seen[token] = true
			}
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for token, count := range docCount {
		// IDF = log(N / df)
		e.idf[token] = math.Log(float64(totalDocs) / float64(count))
	}
}

// normalize normalizes a vector to unit length.
func (e *SimpleEncoder) normalize(vector []float32) {
	norm := 0.0
	for _, v := range vector {
		norm += float64(v * v)
	}

	if norm > 0 {
		norm = math.Sqrt(norm)
		invNorm := 1.0 / norm
		for i := range vector {
			vector[i] *= float32(invNorm)
		}
	}
}

// CosineSimilarity calculates the cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}
