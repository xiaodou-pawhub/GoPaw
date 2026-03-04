package channel

import "sync"

// SessionLocker provides per-session mutual exclusion.
// It ensures that messages belonging to the same session (conversation) are
// processed one at a time, preventing memory/session state races.
//
// Entries are reference-counted and automatically removed when no goroutines
// are waiting or holding the lock, preventing unbounded memory growth.
type SessionLocker struct {
	mu   sync.Mutex
	sess map[string]*sessionEntry
}

type sessionEntry struct {
	mu      sync.Mutex
	waiters int
}

// NewSessionLocker creates an empty SessionLocker.
func NewSessionLocker() *SessionLocker {
	return &SessionLocker{sess: make(map[string]*sessionEntry)}
}

// Lock acquires the per-session lock for key and returns an unlock function.
// Callers must invoke the returned function exactly once when done.
//
//	unlock := locker.Lock(sessionID)
//	defer unlock()
func (sl *SessionLocker) Lock(key string) (unlock func()) {
	sl.mu.Lock()
	e, ok := sl.sess[key]
	if !ok {
		e = &sessionEntry{}
		sl.sess[key] = e
	}
	e.waiters++
	sl.mu.Unlock()

	e.mu.Lock()

	return func() {
		e.mu.Unlock()

		sl.mu.Lock()
		e.waiters--
		if e.waiters == 0 {
			delete(sl.sess, key)
		}
		sl.mu.Unlock()
	}
}
