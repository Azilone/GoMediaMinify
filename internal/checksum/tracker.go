package checksum

import "sync"

// Tracker keeps track of checksums observed during a run.
type Tracker struct {
	mu        sync.RWMutex
	checksums map[uint64][]string
}

// NewTracker creates an empty tracker.
func NewTracker() *Tracker {
	return &Tracker{
		checksums: make(map[uint64][]string),
	}
}

// Register records that a checksum has been seen for the given file path.
func (t *Tracker) Register(sum uint64, path string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.checksums[sum] = append(t.checksums[sum], path)
}

// IsDuplicate reports whether the checksum was already registered.
func (t *Tracker) IsDuplicate(sum uint64) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	paths := t.checksums[sum]
	return len(paths) > 0
}

// FirstPath returns the first path associated with a checksum.
func (t *Tracker) FirstPath(sum uint64) (string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	paths := t.checksums[sum]
	if len(paths) == 0 {
		return "", false
	}
	return paths[0], true
}

// Stats returns the number of unique checksums and duplicates.
func (t *Tracker) Stats() (unique int, duplicates int) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	total := 0
	for _, paths := range t.checksums {
		total += len(paths)
	}

	unique = len(t.checksums)
	duplicates = total - unique
	return
}
