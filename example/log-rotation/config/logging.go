package config

import (
	"github.com/baditaflorin/l"
	"path/filepath"
)

func SetupLogging() *l.Logger {
	err := l.Setup(l.Options{
		FilePath:    filepath.Join("logs", "app.log"),
		MaxFileSize: 5 * 1024 * 1024, // 5MB
		MaxBackups:  3,
		JsonFormat:  true,
		AsyncWrite:  true,
		BufferSize:  1024,
	})
	if err != nil {
		panic(err)
	}

	return l
}
