package l

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var ErrBufferFull = fmt.Errorf("log buffer full")

// StandardLogger implements the Logger interface
type StandardLogger struct {
	handler     HandlerWrapper
	metrics     MetricsCollector
	errHandler  ErrorHandler
	healthCheck HealthChecker
	writer      Writer
	config      Config
	ctx         context.Context
	cancel      context.CancelFunc
	fields      map[string]interface{}
	mu          sync.RWMutex
	asyncWriter *BufferedWriter
	isAsync     bool
	closed      atomic.Bool // Add closed field
}

// NewStandardLogger creates a new logger instance using the provided factory
func NewStandardLogger(factory Factory, config Config) (Logger, error) {
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Ensure we have an output writer
	if config.Output == nil {
		config.Output = os.Stdout
	}

	logger := &StandardLogger{
		config:  config,
		ctx:     ctx,
		cancel:  cancel,
		fields:  make(map[string]interface{}),
		isAsync: config.AsyncWrite,
	}

	// Create metrics collector first
	metrics, err := factory.CreateMetricsCollector(config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create metrics collector: %w", err)
	}
	logger.metrics = metrics

	// Set up writer
	var baseWriter Writer
	if config.AsyncWrite {
		bufferedWriter := NewBufferedWriter(config.Output, config.BufferSize)
		logger.asyncWriter = bufferedWriter
		baseWriter = bufferedWriter
	} else {
		baseWriter, err = factory.CreateWriter(config)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create writer: %w", err)
		}
	}
	logger.writer = baseWriter

	// Create and configure the handler
	var baseHandler slog.Handler
	handlerOpts := &slog.HandlerOptions{
		Level:     config.MinLevel,
		AddSource: config.AddSource,
	}

	if config.JsonFormat {
		baseHandler = slog.NewJSONHandler(logger.writer, handlerOpts)
	} else {
		baseHandler = slog.NewTextHandler(logger.writer, handlerOpts)
	}

	// Create error handler
	errHandler, err := factory.CreateErrorHandler(config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create error handler: %w", err)
	}
	logger.errHandler = errHandler

	// Create handler wrapper
	logger.handler = factory.CreateHandlerWrapper(baseHandler, logger.writer)

	return logger, nil
}

// validateConfig validates the logger configuration
func validateConfig(config *Config) error {
	if config.BufferSize <= 0 {
		config.BufferSize = 1024 * 1024 // Set default buffer size to 1MB
	}
	if config.MaxFileSize < 0 {
		return fmt.Errorf("max file size must be non-negative")
	}
	if config.MaxBackups < 0 {
		return fmt.Errorf("max backups must be non-negative")
	}
	return nil
}

// Logger interface implementation
func (l *StandardLogger) Info(msg string, args ...any) {
	l.log(slog.LevelInfo, msg, args...)
}

func (l *StandardLogger) Error(msg string, args ...any) {
	args = l.enrichErrorArgs(args...)
	l.log(slog.LevelError, msg, args...)
}

func (l *StandardLogger) Warn(msg string, args ...any) {
	l.log(slog.LevelWarn, msg, args...)
}

func (l *StandardLogger) Debug(msg string, args ...any) {
	l.log(slog.LevelDebug, msg, args...)
}

// Update the log method in StandardLogger to ensure metrics are incremented correctly
func (l *StandardLogger) log(level slog.Level, msg string, args ...any) {
	if l.closed.Load() {
		return
	}

	// Use atomic operations for metrics
	if l.metrics != nil {
		l.metrics.IncrementTotal()
		if level == slog.LevelError {
			l.metrics.IncrementErrors()
			l.metrics.SetLastError(time.Now())
		}
	}

	// Create record
	r := slog.Record{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
	}

	// Add fields
	l.mu.RLock()
	for k, v := range l.fields {
		r.Add(slog.Any(k, v))
	}
	l.mu.RUnlock()

	// Add args
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			r.Add(slog.Any(key, args[i+1]))
		}
	}

	// Handle the log
	err := l.errHandler.WithRecovery(func() error {
		if l.isAsync {
			go func() {
				if err := l.handler.Handle(l.ctx, r); err != nil {
					l.errHandler.Handle(err)
				}
			}()
			return nil
		}
		return l.handler.Handle(l.ctx, r)
	})

	if err != nil {
		l.errHandler.Handle(err)
	}
}

