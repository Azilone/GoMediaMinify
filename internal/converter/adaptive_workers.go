package converter

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/kevindurb/media-converter/internal/config"
	"github.com/kevindurb/media-converter/internal/logger"
)

// ResourceSnapshot stores a single polling result for system utilisation.
type ResourceSnapshot struct {
	CPUPercent          float64
	MemAvailablePercent float64
	CPUMeasured         bool
	MemMeasured         bool
}

// ResourceMonitor periodically samples CPU and memory usage using
// lightweight OS-specific helpers. Failures are logged at most once per field.
type ResourceMonitor struct {
	interval time.Duration
	log      *logger.Logger

	warnMu    sync.Mutex
	warnedCPU bool
	warnedMem bool
}

func NewResourceMonitor(interval time.Duration, log *logger.Logger) *ResourceMonitor {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	return &ResourceMonitor{
		interval: interval,
		log:      log,
	}
}

func (m *ResourceMonitor) Start(ctx context.Context) <-chan ResourceSnapshot {
	out := make(chan ResourceSnapshot, 1)

	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				snap := ResourceSnapshot{}

				if cpu, err := cpuUsagePercent(); err == nil {
					snap.CPUPercent = cpu
					snap.CPUMeasured = true
				} else {
					m.warnOnceCPU(err)
				}

				if mem, err := memoryAvailablePercent(); err == nil {
					snap.MemAvailablePercent = mem
					snap.MemMeasured = true
				} else {
					m.warnOnceMem(err)
				}

				select {
				case out <- snap:
				default:
					// Drop sample if consumer is behind.
				}
			}
		}
	}()

	return out
}

func (m *ResourceMonitor) warnOnceCPU(err error) {
	m.warnMu.Lock()
	defer m.warnMu.Unlock()
	if m.warnedCPU {
		return
	}
	m.warnedCPU = true
	if m.log != nil {
		m.log.Warn(fmt.Sprintf("Adaptive workers: CPU metrics unavailable (%v)", err))
	}
}

func (m *ResourceMonitor) warnOnceMem(err error) {
	m.warnMu.Lock()
	defer m.warnMu.Unlock()
	if m.warnedMem {
		return
	}
	m.warnedMem = true
	if m.log != nil {
		m.log.Warn(fmt.Sprintf("Adaptive workers: memory metrics unavailable (%v)", err))
	}
}

// runAdaptiveController listens for snapshots and adjusts the limiter within
// the bounds configured by the user.
func runAdaptiveController(ctx context.Context, limiter *AdaptiveLimiter, cfg config.AdaptiveWorkerConfig, snapshots <-chan ResourceSnapshot, log *logger.Logger) {
	if cfg.MaxWorkers < 1 {
		return
	}

	currentLimit := limiter.Limit()
	highStreak := 0
	lowStreak := 0
	comfortMemThreshold := math.Min(100, cfg.MemLowPercent+5)

	if log != nil {
		log.Info(fmt.Sprintf("Adaptive workers enabled: starting with %d concurrent video conversions (min=%d, max=%d)", currentLimit, cfg.MinWorkers, cfg.MaxWorkers))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case snap, ok := <-snapshots:
			if !ok {
				return
			}

			busy := false
			if snap.CPUMeasured && snap.CPUPercent >= cfg.CPUHigh {
				busy = true
			}
			if snap.MemMeasured && snap.MemAvailablePercent <= cfg.MemLowPercent {
				busy = true
			}

			if busy {
				highStreak++
				lowStreak = 0
				limit := limiter.Limit()
				if highStreak >= 1 && limit > cfg.MinWorkers {
					newLimit := limit - 1
					limiter.SetLimit(newLimit)
					if log != nil {
						log.Warn(fmt.Sprintf("Adaptive workers: reducing video concurrency to %d (CPU %.1f%%, free memory %.1f%%)", newLimit, snap.CPUPercent, snap.MemAvailablePercent))
					}
				}
				continue
			}

			// Consider scaling up only when metrics indicate comfort.
			comfortable := true
			if snap.CPUMeasured && snap.CPUPercent > cfg.CPULow {
				comfortable = false
			}
			if snap.MemMeasured && snap.MemAvailablePercent < comfortMemThreshold {
				comfortable = false
			}

			if !comfortable {
				highStreak = 0
				lowStreak = 0
				continue
			}

			lowStreak++
			if lowStreak < 2 {
				continue
			}

			highStreak = 0
			lowStreak = 0
			limit := limiter.Limit()
			if limit < cfg.MaxWorkers {
				newLimit := limit + 1
				limiter.SetLimit(newLimit)
				if log != nil {
					log.Info(fmt.Sprintf("Adaptive workers: increasing video concurrency to %d (CPU %.1f%%, free memory %.1f%%)", newLimit, snap.CPUPercent, snap.MemAvailablePercent))
				}
			}
		}
	}
}
