# l - Enhanced Structured Logging for Go

A high-performance, structured logging library for Go that builds upon `log/slog` with additional features like automatic error handling, log rotation, metrics collection, and asynchronous writing.

## Features

- Built on top of Go's standard `log/slog` package
- JSON and text output formats
- Automatic log file rotation
- Asynchronous writing support
- Built-in error recovery and retries
- Comprehensive logging metrics
- Stack trace collection for errors
- Source code location tracking
- Health checking and auto-recovery
- Buffered writing with configurable sizes
- Graceful shutdown handling

## Installation

```bash
go get github.com/baditaflorin/l
```

## Quick Start

```go
package main

import (
    "os"
    "github.com/baditaflorin/l"
)

func main() {
    err := l.Setup(l.Options{
        Output:     os.Stdout,
        MinLevel:   0,
        JsonFormat: true,
    })
    if err != nil {
        panic(err)
    }

    l.Info("Starting application", "version", "1.0.0")
    l.Error("Failed to process request", "code", 500)
}
```

## Configuration Options

```go
type Options struct {
    Output        io.Writer    // Output destination (default: os.Stdout)
    MinLevel      slog.Level   // Minimum logging level
    AddSource     bool         // Include source code location
    JsonFormat    bool         // Use JSON output format
    FilePath      string       // Log file path for file output
    MaxFileSize   int64        // Maximum size before rotation (default: 100MB)
    MaxBackups    int          // Number of backup files to keep (default: 5)
    BufferSize    int          // Size of async write buffer (default: 1MB)
    AsyncWrite    bool         // Enable asynchronous writing
    ErrorCallback func(error)  // Custom error handling callback
    Metrics      bool         // Enable metrics collection
}
```

## Advanced Usage Examples

### 1. File-based Logging with Rotation

```go
package main

import (
    "github.com/baditaflorin/l"
)

func main() {
    err := l.Setup(l.Options{
        FilePath:    "/var/log/myapp/app.log",
        MaxFileSize: 50 * 1024 * 1024, // 50MB
        MaxBackups:  3,
        JsonFormat:  true,
        AddSource:   true,
    })
    if err != nil {
        panic(err)
    }
    defer l.Close()

    l.Info("Application started", "pid", os.Getpid())
}
```

### 2. Asynchronous Logging with Custom Error Handling

```go
package main

import (
    "fmt"
    "github.com/baditaflorin/l"
)

func main() {
    err := l.Setup(l.Options{
        AsyncWrite: true,
        BufferSize: 2048,
        ErrorCallback: func(err error) {
            fmt.Printf("Logging error occurred: %v\n", err)
        },
    })
    if err != nil {
        panic(err)
    }
    defer l.Close()

    // Log messages will be written asynchronously
    for i := 0; i < 1000; i++ {
        l.Info("Processing item", "index", i)
    }
}
```

### 3. Metrics Collection and Monitoring

```go
package main

import (
    "github.com/baditaflorin/l"
    "time"
)

func main() {
    err := l.Setup(l.Options{
        Metrics: true,
        JsonFormat: true,
    })
    if err != nil {
        panic(err)
    }
    defer l.Close()

    // Log some messages
    l.Info("Starting batch process")
    l.Error("Process failed", "reason", "timeout")

    // Get metrics after some time
    time.Sleep(time.Second)
    metrics := l.GetMetrics()
    
    l.Info("Logging metrics",
        "total_messages", metrics.TotalMessages.Load(),
        "error_messages", metrics.ErrorMessages.Load(),
        "dropped_messages", metrics.DroppedMessages.Load(),
    )
}
```

## Best Practices

1. **Always use structured logging**: Instead of using string interpolation, pass key-value pairs:
   ```go
   // Good
   l.Info("User logged in", "user_id", 123, "ip", "192.168.1.1")
   
   // Avoid
   l.Info(fmt.Sprintf("User %d logged in from %s", userId, ip))
   ```

2. **Handle errors appropriately**: Always check the error returned by `Setup()`:
   ```go
   if err := l.Setup(options); err != nil {
       panic(err) // or handle appropriately
   }
   ```

3. **Use appropriate log levels**: Choose the correct log level based on the message importance:
    - `Debug`: Detailed information for debugging
    - `Info`: General operational information
    - `Warn`: Warning messages for potentially harmful situations
    - `Error`: Error messages for serious problems

4. **Clean up properly**: Always call `Close()` before your application exits:
   ```go
   defer l.Close()
   ```

## Performance Considerations

- Enable `AsyncWrite` for high-throughput logging scenarios
- Adjust `BufferSize` based on your application's logging volume
- Use `JsonFormat: true` for machine-readable logs
- Enable `Metrics` only when needed
- Consider log rotation settings based on your disk space constraints

## Thread Safety

The logger is designed to be thread-safe and can be safely used from multiple goroutines concurrently.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.