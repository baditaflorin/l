package l

import "log/slog"

// Level is an alias for slog.Level.
type Level = slog.Level

// Predefined log levels.
const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)
