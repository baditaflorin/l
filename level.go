package l

import "strings"

// ParseLevel converts a string (case-insensitive) to the corresponding log Level.
// Supported values are "debug", "warn", "error"; all others default to LevelInfo.
func ParseLevel(levelStr string) Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return LevelDebug
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}