func (l *StandardLogger) With(args ...any) Logger {
	newLogger := &StandardLogger{
		handler:     l.handler,
		metrics:     l.metrics,
		errHandler:  l.errHandler,
		healthCheck: l.healthCheck,
		writer:      l.writer,
		config:      l.config,
		ctx:         l.ctx,
		cancel:      l.cancel,
		fields:      make(map[string]interface{}, len(l.fields)),
	}

	// Copy existing fields
	l.mu.RLock()
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	l.mu.RUnlock()

	// Add new fields
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			newLogger.fields[key] = args[i+1]
		}
	}

	return newLogger
}

func (l *StandardLogger) enrichErrorArgs(args ...any) []any {
	// Add stack trace
	stack := make([]uintptr, 50)
	length := runtime.Callers(3, stack)
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
	if _, file, line, ok := runtime.Caller(2); ok {
		args = append(args, "source", fmt.Sprintf("%s:%d", filepath.Base(file), line))
	}

	return args
}

func (l *StandardLogger) Flush() error {
	if flusher, ok := l.writer.(Flushable); ok {
		return flusher.Flush()
	}
	return nil
}

func (l *StandardLogger) Close() error {
	if !l.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	l.cancel() // Cancel context

	if l.healthCheck != nil {
		l.healthCheck.Stop()
	}

	// Flush before closing
	if err := l.Flush(); err != nil {
		return fmt.Errorf("flush failed during close: %w", err)
	}

	// Close async writer if it exists
	if l.isAsync && l.asyncWriter != nil {
		if err := l.asyncWriter.Close(); err != nil {
			return fmt.Errorf("async writer close failed: %w", err)
		}
	}

	return nil
}

// Implement ContextProvider interface
func (l *StandardLogger) WithContext(ctx context.Context) Logger {
	newLogger := &StandardLogger{
		handler:     l.handler,
		metrics:     l.metrics,
		errHandler:  l.errHandler,
		healthCheck: l.healthCheck,
		writer:      l.writer,
		config:      l.config,
		fields:      l.fields,
		ctx:         ctx,
		cancel:      l.cancel,
		isAsync:     l.isAsync,
		asyncWriter: l.asyncWriter,
	}
	return newLogger
}

func (l *StandardLogger) GetContext() context.Context {
	return l.ctx
}

// Implement FieldProvider interface
func (l *StandardLogger) WithFields(fields map[string]interface{}) Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return l.With(args...)
}

func (l *StandardLogger) GetFields() map[string]interface{} {
	l.mu.RLock()
	defer l.mu.RUnlock()

	fields := make(map[string]interface{}, len(l.fields))
	for k, v := range l.fields {
		fields[k] = v
	}
	return fields
}

// StandardFactory implements the Factory interface
type StandardFactory struct {
	mu               sync.RWMutex
	metricsCollector MetricsCollector
}

func NewStandardFactory() Factory {
	return &StandardFactory{
		metricsCollector: NewStandardMetricsCollector(),
	}
}

