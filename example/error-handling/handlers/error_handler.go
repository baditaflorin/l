package handlers

import (
	"github.com/baditaflorin/l"
	"sync"
	"sync/atomic"
	"time"
)

type ErrorHandler struct {
	errors   []error
	mu       sync.RWMutex
	stopChan chan struct{}
	done     chan struct{}
	stopped  atomic.Bool
	stopOnce sync.Once
}

func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		stopChan: make(chan struct{}),
		done:     make(chan struct{}),
	}
}

func (h *ErrorHandler) Handle(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.errors = append(h.errors, err)

	l.Error("Error handled",
		"error", err,
		"total_errors", len(h.errors),
	)
}

func (h *ErrorHandler) Monitor() {
	defer close(h.done)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.mu.RLock()
			l.Info("Error handler status",
				"total_errors", len(h.errors),
				"last_error", h.getLastError(),
			)
			h.mu.RUnlock()
		case <-h.stopChan:
			return
		}
	}
}

func (h *ErrorHandler) Stop() {
	if h.stopped.Load() {
		return // Already stopped
	}

	h.stopOnce.Do(func() {
		close(h.stopChan)
		h.stopped.Store(true)
	})

	<-h.done // Wait for monitor to finish
}

func (h *ErrorHandler) getLastError() string {
	if len(h.errors) == 0 {
		return "none"
	}
	return h.errors[len(h.errors)-1].Error()
}
