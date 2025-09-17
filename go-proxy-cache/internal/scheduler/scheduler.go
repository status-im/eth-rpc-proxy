package scheduler

import (
	"context"
	"sync"
	"time"
)

// Scheduler manages a background task that runs at regular intervals
type Scheduler struct {
	interval time.Duration
	task     func()
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	running  bool
}

// New creates a new Scheduler instance
func New(interval time.Duration, task func()) *Scheduler {
	return &Scheduler{
		interval: interval,
		task:     task,
	}
}

// Start begins executing the task at the specified interval
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.task()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop terminates the periodic task execution
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.cancel()
	s.wg.Wait()
	s.running = false
}

// IsRunning returns true if the task is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
