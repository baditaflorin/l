package main

import (
	"./config"
	"github.com/baditaflorin/l"
	"time"
)

func main() {
	logger := config.SetupLogging()
	defer logger.Close()

	// Generate lots of logs
	for i := 0; i < 1000000; i++ {
		l.Info("Application metric",
			"iteration", i,
			"timestamp", time.Now().Unix(),
			"data", generateLargeString(100),
		)

		if i%1000 == 0 {
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func generateLargeString(size int) string {
	// Generate dummy data
	data := make([]byte, size)
	for i := range data {
		data[i] = 'A' + byte(i%26)
	}
	return string(data)
}
