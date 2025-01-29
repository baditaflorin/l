package metrics

import (
	"encoding/json"
	"github.com/baditaflorin/l"
	"net/http"
	"time"
)

type Reporter struct {
	stopChan chan struct{}
}

func NewReporter() *Reporter {
	return &Reporter{
		stopChan: make(chan struct{}),
	}
}

func (r *Reporter) Start() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.reportMetrics()
		case <-r.stopChan:
			return
		}
	}
}

func (r *Reporter) Stop() {
	close(r.stopChan)
}

func (r *Reporter) reportMetrics() {
	metrics := l.GetMetrics()

	l.Info("Logging metrics",
		"total_messages", metrics.TotalMessages.Load(),
		"error_messages", metrics.ErrorMessages.Load(),
		"dropped_messages", metrics.DroppedMessages.Load(),
	)
}

func (r *Reporter) HandleMetrics(w http.ResponseWriter, req *http.Request) {
	metrics := l.GetMetrics()
	json.NewEncoder(w).Encode(metrics)
}
