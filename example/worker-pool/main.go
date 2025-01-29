package main

import (
	"github.com/baditaflorin/l"
	"github.com/baditaflorin/l/example/worker-pool/worker"
	"os"
	"time"
)

func main() {
	// Setup async logging with buffer
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

	pool := worker.NewPool(50)

	// Submit jobs
	for i := 0; i < 20; i++ {
		pool.Submit(i)
	}

	// Wait for completion
	time.Sleep(5 * time.Second)
	pool.Stop()
}
