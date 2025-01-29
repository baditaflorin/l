package worker

import (
	"context"
	"github.com/baditaflorin/l"
	"sync"
	"time"
)

type Pool struct {
	logger  l.Logger
	jobs    chan int
	workers int
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewPool(workers int, logger l.Logger) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		logger:  logger,
		jobs:    make(chan int, 100),
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
	}

	p.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go p.worker(i)
	}

	return p
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()

	// Create worker-specific logger
	workerLogger := p.logger.With("worker_id", id)
	workerLogger.Info("Worker started")
	defer workerLogger.Info("Worker stopped")

	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}

			workerLogger.Info("Processing job", "job_id", job)

			// Simulate work
			select {
			case <-p.ctx.Done():
				return
			case <-time.After(time.Second):
			}

			if job%5 == 0 {
				workerLogger.Error("Job failed",
					"job_id", job,
					"reason", "divisible by 5",
				)
				continue
			}

			workerLogger.Info("Job completed", "job_id", job)
		}
	}
}

func (p *Pool) Submit(job int) error {
	select {
	case <-p.ctx.Done():
		return context.Canceled
	case p.jobs <- job:
		return nil
	}
}

func (p *Pool) Stop() {
	// Signal shutdown
	p.cancel()

	// Close jobs channel
	close(p.jobs)

	// Wait for all workers with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("All workers stopped gracefully")
	case <-time.After(3 * time.Second):
		p.logger.Warn("Timeout waiting for workers to stop")
	}
}
