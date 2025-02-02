package l

import "fmt"

// WithError returns a new logger with the error field added.
// If the error is nil, it returns the current logger unchanged.
func (l *StandardLogger) WithError(err error) Logger {
	if err == nil {
		return l
	}

	fields := map[string]interface{}{
		"error": err.Error(),
	}

	// Optionally, handle specific error types (for example, HTTPError)
	// if httpErr, ok := err.(*errors.HTTPError); ok {
	//     fields["status_code"] = httpErr.StatusCode
	//     fields["retryable"] = httpErr.Retryable
	//     if len(httpErr.Body) > 0 {
	//         fields["error_body"] = string(httpErr.Body)
	//     }
	// }

	// Add the error type.
	fields["error_type"] = fmt.Sprintf("%T", err)

	// If you want to add a stack trace in development mode, for example:
	if l.config.Environment != "production" {
		fields["stack_trace"] = formatStackTraceFromLines(captureStackTrace(2))
	}

	return l.WithFields(fields)
}
