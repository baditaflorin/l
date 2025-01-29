package config

import (
	"github.com/baditaflorin/l"
	"path/filepath"
)

func SetupLogging(factory l.Factory) (l.Logger, error) {
	config := l.Config{
		FilePath:    filepath.Join("logs", "app.log"),
		MaxFileSize: 1 * 1024 * 1024, // 5MB
		MaxBackups:  3,
		JsonFormat:  true,
		AsyncWrite:  true,
		BufferSize:  1024,
	}

	return factory.CreateLogger(config)
}
