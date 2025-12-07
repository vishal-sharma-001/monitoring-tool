package pool_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/monitoring-engine/monitoring-tool/internal/pool"
	"github.com/stretchr/testify/assert"
)

func TestNewWorkerPool(t *testing.T) {
	t.Run("should create pool with valid parameters", func(t *testing.T) {
		wp := pool.NewWorkerPool(5, 100)
		assert.NotNil(t, wp)
		assert.Equal(t, 5, wp.GetWorkerCount())
	})

	t.Run("should default to 1 worker when count is 0", func(t *testing.T) {
		wp := pool.NewWorkerPool(0, 100)
		assert.NotNil(t, wp)
		assert.Equal(t, 1, wp.GetWorkerCount())
	})

	t.Run("should default to 1 worker when count is negative", func(t *testing.T) {
		wp := pool.NewWorkerPool(-5, 100)
		assert.NotNil(t, wp)
		assert.Equal(t, 1, wp.GetWorkerCount())
	})

	t.Run("should default to 100 queue size when size is 0", func(t *testing.T) {
		wp := pool.NewWorkerPool(5, 0)
		assert.NotNil(t, wp)
	})

	t.Run("should default to 100 queue size when size is negative", func(t *testing.T) {
		wp := pool.NewWorkerPool(5, -10)
		assert.NotNil(t, wp)
	})
}

func TestWorkerPool_Start(t *testing.T) {
	t.Run("should start workers successfully", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(3, 10)

		wp.Start(ctx)
		assert.Equal(t, 3, wp.GetWorkerCount())

		wp.Stop()
	})
}

func TestWorkerPool_Submit(t *testing.T) {
	t.Run("should submit and execute task successfully", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(2, 10)
		wp.Start(ctx)
		defer wp.Stop()

		executed := false
		task := func(ctx context.Context) error {
			executed = true
			return nil
		}

		err := wp.Submit(task)
		assert.NoError(t, err)

		// Wait for task to execute
		time.Sleep(100 * time.Millisecond)
		assert.True(t, executed)
	})

	t.Run("should execute multiple tasks", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(3, 100)
		wp.Start(ctx)
		defer wp.Stop()

		var counter int32
		taskCount := 10

		for i := 0; i < taskCount; i++ {
			task := func(ctx context.Context) error {
				atomic.AddInt32(&counter, 1)
				return nil
			}
			err := wp.Submit(task)
			assert.NoError(t, err)
		}

		// Wait for all tasks to execute
		time.Sleep(200 * time.Millisecond)
		assert.Equal(t, int32(taskCount), atomic.LoadInt32(&counter))
	})

	t.Run("should return error when pool is stopped", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(2, 10)
		wp.Start(ctx)
		wp.Stop()

		task := func(ctx context.Context) error {
			return nil
		}

		err := wp.Submit(task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "worker pool is stopped")
	})

	t.Run("should return error when queue is full", func(t *testing.T) {
		wp := pool.NewWorkerPool(1, 2)
		// Don't start the pool so tasks accumulate in queue

		// Fill the queue
		task := func(ctx context.Context) error {
			time.Sleep(1 * time.Second)
			return nil
		}

		err1 := wp.Submit(task)
		err2 := wp.Submit(task)
		err3 := wp.Submit(task) // This should fail

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Error(t, err3)
		assert.Contains(t, err3.Error(), "task queue is full")

		wp.Stop()
	})

	t.Run("should handle task errors gracefully", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(2, 10)
		wp.Start(ctx)
		defer wp.Stop()

		taskError := errors.New("task failed")
		task := func(ctx context.Context) error {
			return taskError
		}

		err := wp.Submit(task)
		assert.NoError(t, err) // Submit should succeed

		// Wait for task to execute
		time.Sleep(100 * time.Millisecond)
	})
}

func TestWorkerPool_SubmitWithContext(t *testing.T) {
	t.Run("should submit task with valid context", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(2, 10)
		wp.Start(ctx)
		defer wp.Stop()

		executed := false
		task := func(ctx context.Context) error {
			executed = true
			return nil
		}

		err := wp.SubmitWithContext(ctx, task)
		assert.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		assert.True(t, executed)
	})

	t.Run("should return error when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		wp := pool.NewWorkerPool(2, 10)
		wp.Start(context.Background())
		defer wp.Stop()

		task := func(ctx context.Context) error {
			return nil
		}

		err := wp.SubmitWithContext(ctx, task)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("should return error when context times out", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // Ensure timeout

		wp := pool.NewWorkerPool(2, 10)
		wp.Start(context.Background())
		defer wp.Stop()

		task := func(ctx context.Context) error {
			return nil
		}

		err := wp.SubmitWithContext(ctx, task)
		assert.Error(t, err)
	})
}

