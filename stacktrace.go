package l

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// StackTraceLine represents a single line in a stack trace.
type StackTraceLine struct {
	Function string
	File     string
	Line     int
}

// captureStackTrace returns the program counters for the current goroutine,
// skipping the first `skip` frames.
func captureStackTrace(skip int) []uintptr {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])
	return pcs[:n]
}

// formatStackTraceFromLines takes a slice of program counters and converts them
// into a single string with one line per frame.
func formatStackTraceFromLines(pcs []uintptr) string {
	var trace []string
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		// Use the base file name for brevity.
		file := filepath.Base(frame.File)
		trace = append(trace, fmt.Sprintf("%s:%d %s", file, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	return strings.Join(trace, "\n")
}
