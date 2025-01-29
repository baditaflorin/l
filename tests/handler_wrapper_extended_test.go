package tests

import (
	"context"
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHealthCheckerSuperExtended(t *testing.T) {
	factory := l.NewStandardFactory()
	config := l.Config{
		JsonFormat: true,
		Output:     io.Discard,
	}

	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	t.Run("Multiple Start/Stop Cycles", func(t *testing.T) {
		checker, err := factory.CreateHealthChecker(logger)
		require.NoError(t, err)

		var wg sync.WaitGroup
		// Use atomic flag to track if health checker is running
		var isRunning atomic.Bool

		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(iteration int) {
				defer wg.Done()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Only start if not already running
				if !isRunning.Load() {
					if isRunning.CompareAndSwap(false, true) {
						checker.Start(ctx)
					}
				}

				// Perform health check
				err := checker.Check()
				assert.NoError(t, err)

				time.Sleep(50 * time.Millisecond)

				// Only stop if running
				if isRunning.Load() {
					if isRunning.CompareAndSwap(true, false) {
						checker.Stop()
					}
				}
			}(i)
			// Wait between iterations to avoid race conditions
			time.Sleep(100 * time.Millisecond)
		}
		wg.Wait()
	})

	t.Run("Concurrent Start/Stop", func(t *testing.T) {
		checker, err := factory.CreateHealthChecker(logger)
		require.NoError(t, err)

		var wg sync.WaitGroup
		var isRunning atomic.Bool
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		wg.Add(2)

		// Start in one goroutine
		go func() {
			defer wg.Done()
			if !isRunning.Load() {
				if isRunning.CompareAndSwap(false, true) {
					checker.Start(ctx)
				}
			}
		}()

		// Stop in another goroutine
		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond)
			if isRunning.Load() {
				if isRunning.CompareAndSwap(true, false) {
					checker.Stop()
				}
			}
		}()

		wg.Wait()
	})

	t.Run("Rapid Start/Stop Cycles", func(t *testing.T) {
		checker, err := factory.CreateHealthChecker(logger)
		require.NoError(t, err)

		var wg sync.WaitGroup
		var isRunning atomic.Bool
		cycles := 5

		for i := 0; i < cycles; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx := context.Background()

				if !isRunning.Load() {
					if isRunning.CompareAndSwap(false, true) {
						checker.Start(ctx)
						// Perform quick health check
						err := checker.Check()
						assert.NoError(t, err)
						time.Sleep(10 * time.Millisecond)
						if isRunning.CompareAndSwap(true, false) {
							checker.Stop()
						}
					}
				}
			}()
		}

		wg.Wait()
	})
}