// Modify the CreateLogger method in StandardFactory
func (f *StandardFactory) CreateLogger(config Config) (Logger, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Use the factory's metrics collector
	metrics := f.metricsCollector

	logger := &StandardLogger{
		config:  config,
		metrics: metrics,
		fields:  make(map[string]interface{}),
		isAsync: config.AsyncWrite,
	}

	ctx, cancel := context.WithCancel(context.Background())
	logger.ctx = ctx
	logger.cancel = cancel

	// Create and configure the writer
	var baseWriter Writer
	if config.AsyncWrite {
		bufferedWriter := NewBufferedWriter(config.Output, config.BufferSize)
		logger.asyncWriter = bufferedWriter
		baseWriter = bufferedWriter
	} else {
		var err error
		baseWriter, err = f.CreateWriter(config)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create writer: %w", err)
		}
	}
	logger.writer = baseWriter

	// Create error handler
	errHandler, err := f.CreateErrorHandler(config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create error handler: %w", err)
	}
	logger.errHandler = errHandler

	// Create base slog.Handler
	var baseHandler slog.Handler
	if config.JsonFormat {
		baseHandler = slog.NewJSONHandler(logger.writer, &slog.HandlerOptions{
			Level:     config.MinLevel,
			AddSource: config.AddSource,
		})
	} else {
		baseHandler = slog.NewTextHandler(logger.writer, &slog.HandlerOptions{
			Level:     config.MinLevel,
			AddSource: config.AddSource,
		})
	}

	logger.handler = f.CreateHandlerWrapper(baseHandler, logger.writer)

	return logger, nil
}

func (f *StandardFactory) CreateWriter(config Config) (Writer, error) {
	if config.FilePath != "" {
		return NewFileWriter(config)
	}
	if config.AsyncWrite {
		return NewBufferedWriter(config.Output, config.BufferSize), nil
	}
	return NewDirectWriter(config.Output), nil
}

func (f *StandardFactory) CreateFormatter(config Config) (Formatter, error) {
	if config.JsonFormat {
		return NewJSONFormatter(), nil
	}
	return NewTextFormatter(), nil
}

func (f *StandardFactory) CreateMetricsCollector(config Config) (MetricsCollector, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.metricsCollector, nil
}

func (f *StandardFactory) CreateRotationManager(config Config) (RotationManager, error) {
	return NewStandardRotationManager(config), nil
}

func (f *StandardFactory) CreateErrorHandler(config Config) (ErrorHandler, error) {
	return NewStandardErrorHandler(config.ErrorCallback), nil
}

func (f *StandardFactory) CreateBufferManager(config Config) (BufferManager, error) {
	return NewStandardBufferManager(config.BufferSize), nil
}

func (f *StandardFactory) CreateHandlerWrapper(handler slog.Handler, closer io.Closer) HandlerWrapper {
	return NewStandardHandlerWrapper(handler, closer)
}

func (f *StandardFactory) CreateHealthChecker(logger Logger) (HealthChecker, error) {
	return NewStandardHealthChecker(logger), nil
}

// Writer implementations

// DirectWriter implements Writer for synchronous writes
type DirectWriter struct {
	out io.Writer
	mu  sync.Mutex
}

func NewDirectWriter(out io.Writer) *DirectWriter {
	if out == nil {
		out = os.Stdout
	}
	return &DirectWriter{out: out}
}

func (w *DirectWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.out.Write(p)
}

