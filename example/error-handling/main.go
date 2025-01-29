package main

import (
	"github.com/baditaflorin/l"
	"github.com/baditaflorin/l/example/error-handling/handlers"
	"os"
	"sync"
	"time"
)

func main() {
	// Create factory and config
	factory := l.NewStandardFactory()
	config := l.Config{
		Output:     os.Stdout,
		JsonFormat: true,
		AddSource:  true,
		Metrics:    true,
	}

	errorHandler, err := handlers.NewErrorHandler(factory, config)
	if err != nil {
		panic(err)
	}

	// Create logger
	logger, err := factory.CreateLogger(config)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// Use WaitGroup to coordinate shutdown
	var wg sync.WaitGroup
	wg.Add(2)

	// Start error monitoring
	go func() {
		defer wg.Done()
		errorHandler.Monitor()
	}()

	// Run error simulation
	go func() {
		defer wg.Done()
		simulateErrors(logger)
		errorHandler.Stop()
	}()

	wg.Wait()
}

func simulateErrors(logger l.Logger) {
	scenarios := []struct {
		name string
		fn   func()
	}{
		{"panic", func() { panic("simulated panic") }},
		{"nil_pointer", func() { var p *int; _ = *p }},
		{"division_by_zero", func() { panic("division by zero attempted") }},
	}

	for _, scenario := range scenarios {
		logger.Info("Running error scenario", "name", scenario.name)

		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("Scenario failed",
						"name", scenario.name,
						"error", r,
					)
				}
			}()

			scenario.fn()
		}()

		time.Sleep(time.Second)
	}
}
