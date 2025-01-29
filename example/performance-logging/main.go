package main

import (
	"github.com/baditaflorin/l"
	"os"
	"runtime"
	"time"
)

func main() {
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     os.Stdout,
		JsonFormat: true,
		AsyncWrite: true,
		BufferSize: 1000,
	}

	logger, err := factory.CreateLogger(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// Start performance monitoring
	go monitorPerformance(logger)

	// Simulate application work
	for i := 0; i < 10; i++ {
		simulateWork(logger)
		time.Sleep(time.Second)
	}

	if err := logger.Flush(); err != nil {
		panic(err)
	}
}

func monitorPerformance(logger l.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var m runtime.MemStats
	for range ticker.C {
		runtime.ReadMemStats(&m)

		logger.Info("Performance metrics",
			"goroutines", runtime.NumGoroutine(),
			"heap_alloc_mb", m.Alloc/1024/1024,
			"heap_sys_mb", m.HeapSys/1024/1024,
			"gc_cycles", m.NumGC,
		)
	}
}

func simulateWork(logger l.Logger) {
	start := time.Now()

	// Simulate CPU-intensive work
	data := make([]int, 100000)
	for i := range data {
		data[i] = i * 2
		logger.Debug("Processing data", "index", i)
	}

	duration := time.Since(start)
	logger.Info("Work completed",
		"duration_ms", duration.Milliseconds(),
		"data_size", len(data),
	)
}
