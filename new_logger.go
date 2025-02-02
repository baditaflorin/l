// l/new_logger.go
package l

// NewLogger is a convenience function that creates a new logger
// using the standard factory and the provided configuration.
// It panics if the logger cannot be created.
func NewLogger(cfg Config) Logger {
	logger, err := NewStandardLogger(NewStandardFactory(), cfg)
	if err != nil {
		panic(err)
	}
	// Return the logger as the Logger interface.
	return logger
}
