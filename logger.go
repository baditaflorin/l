package l

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrBufferFull      = fmt.Errorf("log buffer full")
	ErrShutdownTimeout = fmt.Errorf("shutdown timeout")
	output             io.Writer // Add this at package level with other vars

)

const (
	maxStackDepth       = 50
	maxErrorRetries     = 3
	flushTimeout        = 5 * time.Second
	maxBufferSize       = 1024 * 1024 // 1MB
	healthCheckInterval = 30 * time.Second
)

// LogMetrics holds logger operational metrics
type LogMetrics struct {
	TotalMessages   atomic.Uint64
	ErrorMessages   atomic.Uint64
	DroppedMessages atomic.Uint64
	LastError       atomic.Pointer[time.Time]
	LastFlush       atomic.Pointer[time.Time]
}

type Options struct {
	Output        io.Writer
	MinLevel      slog.Level
	AddSource     bool
	JsonFormat    bool
	FilePath      string
	MaxFileSize   int64
	MaxBackups    int
	BufferSize    int
	AsyncWrite    bool
	ErrorCallback func(error)
	Metrics       bool
}

var (
	logger     atomic.Pointer[slog.Logger]
	metrics    LogMetrics
	bufferPool sync.Pool
	mu         sync.RWMutex
	flushMu    sync.Mutex // Separate mutex for flush operations
	shutdown   chan struct{}
	wg         sync.WaitGroup
)

type bufferedWriter struct {
	out       io.Writer
	buf       chan []byte
	errCb     func(error)
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	metrics   *LogMetrics
	batchSize int
	closed    atomic.Bool
}

func newBufferedWriter(w io.Writer, bufSize int, errCb func(error), metrics *LogMetrics) *bufferedWriter {
	ctx, cancel := context.WithCancel(context.Background())
	bw := &bufferedWriter{
		out:       w,
		buf:       make(chan []byte, bufSize),
		errCb:     errCb,
		ctx:       ctx,
		cancel:    cancel,
		metrics:   metrics,
		batchSize: 1000,
	}

	// Start worker goroutines
	const numWorkers = 4
	bw.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go bw.writeLoop()
	}

	return bw
}

func (w *bufferedWriter) writeLoop() {
	defer w.wg.Done()

	batch := make([][]byte, 0, w.batchSize)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			// Flush remaining messages before exiting
			w.flushBatch(batch)
			return

		case data, ok := <-w.buf:
			if !ok {
				w.flushBatch(batch)
				return
			}
			batch = append(batch, data)
			if len(batch) >= w.batchSize {
				w.flushBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				w.flushBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

func (w *bufferedWriter) flushBatch(batch [][]byte) {
	if len(batch) == 0 {
		return
	}

	// Combine batched messages
	totalLen := 0
	for _, msg := range batch {
		totalLen += len(msg)
	}

	combined := make([]byte, 0, totalLen)
	for _, msg := range batch {
		combined = append(combined, msg...)
	}

	// Retry logic for writes
	const maxRetries = 3
	var err error
	for i := 0; i < maxRetries; i++ {
		_, err = w.out.Write(combined)
		if err == nil {
			break
		}
		// Exponential backoff
		time.Sleep(time.Millisecond * 10 * time.Duration(1<<uint(i)))
	}

	if err != nil && w.errCb != nil {
		w.errCb(fmt.Errorf("failed to write log batch after %d retries: %w", maxRetries, err))
	}
}

func (w *bufferedWriter) Write(p []byte) (n int, err error) {
	if w.closed.Load() {
		return 0, fmt.Errorf("writer is closed")
	}

	// Make a copy of the data since p might be reused
	data := make([]byte, len(p))
	copy(data, p)

	// Try to write to buffer with timeout
	select {
	case w.buf <- data:
		return len(p), nil
	case <-time.After(100 * time.Millisecond): // Reduced timeout
		if w.metrics != nil {
			w.metrics.DroppedMessages.Add(1)
		}
		// Fall back to synchronous write if buffer is full
		return w.out.Write(p)
	}
}

// recoveryHandler wraps a slog.Handler with panic recovery
type recoveryHandler struct {
	handler slog.Handler
	errCb   func(error)
}

// Handle must have signature: Handle(ctx context.Context, r slog.Record) error
func (h *recoveryHandler) Handle(ctx context.Context, r slog.Record) error {
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			if h.errCb != nil {
				h.errCb(fmt.Errorf("panic in log handler: %v\n%s", err, stack))
			}
		}
	}()
	return h.handler.Handle(ctx, r)
}

