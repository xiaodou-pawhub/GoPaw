package feishu

import (
	"sync"
	"time"
)

// Deduper keeps track of recently processed message IDs to prevent duplicates.
// It uses a simple map with a timestamp for expiration (TTL).
type Deduper struct {
	mu    sync.Mutex
	seen  map[string]time.Time
	ttl   time.Duration
}

func NewDeduper(ttl time.Duration) *Deduper {
	d := &Deduper{
		seen: make(map[string]time.Time),
		ttl:  ttl,
	}
	go d.runJanitor()
	return d
}

// Seen checks if an ID has been seen before. If not, it marks it as seen and returns false.
func (d *Deduper) Seen(id string) bool {
	if id == "" {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.seen[id]; exists {
		return true
	}

	d.seen[id] = time.Now()
	return false
}

func (d *Deduper) runJanitor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.Lock()
		now := time.Now()
		for id, t := range d.seen {
			if now.Sub(t) > d.ttl {
				delete(d.seen, id)
			}
		}
		d.mu.Unlock()
	}
}