func (w *DirectWriter) Close() error {
	if closer, ok := w.out.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (w *DirectWriter) Flush() error {
	if flusher, ok := w.out.(Flushable); ok {
		return flusher.Flush()
	}
	return nil
}

// BufferedWriter implements Writer with async buffering
type BufferedWriter struct {
	out       io.Writer
	buf       chan []byte
	done      chan struct{}
	wg        sync.WaitGroup
	batchSize int
	closed    atomic.Bool
	flushChan chan chan struct{}
	mu        sync.Mutex // Add mutex for synchronization
}

func NewBufferedWriter(out io.Writer, bufferSize int) *BufferedWriter {
	if out == nil {
		out = os.Stdout // Use stdout as default output
	}

	if bufferSize <= 0 {
		bufferSize = 1024 * 1024 // 1MB default
	}

	w := &BufferedWriter{
		out:       out,
		buf:       make(chan []byte, bufferSize),
		done:      make(chan struct{}),
		batchSize: 1000,
		flushChan: make(chan chan struct{}, 1),
		mu:        sync.Mutex{},
	}

	w.wg.Add(1)
	go w.writeLoop()

	return w
}

func (w *BufferedWriter) writeLoop() {
	defer w.wg.Done()

	batch := make([][]byte, 0, w.batchSize)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	flush := func() {
		if len(batch) > 0 {
			w.flushBatch(batch)
			batch = batch[:0]
		}
	}

	for {
		select {
		case <-w.done:
			flush()
			return

		case data, ok := <-w.buf:
			if !ok {
				flush()
				return
			}
			batch = append(batch, data)
			if len(batch) >= w.batchSize {
				flush()
			}

		case respChan := <-w.flushChan:
			flush()
			respChan <- struct{}{}

		case <-ticker.C:
			flush()
		}
	}
}

func (w *BufferedWriter) flushBatch(batch [][]byte) {
	if len(batch) == 0 {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.out == nil {
		return
	}

	// Calculate total length needed
	totalLen := 0
	for _, msg := range batch {
		if msg != nil {
			totalLen += len(msg)
		}
	}

	// Create a single buffer for all messages
	combined := make([]byte, 0, totalLen)
	for _, msg := range batch {
		if msg != nil {
			combined = append(combined, msg...)
		}
	}

	// Write everything at once if we have data
	if len(combined) > 0 {
		w.out.Write(combined)
	}
}

// Update the Write method in BufferedWriter to handle large messages better
func (w *BufferedWriter) Write(p []byte) (n int, err error) {
	if w.closed.Load() {
		return 0, fmt.Errorf("writer is closed")
	}

	// Ensure we have a valid writer
	w.mu.Lock()
	if w.out == nil {
		w.mu.Unlock()
		return 0, fmt.Errorf("no output writer available")
	}
	w.mu.Unlock()

	// Create a copy of the data to prevent modification
	data := make([]byte, len(p))
	copy(data, p)

	// If message is larger than buffer capacity, write directly
	if len(data) > cap(w.buf) {
		w.mu.Lock()
		defer w.mu.Unlock()
		return w.out.Write(data)
	}

	// Try to write to buffer with timeout
	select {
	case w.buf <- data:
		return len(p), nil
	default:
		// Buffer is full, write directly
		w.mu.Lock()
		defer w.mu.Unlock()
		return w.out.Write(data)
	}
}

func (w *BufferedWriter) Close() error {
	if w.closed.Swap(true) {
		return nil
	}

	// Signal shutdown
	close(w.done)

	// Wait for writeLoop to finish
	w.wg.Wait()

	// Close underlying writer if it supports it
	if closer, ok := w.out.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (w *BufferedWriter) Flush() error {
	if w.closed.Load() {
		return fmt.Errorf("writer is closed")
	}

	respChan := make(chan struct{})

	// Send flush request
	select {
	case w.flushChan <- respChan:
		// Wait for flush to complete with a longer timeout
		select {
		case <-respChan:
			// Wait for a small duration to ensure writes are processed
			time.Sleep(10 * time.Millisecond)
			return nil
		case <-time.After(2 * time.Second):
			return fmt.Errorf("flush timeout")
		}
	case <-time.After(500 * time.Millisecond):
		return fmt.Errorf("flush channel busy")
	}
}

// Metrics implementation
type StandardMetricsCollector struct {
	totalMessages   atomic.Uint64
	errorMessages   atomic.Uint64
	droppedMessages atomic.Uint64
	lastError       atomic.Pointer[time.Time]
	lastFlush       atomic.Pointer[time.Time]
}

func NewStandardMetricsCollector() *StandardMetricsCollector {
	m := &StandardMetricsCollector{}
	now := time.Now()
	m.lastError.Store(&now)
	m.lastFlush.Store(&now)
	return m
}

func (m *StandardMetricsCollector) IncrementTotal()   { m.totalMessages.Add(1) }
func (m *StandardMetricsCollector) IncrementErrors()  { m.errorMessages.Add(1) }
func (m *StandardMetricsCollector) IncrementDropped() { m.droppedMessages.Add(1) }

func (m *StandardMetricsCollector) SetLastError(t time.Time) {
	m.lastError.Store(&t)
}

func (m *StandardMetricsCollector) SetLastFlush(t time.Time) {
	m.lastFlush.Store(&t)
}

func (m *StandardMetricsCollector) GetMetrics() Metrics {
	return Metrics{
		TotalMessages:   m.totalMessages.Load(),
		ErrorMessages:   m.errorMessages.Load(),
		DroppedMessages: m.droppedMessages.Load(),
		LastError:       *m.lastError.Load(),
		LastFlush:       *m.lastFlush.Load(),
	}
}

func (m *StandardMetricsCollector) Reset() {
	m.totalMessages.Store(0)
	m.errorMessages.Store(0)
	m.droppedMessages.Store(0)
	now := time.Now()
	m.lastError.Store(&now)
	m.lastFlush.Store(&now)
}

// Error Handler implementation
type StandardErrorHandler struct {
	callback func(error)
	mu       sync.RWMutex
	errors   []error
}

func NewStandardErrorHandler(callback func(error)) *StandardErrorHandler {
	if callback == nil {
		callback = func(err error) {
			fmt.Fprintf(os.Stderr, "Logger error: %v\n", err)
		}
	}
	return &StandardErrorHandler{
		callback: callback,
		errors:   make([]error, 0),
	}
}

func (h *StandardErrorHandler) Handle(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.errors = append(h.errors, err)
	if h.callback != nil {
		h.callback(err)
	}
}

func (h *StandardErrorHandler) WithRecovery(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %v", r)
			h.Handle(err)
		}
	}()
	return fn()
}

// Health Checker implementation
type StandardHealthChecker struct {
	logger   Logger
	stopChan chan struct{}
	stopped  atomic.Bool
	wg       sync.WaitGroup
	mu       sync.Mutex
}

func NewStandardHealthChecker(logger Logger) *StandardHealthChecker {
	return &StandardHealthChecker{
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

func (h *StandardHealthChecker) Start(ctx context.Context) {
	h.mu.Lock()
	if h.stopped.Load() {
		h.stopChan = make(chan struct{})
		h.stopped.Store(false)
	}
	h.mu.Unlock()

	h.wg.Add(1)
	go h.checkLoop(ctx)
}

func (h *StandardHealthChecker) checkLoop(ctx context.Context) {
	defer h.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopChan:
			return
		case <-ticker.C:
			if err := h.Check(); err != nil {
				if h.logger != nil {
					h.logger.Error("Health check failed", "error", err)
				}
			}
		}
	}
}

func (h *StandardHealthChecker) Check() error {
	// Implement health checks here
	return nil
}

func (h *StandardHealthChecker) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.stopped.Load() {
		close(h.stopChan)
		h.stopped.Store(true)
		h.wg.Wait()
	}
}

