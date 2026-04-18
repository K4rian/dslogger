# dslogger
`dslogger` is a lightweight, production-ready Go logging library wrapping [zap][1] for fast structured logging and [lumberjack][2] for automatic log file rotation.

---

<p align="center">
<a href="#quick-start">Quick Start</a> &bull;
<a href="#advanced-configuration">Advanced Configuration</a> &bull;
<a href="#functional-options">Functional Options</a> &bull;
<a href="#constructors">Constructors</a>
<br>
<a href="#architecture">Architecture</a> &bull;
<a href="#limitations">Limitations</a> &bull;
<a href="#installation">Installation</a> &bull;
<a href="#license">License</a></p>
</p>

---

## Quick Start

### Console-only logger

Use `NewSimpleConsoleLogger` to quickly create a console-only logger:
```go
logger, err := dslogger.NewSimpleConsoleLogger("info")
if err != nil {
    log.Fatal(err)
}

logger.Info("Server started", "port", 8080)
logger.Warn("Slow query", "duration", "2.3s", "table", "users")
```

Output:
```
2026-01-15T10:30:00.000Z | INFO  | Server started | port: 8080
2026-01-15T10:30:00.001Z | WARN  | Slow query | duration: 2.3s | table: users
```

### Console + file logger

Use `NewSimpleLogger` to create a logger that writes to both the console and a file (`./app.log`) using default configuration:
```go
logger, err := dslogger.NewSimpleLogger("debug")
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

logger.Info("Request received", "method", "GET", "path", "/api/users")
```

Output:
```
2026-01-15T10:30:00.000Z | INFO  | Request received | method: GET | path: /api/users
```

Use `NewLogger` to create a logger that writes to both the console and a file using custom configuration:
```go
cfg := dslogger.NewDefaultConfig()
cfg.LogFile = "./my_app.log"
cfg.LogFileFormat = dslogger.LogFormatJSON

logger, err := dslogger.NewLogger("debug", &cfg)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

logger.Info("Request received", "method", "GET", "path", "/api/users")
```

While console output uses the human-readable format, the file receives structured JSON:
```json
{"timestamp":"2026-01-15T10:30:00.000Z","level":"INFO","message":"Request received","method":"GET","path":"/api/users"}
```

### Service-specific logger

The derived logger automatically attaches a `"service": "AuthService"` field to every log entry, making it easier to distinguish logs for different services in the application:
```go
logger, _ := dslogger.NewSimpleConsoleLogger("info")
authLogger := logger.WithService("AuthService")

authLogger.Info("Login successful", "user", "james")
```

Output:
```
2026-01-15T10:30:00.000Z | INFO  | [AuthService] Login successful | user: james
```

### Derived logger with fields

Both lines carry `request_id` and `trace_id`:
```go
reqLogger := logger.WithFields("request_id", "req-42", "trace_id", "abc123")
reqLogger.Info("Processing")
reqLogger.Error("Failed", "err", "timeout")
```

### Context integration

```go
ctx := context.WithValue(ctx, dslogger.RequestIDKey, "req-42")
ctxLogger := logger.WithContext(ctx)
ctxLogger.Info("Handled request")
```

OpenTelemetry spans are also extracted automatically:

```go
ctx, span := tracer.Start(ctx, "handleRequest")
defer span.End()
ctxLogger := logger.WithContext(ctx)
ctxLogger.Info("Traced request") // includes trace_id and span_id
```

### Fatal and Panic

```go
logger.Fatal("cannot connect to database", "err", err)
// logs at ERROR with [FATAL] prefix, flushes, then calls os.Exit(1)

logger.Panic("invariant violated", "state", s)
// logs at ERROR with [PANIC] prefix, then panics with the message
```

### Custom console writer

```go
cfg := dslogger.NewDefaultConfig()
cfg.ConsoleWriter = os.Stderr // redirect console output to stderr

// or capture to a buffer
var buf bytes.Buffer
cfg.ConsoleWriter = &buf
```

### JSON console output

```go
cfg := dslogger.NewDefaultConfig()
cfg.ConsoleFormat = dslogger.LogFormatJSON
// stdout now emits structured JSON, useful for platforms that ingest stdout as JSON
```

### slog bridge

```go
logger, _ := dslogger.NewSimpleConsoleLogger("info")
slogger := slog.New(dslogger.NewSlogHandler(logger))

slogger.Info("hello from slog", "count", 42)
slogger.WithGroup("http").Info("request", "method", "GET")
```

## Advanced Configuration

```go
cfg := &dslogger.Config{
    LogFile:               "./app.log",
    LogFileFormat:         dslogger.LogFormatJSON,
    MaxSize:               50,        // MB per file
    MaxBackups:            7,
    MaxAge:                60,        // days
    Compress:              true,
    FileMode:              0640,      // pre-create with these permissions
    NoColor:               false,     // auto-detected from TTY
    ForceColor:            false,     // set true for CI with ANSI support
    ConsoleConfig:         dslogger.DefaultConsoleEncoderConfig,
    FileConfig:            dslogger.DefaultJSONEncoderConfig,
    FieldSeparator:        "=",
    ConsoleSeparator:      "    ",
    ServiceNameDecorators: [2]string{"(", ")"},
}

logger, err := dslogger.NewLogger("debug", cfg,
    dslogger.WithServiceName("MyApp"),
    dslogger.WithCustomFields("version", "1.2.0"),
)
```

## Functional Options

Option                        | Description
---                           |---
`WithServiceName(name)`       | Set service name at construction
`WithCustomFields(k, v, ...)` | Attach fields to every log entry
`WithCustomField(k, v)`       | Attach a single field
`WithCallerSkip(n)`           | Adjust caller-skip for wrapper libraries
`WithConsoleEncoder(enc)`     | Replace the console encoder entirely
`WithFileEncoder(enc)`        | Replace the file encoder entirely
`WithCustomLevelFormats(map)` | Custom level strings and colours

## Constructors

Function                                 | Console | File  | Config
---                                      | ---     | ---   | ---
 `NewSimpleConsoleLogger(level)`         | Yes     | No    | Defaults
 `NewSimpleLogger(level)`                | Yes     | Yes   | Defaults
 `NewConsoleLogger(level, cfg, opts...)` | Yes     | No    | Custom
 `NewLogger(level, cfg, opts...)`        | Yes     | Yes   | Custom

## Architecture

- **Custom encoder** (`dsConsoleEncoder`) handles all console/text formatting inside `zapcore.Encoder.EncodeEntry`: no intermediate string allocations
- **Atomic hot path**: level gate is a single atomic load, both zap loggers sit behind `atomic.Pointer`. `SetLogLevel` is a one-line `AtomicLevel.SetLevel` with no core rebuild.
- **Deep-copy config**: user-supplied `*Config` is cloned at construction, no mutation of caller state.
- **Precomputed level strings**: ANSI colour and fixed-width formatting are computed once at encoder creation, not per log call.

## Limitations

- **Rotated file permissions**: `FileMode` controls the primary log file. Rotated backups use lumberjack's internal defaults.
- **`WithCustomLevelFormats` rebuilds cores**: safe but not recommended to call on a live logger under heavy traffic. Use it as a functional option at construction.

## Installation
```bash
go get github.com/K4rian/dslogger@latest
```
Requires **Go 1.26.2+**.

## License
[MIT][3]

[1]: https://github.com/uber-go/zap
[2]: https://github.com/natefinch/lumberjack
[3]: https://github.com/K4rian/dslogger/blob/main/LICENSE
