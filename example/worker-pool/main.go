package main

import (
	"github.com/baditaflorin/l"
	"github.com/baditaflorin/l/example/worker-pool/worker"
	"os"
	"time"
)

func main() {
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     os.Stdout,
		JsonFormat: true,
		AsyncWrite: true,
		BufferSize: 1024,
	}

	logger, err := factory.CreateLogger(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	pool := worker.NewPool(50, logger)

	// Submit jobs
	for i := 0; i < 20; i++ {
		pool.Submit(i)
	}

	// Wait for completion
	time.Sleep(5 * time.Second)
	pool.Stop()

	// Ensure all logs are written
	if err := logger.Flush(); err != nil {
		panic(err)
	}
}
