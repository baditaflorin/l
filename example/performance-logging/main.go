// example/performance-logging/main.go
package main

import (
	"github.com/baditaflorin/l"
	"os"
	"runtime"
	"time"
)

func main() {
	err := l.Setup(l.Options{
		Output:     os.Stdout,
		JsonFormat: true,
		AsyncWrite: true,
		BufferSize: 1024,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	// Start performance monitoring
	go monitorPerformance()

	// Simulate application work
	for i := 0; i < 10; i++ {
		simulateWork()
		time.Sleep(time.Second)
	}
}

func monitorPerformance() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var m runtime.MemStats
	for range ticker.C {
		runtime.ReadMemStats(&m)

		l.Info("Performance metrics",
			"goroutines", runtime.NumGoroutine(),
			"heap_alloc_mb", m.Alloc/1024/1024,
			"heap_sys_mb", m.HeapSys/1024/1024,
			"gc_cycles", m.NumGC,
		)
	}
}

func simulateWork() {
	start := time.Now()
	// Simulate CPU-intensive work
	data := make([]int, 1000000)
	for i := range data {
		data[i] = i * 2
	}

	l.Info("Work completed",
		"duration_ms", time.Since(start).Milliseconds(),
		"data_size", len(data),
	)
}
