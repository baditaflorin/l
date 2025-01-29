package main

import (
	"github.com/baditaflorin/l"
	"github.com/baditaflorin/l/example/error-handling/handlers"
	"os"
	"sync"
	"time"
)

func main() {
	errorHandler := handlers.NewErrorHandler()

	err := l.Setup(l.Options{
		Output:        os.Stdout,
		JsonFormat:    true,
		ErrorCallback: errorHandler.Handle,
		Metrics:       true,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	// Use WaitGroup to coordinate shutdown
	var wg sync.WaitGroup
	wg.Add(2) // One for Monitor, one for simulateErrors

	// Start error monitoring
	go func() {
		defer wg.Done()
		errorHandler.Monitor()
	}()

	// Run error simulation
	go func() {
		defer wg.Done()
		simulateErrors()
		errorHandler.Stop() // Stop monitoring after scenarios complete
	}()

	// Wait for all goroutines to complete
	wg.Wait()
}

func simulateErrors() {
	scenarios := []struct {
		name string
		fn   func()
	}{
		{"panic", func() { panic("simulated panic") }},
		{"nil_pointer", func() { var p *int; _ = *p }},
		{"division_by_zero", func() { panic("division by zero attempted") }},
	}

	for _, scenario := range scenarios {
		l.Info("Running error scenario", "name", scenario.name)

		func() {
			defer func() {
				if r := recover(); r != nil {
					l.Error("Scenario failed",
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
