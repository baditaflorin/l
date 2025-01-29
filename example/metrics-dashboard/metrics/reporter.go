package metrics

import (
	"encoding/json"
	"github.com/baditaflorin/l"
	"net/http"
	"time"
)

type Reporter struct {
	logger           l.Logger
	metricsCollector l.MetricsCollector
	stopChan         chan struct{}
}

func NewReporter(factory l.Factory, config l.Config) (*Reporter, error) {
	logger, err := factory.CreateLogger(config)
	if err != nil {
		return nil, err
	}

	metricsCollector, err := factory.CreateMetricsCollector(config)
	if err != nil {
		return nil, err
	}

	return &Reporter{
		logger:           logger,
		metricsCollector: metricsCollector,
		stopChan:         make(chan struct{}),
	}, nil
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
	metrics := r.metricsCollector.GetMetrics()

	r.logger.Info("Logging metrics",
		"total_messages", metrics.TotalMessages,
		"error_messages", metrics.ErrorMessages,
		"dropped_messages", metrics.DroppedMessages,
	)
}

func (r *Reporter) HandleMetrics(w http.ResponseWriter, req *http.Request) {
	metrics := r.metricsCollector.GetMetrics()
	json.NewEncoder(w).Encode(metrics)
}
