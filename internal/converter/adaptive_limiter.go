package converter

import "sync"

// AdaptiveLimiter controls the number of concurrent workers and allows
// adjusting the limit at runtime without interrupting in-flight work.
type AdaptiveLimiter struct {
	mu     sync.Mutex
	cond   *sync.Cond
	limit  int
	active int
}

// NewAdaptiveLimiter creates a limiter with the provided initial limit.
func NewAdaptiveLimiter(limit int) *AdaptiveLimiter {
	if limit < 1 {
		limit = 1
	}

	l := &AdaptiveLimiter{
		limit: limit,
	}
	l.cond = sync.NewCond(&l.mu)
	return l
}

// Acquire blocks until a worker slot is available.
func (l *AdaptiveLimiter) Acquire() {
	l.mu.Lock()
	for l.active >= l.limit {
		l.cond.Wait()
	}
	l.active++
	l.mu.Unlock()
}

// Release frees a worker slot and wakes up any waiting goroutine.
func (l *AdaptiveLimiter) Release() {
	l.mu.Lock()
	if l.active > 0 {
		l.active--
	}
	l.cond.Signal()
	l.mu.Unlock()
}

// SetLimit updates the maximum number of concurrent workers. If the limit
// increases, waiting goroutines are woken up so that they can proceed.
func (l *AdaptiveLimiter) SetLimit(limit int) {
	if limit < 1 {
		limit = 1
	}

	l.mu.Lock()
	previous := l.limit
	l.limit = limit

	if limit > previous {
		l.cond.Broadcast()
	} else {
		l.cond.Signal()
	}
	l.mu.Unlock()
}

// Limit returns the current maximum number of workers.
func (l *AdaptiveLimiter) Limit() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.limit
}

// Active returns the number of workers currently holding the limiter.
func (l *AdaptiveLimiter) Active() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.active
}
