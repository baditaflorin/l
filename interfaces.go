// Package l provides a structured logging framework with interface-based components
package l

import (
	"context"
	"io"
	"log/slog"
	"time"
)

// Core interfaces

// Logger defines the main logging interface
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
	Debug(msg string, args ...any)
	With(args ...any) Logger
	Flush() error
	Close() error
	GetContext() context.Context
	WithContext(ctx context.Context) Logger
}

// Writer defines the interface for log output handling with lifecycle management
type Writer interface {
	io.Writer
	Lifecycle
	Flushable
}

// RotationManager handles log file rotation operations
type RotationManager interface {
	ShouldRotate(size int64) bool
	Rotate() error
	Cleanup() error
	GetCurrentFile() (io.Writer, error)
}

// BufferManager handles message buffering operations
type BufferManager interface {
	Write([]byte) (int, error)
	Lifecycle
	Flushable
	IsFull() bool
}

// MetricsCollector handles logging metrics collection and reporting
type MetricsCollector interface {
	IncrementTotal()
	IncrementErrors()
	IncrementDropped()
	SetLastError(time.Time)
	SetLastFlush(time.Time)
	GetMetrics() Metrics
	Reset()
}

// ErrorHandler manages error handling and recovery
type ErrorHandler interface {
	Handle(error)
	WithRecovery(fn func() error) error
}

// HandlerWrapper provides a wrapper for slog.Handler with lifecycle management
type HandlerWrapper interface {
	slog.Handler
	Lifecycle
}

// HealthChecker monitors logger health
type HealthChecker interface {
	Start(context.Context)
	Check() error
	Stop()
}

// Formatter handles log message formatting
type Formatter interface {
	Format(record LogRecord) ([]byte, error)
	WithOptions(opts FormatterOptions) Formatter
}

// Component interfaces

// Lifecycle defines common lifecycle methods
type Lifecycle interface {
	Close() error
}

// Flushable defines the flush operation
type Flushable interface {
	Flush() error
}

// Configurable defines configuration management
type Configurable interface {
	Configure(config Config) error
	GetConfig() Config
}

// Data structures

// LogRecord represents a single log entry
type LogRecord struct {
	Level     slog.Level
	Message   string
	Time      time.Time
	Source    string
	Args      []any
	Attrs     []slog.Attr
	Context   context.Context
	AddSource bool
}

// Metrics holds logger operational metrics
type Metrics struct {
	TotalMessages   uint64
	ErrorMessages   uint64
	DroppedMessages uint64
	LastError       time.Time
	LastFlush       time.Time
	BufferSize      int
	BufferUsage     float64
}

// Config holds logger configuration
type Config struct {
	MinLevel      slog.Level
	AddSource     bool
	JsonFormat    bool
	AsyncWrite    bool
	BufferSize    int
	MaxFileSize   int64
	MaxBackups    int
	ErrorCallback func(error)
	Metrics       bool
	Output        io.Writer
	FilePath      string
}

// FormatterOptions defines formatting options
type FormatterOptions struct {
	TimeFormat string
	UseColor   bool
	Indent     string
}

// Factory interface for creating logger components
type Factory interface {
	CreateLogger(config Config) (Logger, error)
	CreateWriter(config Config) (Writer, error)
	CreateFormatter(config Config) (Formatter, error)
	CreateMetricsCollector(config Config) (MetricsCollector, error)
	CreateRotationManager(config Config) (RotationManager, error)
	CreateErrorHandler(config Config) (ErrorHandler, error)
	CreateBufferManager(config Config) (BufferManager, error)
	CreateHandlerWrapper(handler slog.Handler, closer io.Closer) HandlerWrapper
	CreateHealthChecker(logger Logger) (HealthChecker, error)
}

// Helper interfaces for specific functionality

// Filterable defines log filtering capabilities
type Filterable interface {
	ShouldLog(record LogRecord) bool
}

// Enricher defines log enrichment capabilities
type Enricher interface {
	Enrich(record *LogRecord) error
}

// BatchProcessor defines batch processing capabilities
type BatchProcessor interface {
	ProcessBatch(records []LogRecord) error
	IsBatchReady() bool
}

// Validator defines validation capabilities
type Validator interface {
	Validate() error
}

// Context management interfaces

// ContextProvider defines context management
type ContextProvider interface {
	WithContext(ctx context.Context) Logger
	GetContext() context.Context
}

// FieldProvider defines structured field management
type FieldProvider interface {
	WithFields(fields map[string]interface{}) Logger
	GetFields() map[string]interface{}
}

// Extension interfaces for advanced functionality

// Sampler defines sampling capabilities
type Sampler interface {
	ShouldSample(record LogRecord) bool
	GetSamplingRate() float64
}

// RateLimiter defines rate limiting capabilities
type RateLimiter interface {
	Allow() bool
	GetLimit() int
	GetBurst() int
}

// Rotatable defines rotation capabilities
type Rotatable interface {
	ShouldRotate() bool
	Rotate() error
}

// Queryable defines query capabilities for logs
type Queryable interface {
	Query(filter FilterFunc, opts QueryOptions) ([]LogRecord, error)
}

// FilterFunc defines a function type for log filtering
type FilterFunc func(LogRecord) bool

// QueryOptions defines options for log querying
type QueryOptions struct {
	Limit     int
	Offset    int
	StartTime time.Time
	EndTime   time.Time
	OrderBy   string
	Order     string
}