// Handler Wrapper implementation
type StandardHandlerWrapper struct {
	handler slog.Handler
	closer  io.Closer
}

func NewStandardHandlerWrapper(handler slog.Handler, closer io.Closer) *StandardHandlerWrapper {
	return &StandardHandlerWrapper{
		handler: handler,
		closer:  closer,
	}
}

func (h *StandardHandlerWrapper) Handle(ctx context.Context, r slog.Record) error {
	return h.handler.Handle(ctx, r)
}

func (h *StandardHandlerWrapper) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *StandardHandlerWrapper) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.handler.WithAttrs(attrs)
}

func (h *StandardHandlerWrapper) WithGroup(name string) slog.Handler {
	return h.handler.WithGroup(name)
}

func (h *StandardHandlerWrapper) Close() error {
	if h.closer != nil {
		return h.closer.Close()
	}
	return nil
}

// FileWriter implements Writer for file-based logging
type FileWriter struct {
	file     *os.File
	mu       sync.Mutex
	rotation RotationManager
}

func NewFileWriter(config Config) (*FileWriter, error) {
	dir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	rotation := NewStandardRotationManager(config)

	return &FileWriter{
		file:     file,
		rotation: rotation,
	}, nil
}

func (w *FileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if info, err := w.file.Stat(); err == nil {
		if w.rotation.ShouldRotate(info.Size()) {
			if err := w.rotation.Rotate(); err != nil {
				return 0, fmt.Errorf("rotation failed: %w", err)
			}
		}
	}

	return w.file.Write(p)
}

