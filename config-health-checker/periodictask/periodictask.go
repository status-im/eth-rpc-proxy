package periodictask

import (
	"context"
	"sync"
	"time"
)

// PeriodicTask manages a background task that runs at regular intervals
type PeriodicTask struct {
	interval time.Duration
	task     func()
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// New creates a new PeriodicTask instance
func New(interval time.Duration, task func()) *PeriodicTask {
	return &PeriodicTask{
		interval: interval,
		task:     task,
	}
}

// Start begins executing the task at the specified interval
func (pt *PeriodicTask) Start() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.running {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	pt.cancel = cancel
	pt.running = true

	pt.wg.Add(1)
	go func() {
		defer pt.wg.Done()
		ticker := time.NewTicker(pt.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pt.task()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop terminates the periodic task execution
func (pt *PeriodicTask) Stop() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if !pt.running {
		return
	}

	pt.cancel()
	pt.wg.Wait()
	pt.running = false
}

// IsRunning returns true if the task is currently running
func (pt *PeriodicTask) IsRunning() bool {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return pt.running
}
