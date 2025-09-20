package converter

import (
	"context"
	"testing"
	"time"

	"github.com/kevindurb/media-converter/internal/config"
)

func TestAdaptiveControllerScalesDownAndUp(t *testing.T) {
	cfg := config.AdaptiveWorkerConfig{
		Enabled:       true,
		MinWorkers:    1,
		MaxWorkers:    3,
		CPUHigh:       80,
		CPULow:        40,
		MemLowPercent: 20,
		CheckInterval: time.Second,
	}

	limiter := NewAdaptiveLimiter(cfg.MaxWorkers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	snapshots := make(chan ResourceSnapshot)

	go runAdaptiveController(ctx, limiter, cfg, snapshots, nil)

	snapshots <- ResourceSnapshot{CPUMeasured: true, CPUPercent: 90, MemMeasured: true, MemAvailablePercent: 50}
	waitForLimit(t, limiter, 2)

	snapshots <- ResourceSnapshot{CPUMeasured: true, CPUPercent: 90, MemMeasured: true, MemAvailablePercent: 50}
	waitForLimit(t, limiter, 1)

	// Provide relaxed samples to scale up (needs two consecutive comfortable samples)
	snapshots <- ResourceSnapshot{CPUMeasured: true, CPUPercent: 20, MemMeasured: true, MemAvailablePercent: 80}
	snapshots <- ResourceSnapshot{CPUMeasured: true, CPUPercent: 20, MemMeasured: true, MemAvailablePercent: 80}
	waitForLimit(t, limiter, 2)
}

func waitForLimit(t *testing.T, limiter *AdaptiveLimiter, expected int) {
	t.Helper()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if limiter.Limit() == expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for limiter to reach %d (current %d)", expected, limiter.Limit())
}