func (w *FileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

func (w *FileWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Sync()
}

// Formatters
type JSONFormatter struct{}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

func (f *JSONFormatter) Format(record LogRecord) ([]byte, error) {
	data := make(map[string]interface{})

	// Add base fields
	data["time"] = record.Time.Format(time.RFC3339Nano)
	data["level"] = record.Level.String()
	data["msg"] = record.Message

	// Add args as key-value pairs
	for i := 0; i < len(record.Args)-1; i += 2 {
		if key, ok := record.Args[i].(string); ok {
			data[key] = record.Args[i+1]
		}
	}

	// Process slog.Attr entries
	for _, attr := range record.Attrs {
		// Handle group attributes
		if attr.Value.Kind() == slog.KindGroup {
			groupData := make(map[string]interface{})
			for _, groupAttr := range attr.Value.Group() {
				groupData[groupAttr.Key] = groupAttr.Value.Any()
			}
			data[attr.Key] = groupData
		} else {
			data[attr.Key] = attr.Value.Any()
		}
	}

	return json.Marshal(data)
}

func (f *JSONFormatter) WithOptions(opts FormatterOptions) Formatter {
	return f // JSON formatter ignores options
}

type TextFormatter struct {
	opts FormatterOptions
}

func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		opts: FormatterOptions{
			TimeFormat: time.RFC3339,
			UseColor:   false,
			Indent:     "  ",
		},
	}
}

func (f *TextFormatter) Format(record LogRecord) ([]byte, error) {
	var result []byte

	// Add indentation if specified
	if f.opts.Indent != "" {
		result = append(result, f.opts.Indent...)
	}

	result = append(result, record.Time.Format(f.opts.TimeFormat)...)

	// Add level with optional color
	result = append(result, " ["...)
	if f.opts.UseColor {
		result = append(result, f.getLevelColor(record.Level)...)
		result = append(result, record.Level.String()...)
		result = append(result, "\x1b[0m"...) // Reset color
	} else {
		result = append(result, record.Level.String()...)
	}
	result = append(result, "] "...)

	// Add source information if available and AddSource is true
	if record.AddSource && record.Source != "" {
		result = append(result, record.Source...)
		result = append(result, " "...)
	}

	// Add message
	result = append(result, record.Message...)

	// Add args as key=value pairs
	for i := 0; i < len(record.Args)-1; i += 2 {
		if key, ok := record.Args[i].(string); ok {
			result = append(result, ' ')
			result = append(result, key...)
			result = append(result, '=')
			result = append(result, fmt.Sprint(record.Args[i+1])...)
		}
	}

	// Add newline
	result = append(result, '\n')

	return result, nil
}

func (f *TextFormatter) getLevelColor(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "\x1b[36m" // Cyan
	case slog.LevelInfo:
		return "\x1b[32m" // Green
	case slog.LevelWarn:
		return "\x1b[33m" // Yellow
	case slog.LevelError:
		return "\x1b[31m" // Red
	default:
		return "\x1b[37m" // White
	}
}

func (f *TextFormatter) WithOptions(opts FormatterOptions) Formatter {
	newFormatter := &TextFormatter{
		opts: opts,
	}
	return newFormatter
}

// StandardRotationManager implements RotationManager
type StandardRotationManager struct {
	config     Config
	mu         sync.Mutex
	currentLog string
}