// Enabled must have signature: Enabled(ctx context.Context, level slog.Level) bool
func (h *recoveryHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *recoveryHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &recoveryHandler{h.handler.WithAttrs(attrs), h.errCb}
}

func (h *recoveryHandler) WithGroup(name string) slog.Handler {
	return &recoveryHandler{h.handler.WithGroup(name), h.errCb}
}

// Setup configures the logger with enhanced error handling
func Setup(opts Options) error {
	mu.Lock()
	defer mu.Unlock()

	if err := validateOptions(&opts); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	// Close existing logger if present
	if l := logger.Load(); l != nil {
		if err := Close(); err != nil {
			return fmt.Errorf("failed to close existing logger: %w", err)
		}
	}

	var finalOutput io.Writer

	// Setup file output if FilePath is specified, otherwise use default output
	if opts.FilePath != "" {
		file, err := setupLogFile(opts)
		if err != nil {
			return fmt.Errorf("failed to setup log file: %w", err)
		}
		finalOutput = file
	} else {
		finalOutput = opts.Output
	}

	var asyncWriter *bufferedWriter

	if opts.AsyncWrite {
		asyncWriter = newBufferedWriter(finalOutput, opts.BufferSize, opts.ErrorCallback, &metrics)
		finalOutput = asyncWriter
	}

	// Create and store new logger
	handlerOpts := &slog.HandlerOptions{
		Level:     opts.MinLevel,
		AddSource: opts.AddSource,
	}

	var handler slog.Handler
	if opts.JsonFormat {
		handler = slog.NewJSONHandler(finalOutput, handlerOpts)
	} else {
		handler = slog.NewTextHandler(finalOutput, handlerOpts)
	}

	// Wrap with recovery handler
	handler = &recoveryHandler{handler, opts.ErrorCallback}

	if asyncWriter != nil {
		handler = &closingHandler{
			Handler: handler,
			closer:  asyncWriter,
		}
	}

	logger.Store(slog.New(handler))

	// Start health check if metrics are enabled
	if opts.Metrics {
		startHealthCheck()
	}

	return nil
}

func setupLogFile(opts Options) (*os.File, error) {
	dir := filepath.Dir(opts.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(opts.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	wg.Add(1)
	go monitorFileSize(opts, file)

	return file, nil
}

// closingHandler wraps a handler and ensures proper cleanup
type closingHandler struct {
	slog.Handler
	closer io.Closer
}

func (h *closingHandler) Close() error {
	if h.closer != nil {
		return h.closer.Close()
	}
	return nil
}

func validateOptions(opts *Options) error {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}
	if opts.BufferSize == 0 {
		opts.BufferSize = maxBufferSize
	}
	if opts.MaxFileSize == 0 {
		opts.MaxFileSize = 100 * 1024 * 1024 // 100MB
	}
	if opts.MaxBackups == 0 {
		opts.MaxBackups = 5
	}
	return nil
}

func monitorFileSize(opts Options, file *os.File) {
	defer wg.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-shutdown:
			return
		case <-ticker.C:
			if info, err := file.Stat(); err == nil {
				if info.Size() > opts.MaxFileSize {
					rotateLog(opts, file)
				}
			}
		}
	}
}

func rotateLog(opts Options, file *os.File) {
	mu.Lock()
	defer mu.Unlock()

	// Rotate files
	for i := opts.MaxBackups - 1; i > 0; i-- {
		oldPath := fmt.Sprintf("%s.%d", opts.FilePath, i)
		newPath := fmt.Sprintf("%s.%d", opts.FilePath, i+1)
		_ = os.Rename(oldPath, newPath)
	}

	file.Close()
	_ = os.Rename(opts.FilePath, opts.FilePath+".1")

	newFile, err := os.OpenFile(opts.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil && opts.ErrorCallback != nil {
		opts.ErrorCallback(fmt.Errorf("failed to rotate log file: %w", err))
		return
	}

	// Update the logger’s handler to a new file if needed
	if l := logger.Load(); l != nil {
		handler := l.Handler()
		// Example if you used JSONHandler – adapt if you need text logs
		if _, ok := handler.(*slog.JSONHandler); ok {
			newHandler := slog.NewJSONHandler(newFile, &slog.HandlerOptions{
				Level:     slog.LevelInfo,
				AddSource: true,
			})
			wrapped := &recoveryHandler{newHandler, opts.ErrorCallback}
			logger.Store(slog.New(wrapped))
		}
	}
}

func startHealthCheck() {
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(healthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-shutdown:
				return
			case <-ticker.C:
				checkLoggerHealth()
			}
		}
	}()
}

