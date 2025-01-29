package examples

import (
	"context"
	"github.com/baditaflorin/l"
	"github.com/baditaflorin/l/example/worker-pool/worker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool(t *testing.T) {
	// Create logger for testing
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     io.Discard,
		JsonFormat: true,
	}

	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	t.Run("Basic Pool Operations", func(t *testing.T) {
		pool := worker.NewPool(5, logger)
		require.NotNil(t, pool)

		var wg sync.WaitGroup
		wg.Add(10)

		// Submit some jobs
		for i := 0; i < 10; i++ {
			i := i // Create new variable for goroutine
			go func() {
				defer wg.Done()
				err := pool.Submit(i)
				assert.NoError(t, err)
			}()
		}

		wg.Wait()

		// Give time for jobs to process
		time.Sleep(time.Second)

		// Stop the pool
		pool.Stop()
	})

	t.Run("Pool Under Load", func(t *testing.T) {
		pool := worker.NewPool(10, logger)
		require.NotNil(t, pool)

		// Track processed jobs
		var processedJobs atomic.Int32
		var wg sync.WaitGroup
		totalJobs := 100
		wg.Add(totalJobs)

		// Submit many jobs concurrently
		for i := 0; i < totalJobs; i++ {
			i := i // Create new variable for goroutine
			go func() {
				defer wg.Done()
				err := pool.Submit(i)
				if err == nil {
					processedJobs.Add(1)
				}
			}()
		}

		wg.Wait()
		time.Sleep(time.Second)

		pool.Stop()

		// Verify all jobs that were successfully submitted were processed
		assert.Equal(t, int32(totalJobs), processedJobs.Load())
	})

	t.Run("Pool Graceful Shutdown", func(t *testing.T) {
		pool := worker.NewPool(3, logger)
		require.NotNil(t, pool)

		var wg sync.WaitGroup
		wg.Add(5)

		// Submit slow jobs
		for i := 0; i < 5; i++ {
			i := i
			go func() {
				defer wg.Done()
				err := pool.Submit(i)
				assert.NoError(t, err)
			}()
		}

		wg.Wait()

		// Start shutdown
		done := make(chan struct{})
		go func() {
			pool.Stop()
			close(done)
		}()

		// Wait for shutdown with timeout
		select {
		case <-done:
			// Success - pool stopped gracefully
		case <-time.After(5 * time.Second):
			t.Error("Pool shutdown timed out")
		}
	})

	t.Run("Pool Cancel Context", func(t *testing.T) {
		pool := worker.NewPool(3, logger)
		require.NotNil(t, pool)

		// Use WaitGroup to ensure all submissions are complete
		var wg sync.WaitGroup
		wg.Add(5)

		// Submit jobs that will be cancelled
		for i := 0; i < 5; i++ {
			i := i
			go func() {
				defer wg.Done()
				err := pool.Submit(i)
				assert.NoError(t, err)
			}()
		}

		// Wait for all submissions
		wg.Wait()

		// Stop the pool
		pool.Stop()

		// Try to submit after stop - should return context.Canceled
		err := pool.Submit(999)
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("Pool Error Handling", func(t *testing.T) {
		pool := worker.NewPool(3, logger)
		require.NotNil(t, pool)

		var wg sync.WaitGroup
		wg.Add(10)

		// Submit jobs that will cause errors (divisible by 5)
		for i := 0; i < 10; i++ {
			i := i
			go func() {
				defer wg.Done()
				err := pool.Submit(i * 5) // These should trigger error logging
				assert.NoError(t, err)
			}()
		}

		wg.Wait()
		time.Sleep(time.Second)

		pool.Stop()
	})
}
