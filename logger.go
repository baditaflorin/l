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
	once       sync.Once
	shutdown   chan struct{}
	wg         sync.WaitGroup
)

type bufferedWriter struct {
	out    io.Writer
	buf    chan []byte
	errCb  func(error)
	ctx    context.Context
	cancel context.CancelFunc
}

func newBufferedWriter(w io.Writer, bufSize int, errCb func(error)) *bufferedWriter {
	ctx, cancel := context.WithCancel(context.Background())
	bw := &bufferedWriter{
		out:    w,
		buf:    make(chan []byte, bufSize),
		errCb:  errCb,
		ctx:    ctx,
		cancel: cancel,
	}

	wg.Add(1)
	go bw.writeLoop()
	return bw
}

func (w *bufferedWriter) writeLoop() {
	defer wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			// Drain remaining messages before exit
			for len(w.buf) > 0 {
				data := <-w.buf
				_, err := w.out.Write(data)
				if err != nil && w.errCb != nil {
					w.errCb(fmt.Errorf("error writing log: %w", err))
				}
			}
			return
		case data, ok := <-w.buf:
			if !ok {
				return
			}
			_, err := w.out.Write(data)
			if err != nil && w.errCb != nil {
				w.errCb(fmt.Errorf("error writing log: %w", err))
			}
		}
	}
}

func (w *bufferedWriter) Write(p []byte) (n int, err error) {
	select {
	case w.buf <- append([]byte(nil), p...):
		return len(p), nil
	default:
		metrics.DroppedMessages.Add(1)
		return 0, fmt.Errorf("log buffer full")
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

	var output io.Writer = opts.Output
	if opts.FilePath != "" {
		file, err := setupLogFile(opts)
		if err != nil {
			return err
		}
		output = file
	}

	// Setup buffered/async writing if requested
	if opts.AsyncWrite {
		bw := newBufferedWriter(output, opts.BufferSize, opts.ErrorCallback)
		output = bw
	}

	handlerOpts := &slog.HandlerOptions{
		Level:     opts.MinLevel,
		AddSource: opts.AddSource,
	}

	var handler slog.Handler
	if opts.JsonFormat {
		handler = slog.NewJSONHandler(output, handlerOpts)
	} else {
		handler = slog.NewTextHandler(output, handlerOpts)
	}

	// Wrap with recovery handler
	handler = &recoveryHandler{handler, opts.ErrorCallback}

	newLogger := slog.New(handler)
	logger.Store(newLogger)

	if opts.Metrics {
		startHealthCheck()
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

func Flush() error {
	mu.Lock()

	Info("Flushing logs before exit")

	// Copy the current logger pointer
	l := logger.Load()

	// Unlock before performing potentially long operations
	mu.Unlock()

	if l == nil {
		return nil
	}

	// Ensure all buffered logs are written
	time.Sleep(200 * time.Millisecond)

	// Check if handler supports flushing
	if handler, ok := l.Handler().(interface{ Flush() error }); ok {
		if err := handler.Flush(); err != nil {
			return fmt.Errorf("failed to flush logs: %w", err)
		}
	}

	return nil
}

func Close() error {
	mu.Lock()

	// Close shutdown channel to signal all goroutines to exit
	if shutdown != nil {
		close(shutdown)
	}

	mu.Unlock()

	// Give some time for goroutines to finish their work
	time.Sleep(200 * time.Millisecond)

	// Ensure all pending logs are flushed
	if err := Flush(); err != nil {
		return err
	}

	// Wait for goroutines to exit, but add a timeout to prevent deadlocks
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Successfully waited for all goroutines to finish
	case <-time.After(5 * time.Second):
		fmt.Println("Warning: Timeout while waiting for goroutines to exit")
	}

	mu.Lock()
	defer mu.Unlock()

	if l := logger.Load(); l != nil {
		if closer, ok := l.Handler().(io.Closer); ok {
			return closer.Close()
		}
	}

	return nil
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
