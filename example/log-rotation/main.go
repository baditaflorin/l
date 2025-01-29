package main

import (
	"github.com/baditaflorin/l"
	"github.com/baditaflorin/l/example/log-rotation/config"
	"time"
)

func main() {
	factory := l.NewStandardFactory()
	logger, err := config.SetupLogging(factory)
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// Generate lots of logs
	for i := 0; i < 1000000; i++ {
		logger.Info("Application metric",
			"iteration", i,
			"timestamp", time.Now().Unix(),
			"data", generateLargeString(100),
		)

		if i%1000 == 0 {
			time.Sleep(time.Millisecond * 100)
		}
	}

	if err := logger.Flush(); err != nil {
		panic(err)
	}
}

func generateLargeString(size int) string {
	data := make([]byte, size)
	for i := range data {
		data[i] = 'A' + byte(i%26)
	}
	return string(data)
}
