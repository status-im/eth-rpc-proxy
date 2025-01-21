package scheduler

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPeriodicTask(t *testing.T) {
	var counter int32

	// Create task that increments counter
	task := func() {
		atomic.AddInt32(&counter, 1)
	}

	// Create periodic task with 100ms interval
	pt := New(100*time.Millisecond, task)

	// Start the task
	pt.Start()
	assert.True(t, pt.IsRunning())

	// Wait for 3 executions
	time.Sleep(350 * time.Millisecond)

	// Stop the task
	pt.Stop()
	assert.False(t, pt.IsRunning())

	// Verify counter was incremented at least 3 times
	assert.GreaterOrEqual(t, atomic.LoadInt32(&counter), int32(3))

	// Wait a bit longer to ensure task is stopped
	time.Sleep(200 * time.Millisecond)
	finalCount := atomic.LoadInt32(&counter)

	// Verify counter didn't increment after stop
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, finalCount, atomic.LoadInt32(&counter))
}

func TestPeriodicTask_StopBeforeStart(t *testing.T) {
	pt := New(100*time.Millisecond, func() {})
	pt.Stop() // Should not panic
	assert.False(t, pt.IsRunning())
}

func TestPeriodicTask_DoubleStart(t *testing.T) {
	var counter int32
	pt := New(100*time.Millisecond, func() {
		atomic.AddInt32(&counter, 1)
	})

	pt.Start()
	pt.Start() // Second start should be ignored

	time.Sleep(150 * time.Millisecond)
	pt.Stop()

	assert.GreaterOrEqual(t, atomic.LoadInt32(&counter), int32(1))
}