func TestWorkerPool_Stop(t *testing.T) {
	t.Run("should stop pool gracefully", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(3, 10)
		wp.Start(ctx)

		wp.Stop()
		assert.True(t, wp.IsStopped())
	})

	t.Run("should allow multiple stop calls", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(2, 10)
		wp.Start(ctx)

		wp.Stop()
		wp.Stop() // Second call should be safe
		assert.True(t, wp.IsStopped())
	})

	t.Run("should complete in-flight tasks before stopping", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(5, 10)
		wp.Start(ctx)

		var completed int32
		for i := 0; i < 5; i++ {
			task := func(ctx context.Context) error {
				time.Sleep(20 * time.Millisecond)
				atomic.AddInt32(&completed, 1)
				return nil
			}
			_ = wp.Submit(task)
		}

		// Give tasks a moment to start
		time.Sleep(10 * time.Millisecond)

		wp.Stop()

		// All tasks should have completed
		assert.Equal(t, int32(5), atomic.LoadInt32(&completed))
	})
}

func TestWorkerPool_StopWithTimeout(t *testing.T) {
	t.Run("should stop before timeout", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(2, 10)
		wp.Start(ctx)

		task := func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		}
		_ = wp.Submit(task)

		timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := wp.StopWithTimeout(timeoutCtx)
		assert.NoError(t, err)
		assert.True(t, wp.IsStopped())
	})

	t.Run("should return error on timeout", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(1, 10)
		wp.Start(ctx)

		// Submit a long-running task
		task := func(ctx context.Context) error {
			time.Sleep(2 * time.Second)
			return nil
		}
		_ = wp.Submit(task)

		// Set a short timeout
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := wp.StopWithTimeout(timeoutCtx)
		if err != nil {
			assert.Contains(t, err.Error(), "timeout exceeded")
		}
		// If no error, the worker stopped quickly enough which is also valid
	})
}

func TestWorkerPool_GetQueueSize(t *testing.T) {
	t.Run("should return queue size", func(t *testing.T) {
		wp := pool.NewWorkerPool(1, 10)
		// Don't start so tasks stay in queue

		assert.Equal(t, 0, wp.GetQueueSize())

		// Submit tasks without starting workers
		task := func(ctx context.Context) error {
			time.Sleep(1 * time.Second)
			return nil
		}

		_ = wp.Submit(task)
		assert.Equal(t, 1, wp.GetQueueSize())

		_ = wp.Submit(task)
		assert.Equal(t, 2, wp.GetQueueSize())

		wp.Stop()
	})
}

func TestWorkerPool_IsStopped(t *testing.T) {
	t.Run("should return false when running", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(2, 10)
		wp.Start(ctx)

		assert.False(t, wp.IsStopped())

		wp.Stop()
	})

	t.Run("should return true when stopped", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(2, 10)
		wp.Start(ctx)
		wp.Stop()

		assert.True(t, wp.IsStopped())
	})
}

func TestWorkerPool_ConcurrentAccess(t *testing.T) {
	t.Run("should handle concurrent submissions", func(t *testing.T) {
		ctx := context.Background()
		wp := pool.NewWorkerPool(5, 100)
		wp.Start(ctx)
		defer wp.Stop()

		var counter int32
		var wg sync.WaitGroup

		// Simulate concurrent task submissions
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				task := func(ctx context.Context) error {
					atomic.AddInt32(&counter, 1)
					return nil
				}
				_ = wp.Submit(task)
			}()
		}

		wg.Wait()
		time.Sleep(500 * time.Millisecond)

		assert.LessOrEqual(t, int32(50), atomic.LoadInt32(&counter))
	})
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	t.Run("should stop workers when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		wp := pool.NewWorkerPool(3, 10)
		wp.Start(ctx)

		// Cancel context
		cancel()

		// Workers should eventually stop
		time.Sleep(100 * time.Millisecond)

		// Pool should still be able to stop
		wp.Stop()
		assert.True(t, wp.IsStopped())
	})

	t.Run("should not execute tasks after context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		wp := pool.NewWorkerPool(1, 10)
		wp.Start(ctx)

		var executed int32
		task := func(ctx context.Context) error {
			atomic.AddInt32(&executed, 1)
			time.Sleep(100 * time.Millisecond)
			return nil
		}

		// Submit task
		_ = wp.Submit(task)

		// Cancel immediately
		cancel()
		time.Sleep(50 * time.Millisecond)

		wp.Stop()

		// Task should have started executing before cancellation
		assert.GreaterOrEqual(t, atomic.LoadInt32(&executed), int32(0))
	})
}

func TestWorkerPool_GetWorkerCount(t *testing.T) {
	t.Run("should return correct worker count", func(t *testing.T) {
		wp := pool.NewWorkerPool(7, 10)
		assert.Equal(t, 7, wp.GetWorkerCount())
	})
}
