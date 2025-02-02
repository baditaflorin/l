package l

import "context"

// LogError is a convenience method that logs an error with enhanced context.
// It simply adds the error (via WithError) and then logs it at error level.
func (l *StandardLogger) LogError(ctx context.Context, err error) {
	l.WithError(err).Error(err.Error())
}