func NewStandardRotationManager(config Config) *StandardRotationManager {
	return &StandardRotationManager{
		config:     config,
		currentLog: config.FilePath,
	}
}

func (r *StandardRotationManager) ShouldRotate(size int64) bool {
	return size > r.config.MaxFileSize
}

func (r *StandardRotationManager) Rotate() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// First check if the current log file exists
	currentStat, err := os.Stat(r.config.FilePath)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to stat current log file: %w", err)
	}

	// Create logs directory if it doesn't exist
	dir := filepath.Dir(r.config.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Read the current file permissions to preserve them
	currentPerms := currentStat.Mode()

	// Find the oldest existing backup
	lastIndex := 0
	for i := 1; i <= r.config.MaxBackups; i++ {
		backupPath := fmt.Sprintf("%s.%d", r.config.FilePath, i)
		if _, err := os.Stat(backupPath); err == nil {
			lastIndex = i
		}
	}

	// If we've reached max backups, remove the oldest one
	if lastIndex == r.config.MaxBackups {
		oldestBackup := fmt.Sprintf("%s.%d", r.config.FilePath, r.config.MaxBackups)
		if err := os.Remove(oldestBackup); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove oldest backup: %w", err)
		}
		lastIndex--
	}

	// Shift all existing backups
	for i := lastIndex; i > 0; i-- {
		oldPath := fmt.Sprintf("%s.%d", r.config.FilePath, i)
		newPath := fmt.Sprintf("%s.%d", r.config.FilePath, i+1)

		// Only try to rename if the source file exists
		if _, err := os.Stat(oldPath); err == nil {
			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("failed to rotate backup %d: %w", i, err)
			}
		}
	}

	// Copy current log to .1 instead of moving it
	newPath := r.config.FilePath + ".1"
	if err := copyFile(r.config.FilePath, newPath); err != nil {
		return fmt.Errorf("failed to copy current log: %w", err)
	}

	// Truncate the current log file but keep it open
	if err := os.Truncate(r.config.FilePath, 0); err != nil {
		return fmt.Errorf("failed to truncate current log: %w", err)
	}

	// Ensure the file permissions are preserved
	if err := os.Chmod(r.config.FilePath, currentPerms); err != nil {
		return fmt.Errorf("failed to set permissions on new log file: %w", err)
	}

	return nil
}

func (r *StandardRotationManager) Cleanup() error {
	pattern := r.config.FilePath + ".*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob log files: %w", err)
	}

	for _, match := range matches {
		if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove log file %s: %w", match, err)
		}
	}
	return nil
}

func (r *StandardRotationManager) GetCurrentFile() (io.Writer, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(r.currentLog)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	return os.OpenFile(r.currentLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
}

// StandardBufferManager implements BufferManager
type StandardBufferManager struct {
	buf    chan []byte
	size   int
	closed bool
	mu     sync.RWMutex
}

func NewStandardBufferManager(size int) *StandardBufferManager {
	if size <= 0 {
		size = 1024 * 1024 // 1MB default
	}

	return &StandardBufferManager{
		buf:  make(chan []byte, size),
		size: size,
	}
}

func (b *StandardBufferManager) Write(p []byte) (n int, err error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return 0, fmt.Errorf("buffer manager is closed")
	}
	b.mu.RUnlock()

	select {
	case b.buf <- p:
		return len(p), nil
	default:
		return 0, ErrBufferFull
	}
}

func (b *StandardBufferManager) Flush() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return fmt.Errorf("buffer manager is closed")
	}

	// Clear the buffer
	for len(b.buf) > 0 {
		<-b.buf
	}

	return nil
}

func (b *StandardBufferManager) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	close(b.buf)
	return nil
}

func (b *StandardBufferManager) IsFull() bool {
	return len(b.buf) == cap(b.buf)
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Get source file mode
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	// Create destination file with same permissions
	destFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the contents
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return nil
}
