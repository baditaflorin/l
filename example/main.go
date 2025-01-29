package main

import (
	"fmt"
	"os"

	"github.com/baditaflorin/l"
)

func main() {
	// Initialize the logger
	err := l.Setup(l.Options{
		Output:     os.Stdout,
		MinLevel:   0,
		JsonFormat: true,
	})
	if err != nil {
		fmt.Println("Logger setup failed:", err)
		return
	}

	// Log some messages
	l.Info("This is an info log", "module", "example")
	l.Error("This is an error log", "code", 500)
}
