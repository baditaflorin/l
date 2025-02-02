# Advanced Structured Logging Framework for Go

A high-performance, feature-rich structured logging framework for Go applications with support for asynchronous writes, log rotation, Metrics collection, and more.

## Features

- **Structured Logging**: JSON and text format support with key-value pairs
- **Asynchronous Writing**: Non-blocking log writes with configurable buffer size
- **Log Rotation**: Automatic file rotation based on size with configurable backups
- **Metrics Collection**: Built-in logging Metrics and performance monitoring
- **Health Checking**: Integrated health monitoring system
- **Error Handling**: Comprehensive error handling with panic recovery
- **Context Support**: Context propagation through logging chain
- **Flexible Output**: Support for multiple output formats and destinations
- **Performance Optimized**: High-throughput logging with batched writes
- **Extensible**: Interface-based design for easy customization

## Installation

```bash
go get github.com/baditaflorin/l
```

## Quick Start

```go
package main

import (
    "github.com/baditaflorin/l"
)

func main() {
    // Use default logger
    l.Info("Hello, World!")
    
    // Log with context
    l.Info("User logged in",
        "user_id", "123",
        "ip", "192.168.1.1",
    )
}
```

## Configuration

Create a custom logger with specific configuration:

```go
config := l.Config{
    Output:      os.Stdout,
    FilePath:    "logs/app.log",
    JsonFormat:  true,
    AsyncWrite:  true,
    BufferSize:  1024 * 1024,      // 1MB buffer
    MaxFileSize: 10 * 1024 * 1024, // 10MB max file size
    MaxBackups:  5,
    AddSource:   true,
    Metrics:     true,
}

factory := l.NewStandardFactory()
logger, err := factory.CreateLogger(config)
if err != nil {
    panic(err)
}
defer logger.Close()
```

## Advanced Usage

### Structured Logging with Context

```go
logger := l.With(
    "service", "auth-service",
    "environment", "production",
)

logger.Info("Authentication attempt",
    "user_id", "123",
    "success", true,
    "method", "oauth2",
)
```

### Error Handling

```go
logger.Error("Database connection failed",
    "error", err,
    "retry_count", retries,
    "database", "users",
)
```

### Log Rotation

```go
config := l.Config{
    FilePath:    "logs/app.log",
    MaxFileSize: 10 * 1024 * 1024, // 10MB
    MaxBackups:  5,                // Keep 5 backups
}
```

### Metrics Collection

```go
Metrics, err := factory.CreateMetricsCollector(config)
if err != nil {
    panic(err)
}

// Get Metrics
stats := Metrics.GetMetrics()
fmt.Printf("Total Messages: %d\n", stats.TotalMessages)
fmt.Printf("Error Messages: %d\n", stats.ErrorMessages)
```

### Health Checking

```go
healthChecker, err := factory.CreateHealthChecker(logger)
if err != nil {
    panic(err)
}

healthChecker.Start(context.Background())
defer healthChecker.Stop()

if err := healthChecker.Check(); err != nil {
    logger.Error("Health check failed", "error", err)
}
```

## Examples

The library includes several example applications demonstrating different use cases:

- Basic logging (`example/simple`)
- Web service logging (`example/web-service`)
- Audit trail logging (`example/audit-trail`)
- High-throughput logging (`example/high-throughput`)
- Worker pool logging (`example/worker-pool`)
- Metrics dashboard (`example/Metrics-dashboard`)
- Error handling (`example/error-handling`)
- Log rotation (`example/log-rotation`)
- Security logging (`example/security-logging`)
- Performance logging (`example/performance-logging`)
- Structured logging (`example/structured-logging`)

## Performance

The library is designed for high performance with features like:

- Asynchronous writing with buffering
- Batch processing of log messages
- Efficient JSON encoding
- Minimal allocations
- Concurrent-safe operations

## Best Practices

1. **Always defer Close()**:
   ```go
   logger, err := factory.CreateLogger(config)
   if err != nil {
       panic(err)
   }
   defer logger.Close()
   ```

2. **Use structured logging**:
   ```go
   // Good
   logger.Info("User action", "user_id", userID, "action", action)
   
   // Avoid
   logger.Info(fmt.Sprintf("User %s performed action %s", userID, action))
   ```

3. **Include context**:
   ```go
   logger = logger.With(
       "service", serviceName,
       "version", version,
       "environment", env,
   )
   ```

4. **Handle errors appropriately**:
   ```go
   if err := logger.Flush(); err != nil {
       // Handle error
   }
   ```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Testing

Run the test suite:

```bash
./run_coverage.sh
```

This will run all tests and generate a coverage report in the `coverage` directory.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support, please open an issue in the GitHub repository or contact the maintainers.