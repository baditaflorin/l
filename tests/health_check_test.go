package tests

import (
	"context"
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestHealthCheckerExtended(t *testing.T) {
	factory := l.NewStandardFactory()
	config := l.Config{
		JsonFormat: true,
	}

	logger, err := factory.CreateLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	t.Run("Health Checker Lifecycle", func(t *testing.T) {
		checker, err := factory.CreateHealthChecker(logger)
		require.NoError(t, err)

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Start the health checker
		checker.Start(ctx)

		// Give some time for check loop to run
		time.Sleep(100 * time.Millisecond)

		// Perform health check
		err = checker.Check()
		assert.NoError(t, err)

		// Stop the health checker
		checker.Stop()

		// Try health check after stop
		err = checker.Check()
		assert.NoError(t, err) // Should still work after stop
	})

	t.Run("Context Cancellation", func(t *testing.T) {
		checker, err := factory.CreateHealthChecker(logger)
		require.NoError(t, err)

		// Create a context that we'll cancel quickly
		ctx, cancel := context.WithCancel(context.Background())

		// Start the health checker
		checker.Start(ctx)

		// Cancel context immediately
		cancel()

		// Give some time for check loop to notice cancellation
		time.Sleep(100 * time.Millisecond)

		// Verify checker still responds to health checks
		err = checker.Check()
		assert.NoError(t, err)

		// Clean up
		checker.Stop()
	})

	t.Run("Multiple Start/Stop Cycles", func(t *testing.T) {
		checker, err := factory.CreateHealthChecker(logger)
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			ctx := context.Background()
			checker.Start(ctx)

			// Perform health check
			err = checker.Check()
			assert.NoError(t, err)

			// Stop checker
			checker.Stop()

			// Small pause between cycles
			time.Sleep(50 * time.Millisecond)
		}
	})

	t.Run("Concurrent Health Checks", func(t *testing.T) {
		checker, err := factory.CreateHealthChecker(logger)
		require.NoError(t, err)

		ctx := context.Background()
		checker.Start(ctx)
		defer checker.Stop()

		// Perform concurrent health checks
		concurrency := 10
		done := make(chan bool)

		for i := 0; i < concurrency; i++ {
			go func() {
				err := checker.Check()
				assert.NoError(t, err)
				done <- true
			}()
		}

		// Wait for all checks to complete
		for i := 0; i < concurrency; i++ {
			<-done
		}
	})
}