func checkLoggerHealth() {
	if l := logger.Load(); l == nil {
		if err := Setup(Options{
			Output:     os.Stdout,
			MinLevel:   slog.LevelInfo,
			JsonFormat: true,
		}); err != nil && defaultErrorCallback != nil {
			defaultErrorCallback(fmt.Errorf("health check failed to restore logger: %w", err))
		}
	}
}

func defaultErrorCallback(err error) {
	fmt.Fprintf(os.Stderr, "Logger error: %v\n", err)
	metrics.LastError.Store(&time.Time{})
}

func GetMetrics() LogMetrics {
	return metrics
}

func Info(msg string, args ...any) {
	metrics.TotalMessages.Add(1)
	if l := logger.Load(); l != nil {
		l.Info(msg, args...)
	}
}

func Error(msg string, args ...any) {
	metrics.TotalMessages.Add(1)
	metrics.ErrorMessages.Add(1)
	if l := logger.Load(); l != nil {
		// Ensure we have even # of args
		if len(args)%2 == 1 {
			args = append(args, nil)
		}

		// Add stack trace
		stack := make([]uintptr, maxStackDepth)
		length := runtime.Callers(2, stack)
		if length > 0 {
			frames := runtime.CallersFrames(stack[:length])
			var trace []string
			for {
				frame, more := frames.Next()
				trace = append(trace, fmt.Sprintf("%s:%d", frame.Function, frame.Line))
				if !more {
					break
				}
			}
			args = append(args, "stack", trace)
		}

		// Add source location
		if _, file, line, ok := runtime.Caller(1); ok {
			args = append(args, "source", fmt.Sprintf("%s:%d", filepath.Base(file), line))
		}

		l.Error(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	metrics.TotalMessages.Add(1)
	if l := logger.Load(); l != nil {
		l.Warn(msg, args...)
	}
}

func Debug(msg string, args ...any) {
	metrics.TotalMessages.Add(1)
	if l := logger.Load(); l != nil {
		l.Debug(msg, args...)
	}
}

func With(args ...any) *slog.Logger {
	if l := logger.Load(); l != nil {
		return l.With(args...)
	}
	return nil
}

// Flush ensures all pending logs are written
func Flush() error {
	flushMu.Lock()
	defer flushMu.Unlock()

	l := logger.Load()
	if l == nil {
		return nil
	}

	if h, ok := l.Handler().(*closingHandler); ok {
		if closer, ok := h.closer.(*bufferedWriter); ok {
			// Check if there are pending messages
			if len(closer.buf) == 0 {
				return nil
			}

			// Wait for buffered writes to complete with timeout
			timeout := time.After(2 * time.Second)
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-timeout:
					return ErrShutdownTimeout
				case <-ticker.C:
					if len(closer.buf) == 0 {
						return nil
					}
				}
			}
		}
	}
	return nil
}

func (w *bufferedWriter) Close() error {
	if w.closed.Swap(true) {
		return nil // Already closed
	}

	// Signal shutdown
	w.cancel()

	// Create a channel to signal when all writes are complete
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	// Wait for all writers to finish with timeout
	select {
	case <-done:
		return nil
	case <-time.After(2 * time.Second): // Reduced timeout
		return ErrShutdownTimeout
	}
}

func init() {
	shutdown = make(chan struct{})
	bufferPool.New = func() interface{} {
		return make([]byte, 0, 1024)
	}

	// Default initialization
	if err := Setup(Options{
		Output:     os.Stdout,
		MinLevel:   slog.LevelInfo,
		JsonFormat: true,
		Metrics:    true,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		logger.Store(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	}
}

// Close gracefully shuts down the logger and flushes any pending writes
func Close() error {
	flushMu.Lock()
	defer flushMu.Unlock()

	l := logger.Load()
	if l == nil {
		return nil
	}

	// Close the handler if it supports closing
	if h, ok := l.Handler().(*closingHandler); ok {
		if err := h.Close(); err != nil {
			return fmt.Errorf("failed to close handler: %w", err)
		}
	}

	// Clear the logger
	logger.Store(nil)
	return nil
}
