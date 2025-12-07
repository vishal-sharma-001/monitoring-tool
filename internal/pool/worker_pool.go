package pool

import (
	"context"
	"fmt"
	"sync"
)

// Task represents a unit of work to be executed by the worker pool
// Following Interface Segregation Principle: simple function signature
type Task func(ctx context.Context) error

// WorkerPool manages a fixed number of goroutine workers
// Following SOLID principles:
// - Single Responsibility: manages worker lifecycle and task distribution
// - Open/Closed: can be extended with different task types
type WorkerPool struct {
	workerCount int
	taskQueue   chan Task
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mu          sync.RWMutex
	stopped     bool
}

// NewWorkerPool creates a new worker pool
// Parameters:
//   - workerCount: number of goroutine workers
//   - queueSize: capacity of the task queue channel (buffered)
func NewWorkerPool(workerCount int, queueSize int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}
	if queueSize <= 0 {
		queueSize = 100
	}

	return &WorkerPool{
		workerCount: workerCount,
		taskQueue:   make(chan Task, queueSize),
		stopChan:    make(chan struct{}),
		stopped:     false,
	}
}

// Start initializes and starts all worker goroutines
// Each worker consumes tasks from the taskQueue channel
func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(ctx, i)
	}
}

// worker is the goroutine function that processes tasks
// Following Go concurrency patterns: select with multiple channels
func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done()

	for {
		select {
		case task, ok := <-wp.taskQueue:
			if !ok {
				// Channel closed, worker should exit
				return
			}

			// Execute task with error handling
			if err := task(ctx); err != nil {
				// Log error but continue processing
				// In production, could send to error channel or metrics
				_ = err // Error logged by caller or ignored for MVP
			}

		case <-wp.stopChan:
			// Stop signal received
			return

		case <-ctx.Done():
			// Context cancelled
			return
		}
	}
}

// Submit adds a task to the queue for processing
// Returns error if pool is stopped or queue is full
// Following Fail-Fast principle
func (wp *WorkerPool) Submit(task Task) error {
	wp.mu.RLock()
	if wp.stopped {
		wp.mu.RUnlock()
		return fmt.Errorf("worker pool is stopped")
	}
	wp.mu.RUnlock()

	select {
	case wp.taskQueue <- task:
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// SubmitWithContext adds a task with context checking
// Returns error if context is done, pool is stopped, or queue is full
func (wp *WorkerPool) SubmitWithContext(ctx context.Context, task Task) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return wp.Submit(task)
}

// Stop gracefully shuts down the worker pool
// Waits for all in-flight tasks to complete
// Following graceful shutdown pattern
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	if wp.stopped {
		wp.mu.Unlock()
		return
	}
	wp.stopped = true
	wp.mu.Unlock()

	// Close stop channel to signal all workers
	close(wp.stopChan)

	// Wait for all workers to finish
	wp.wg.Wait()

	// Close task queue
	close(wp.taskQueue)
}

// StopWithTimeout stops the pool with a timeout
// Returns error if timeout is exceeded
func (wp *WorkerPool) StopWithTimeout(timeout context.Context) error {
	done := make(chan struct{})

	go func() {
		wp.Stop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-timeout.Done():
		return fmt.Errorf("worker pool shutdown timeout exceeded")
	}
}

// GetWorkerCount returns the number of workers in the pool
func (wp *WorkerPool) GetWorkerCount() int {
	return wp.workerCount
}

// GetQueueSize returns the current number of tasks in the queue
func (wp *WorkerPool) GetQueueSize() int {
	return len(wp.taskQueue)
}

// IsStopped returns whether the pool has been stopped
func (wp *WorkerPool) IsStopped() bool {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.stopped
}
