package channel

import (
	"math/rand"
	"time"
)

// Backoff computes exponentially increasing delays with optional jitter.
// It is not safe for concurrent use; each supervised goroutine should
// create its own instance.
type Backoff struct {
	// Initial is the first delay duration.
	Initial time.Duration
	// Max caps the delay.
	Max time.Duration
	// Factor is the multiplier applied after each failure (e.g. 2.0).
	Factor float64
	// Jitter is the fractional random variation added to each delay (e.g. 0.1 → ±10%).
	Jitter float64

	current time.Duration
}

// defaultBackoff returns a Backoff suitable for channel reconnection.
func defaultBackoff() Backoff {
	return Backoff{
		Initial: 5 * time.Second,
		Max:     5 * time.Minute,
		Factor:  2.0,
		Jitter:  0.1,
	}
}

// Next returns the next delay and advances the internal state.
func (b *Backoff) Next() time.Duration {
	if b.current == 0 {
		b.current = b.Initial
	}

	d := b.current

	// Apply jitter: ±Jitter fraction
	if b.Jitter > 0 {
		delta := float64(d) * b.Jitter
		d += time.Duration((rand.Float64()*2 - 1) * delta) //nolint:gosec
	}

	// Advance for next call
	next := time.Duration(float64(b.current) * b.Factor)
	if next > b.Max {
		next = b.Max
	}
	b.current = next

	return d
}

// Reset resets the backoff to the initial delay.
func (b *Backoff) Reset() {
	b.current = 0
}
