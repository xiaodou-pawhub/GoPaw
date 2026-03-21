// Package embedding provides local text embedding using TF-IDF.
// This implementation is zero-dependency and doesn't require external models.
package embedding

// Encoder provides text embedding functionality.
type Encoder interface {
	// Encode converts text to a vector embedding.
	Encode(text string) ([]float32, error)

	// Dimension returns the embedding dimension.
	Dimension() int

	// Close releases resources.
	Close() error
}

// Global encoder instance
var defaultEncoder Encoder

// GetDefaultEncoder returns the default encoder instance (384 dimensions).
func GetDefaultEncoder() Encoder {
	if defaultEncoder == nil {
		defaultEncoder = NewSimpleEncoder(384)
	}
	return defaultEncoder
}
