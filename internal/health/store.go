package health

import "sync"

// Store maintains the health status for each endpoint (thread-safe).
type Store struct {
	mu     sync.RWMutex
	status map[string]bool
}

// NewStore initializes a new Store. Unchecked endpoints are considered healthy.
func NewStore() *Store {
	return &Store{status: make(map[string]bool)}
}

// Set updates the health status for a given address.
func (s *Store) Set(addr string, healthy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status[addr] = healthy
}

// IsHealthy returns the health status for an address.
// Unchecked endpoints are considered healthy (optimistic default = fail-open).
func (s *Store) IsHealthy(addr string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.status[addr]
	if !ok {
		return true
	}
	return v
}

// All retrieves a snapshot of all statuses. Useful for the /health API.
func (s *Store) All() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]bool, len(s.status))
	for k, v := range s.status {
		out[k] = v
	}
	return out
}
