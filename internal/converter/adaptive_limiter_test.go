package converter

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestAdaptiveLimiterBlocksWhenFull(t *testing.T) {
	limiter := NewAdaptiveLimiter(2)

	limiter.Acquire()
	limiter.Acquire()

	blocked := make(chan struct{})
	go func() {
		limiter.Acquire()
		close(blocked)
	}()

	select {
	case <-blocked:
		t.Fatalf("expected acquire to block when limit reached")
	case <-time.After(100 * time.Millisecond):
		// expected path
	}

	limiter.Release()

	select {
	case <-blocked:
		// acquire should proceed now that slot freed
	case <-time.After(time.Second):
		t.Fatalf("expected acquire to proceed after release")
	}
}

func TestAdaptiveLimiterAdjustsLimit(t *testing.T) {
	limiter := NewAdaptiveLimiter(2)

	limiter.Acquire()
	limiter.Acquire()

	limiter.SetLimit(1)

	resumed := make(chan struct{})
	go func() {
		limiter.Acquire()
		close(resumed)
	}()

	// release one slot; waiting goroutine should still block because active==limit
	limiter.Release()

	select {
	case <-resumed:
		t.Fatalf("acquire should remain blocked while active equals new limit")
	case <-time.After(100 * time.Millisecond):
		// expected
	}

	limiter.Release()

	select {
	case <-resumed:
	case <-time.After(time.Second):
		t.Fatalf("expected acquire to unblock after second release")
	}

	limiter.Release()
}

func TestAdaptiveLimiterIncreaseWakesWaiters(t *testing.T) {
	limiter := NewAdaptiveLimiter(1)

	var counter int32

	go func() {
		limiter.Acquire()
		atomic.AddInt32(&counter, 1)
	}()

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&counter) != 1 {
		t.Fatalf("first worker should have acquired immediately")
	}

	done := make(chan struct{})
	go func() {
		limiter.Acquire()
		atomic.AddInt32(&counter, 1)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&counter) != 1 {
		t.Fatalf("second worker should be waiting before limit increase")
	}

	limiter.SetLimit(2)

	select {
	case <-done:
		if atomic.LoadInt32(&counter) != 2 {
			t.Fatalf("expected counter to reach 2 after limit increase")
		}
	case <-time.After(time.Second):
		t.Fatalf("waiter should be released after increasing limit")
	}
}
