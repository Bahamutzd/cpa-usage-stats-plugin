// Package cache provides a simple TTL-based in-memory cache for dashboard
// and monitoring responses. This avoids re-running 6-9 separate SQLite
// aggregation queries on every 30-second auto-refresh cycle.
package cache

import (
	"sync"
	"time"
)

type entry struct {
	value   any
	expires int64
}

// Store holds TTL-cached values keyed by string.
type Store struct {
	mu    sync.RWMutex
	items map[string]entry
}

// New returns a cache store that starts a background goroutine to reap expired
// entries every minute. The reap interval is deliberately coarse because the
// store is expected to hold at most 2-3 entries.
func New() *Store {
	s := &Store{items: make(map[string]entry)}
	go s.reap(60 * time.Second)
	return s
}

// Get returns the cached value and true if the key exists and is not expired.
func (s *Store) Get(key string) (any, bool) {
	s.mu.RLock()
	entry, ok := s.items[key]
	s.mu.RUnlock()
	if !ok || entry.expires <= time.Now().UnixMilli() {
		return nil, false
	}
	return entry.value, true
}

// Set stores a value with a TTL in milliseconds. Passing a TTL <= 0 means
// the entry never expires (caller must delete it explicitly).
func (s *Store) Set(key string, value any, ttlMS int64) {
	s.mu.Lock()
	s.items[key] = entry{value: value, expires: time.Now().UnixMilli() + ttlMS}
	s.mu.Unlock()
}

// Delete removes a key.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
}

func (s *Store) reap(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now().UnixMilli()
		for key, entry := range s.items {
			if entry.expires > 0 && entry.expires <= now {
				delete(s.items, key)
			}
		}
		s.mu.Unlock()
	}
}