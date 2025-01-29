package worker

import (
	"context"
	"github.com/baditaflorin/l"
	"sync"
	"time"
)

type Pool struct {
	jobs    chan int
	workers int
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewPool(workers int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
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

	l.Info("Worker started", "worker_id", id)
	defer l.Info("Worker stopped", "worker_id", id)

	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}

			l.Info("Processing job",
				"worker_id", id,
				"job_id", job,
			)

			// Simulate work
			select {
			case <-p.ctx.Done():
				return
			case <-time.After(time.Second):
			}

			if job%5 == 0 {
				l.Error("Job failed",
					"worker_id", id,
					"job_id", job,
					"reason", "divisible by 5",
				)
				continue
			}

			l.Info("Job completed",
				"worker_id", id,
				"job_id", job,
			)
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
		l.Info("All workers stopped gracefully")
	case <-time.After(3 * time.Second):
		l.Warn("Timeout waiting for workers to stop")
	}
}
