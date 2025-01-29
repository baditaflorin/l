// File: tests/benchmark_test.go
package tests

import (
	"github.com/baditaflorin/l"
	"testing"
)

func BenchmarkLogger(b *testing.B) {
	factory := l.NewStandardFactory()
	config := l.Config{
		AsyncWrite: true,
		BufferSize: 1024 * 1024,
	}

	logger, err := factory.CreateLogger(config)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("SimpleMessage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info("benchmark test")
		}
	})

	b.Run("StructuredMessage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Info("benchmark test",
				"iteration", i,
				"level", "info",
				"component", "benchmark",
			)
		}
	})

	b.Run("ErrorWithStack", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.Error("benchmark error",
				"error", "test error",
				"code", 500,
			)
		}
	})
}
