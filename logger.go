package dslogger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// osExit is os.Exit by default.
// Overridden in tests to avoid killing the test process.
var osExit = os.Exit

// Option represents a functional option for configuring a Logger.
type Option func(*Logger) error

// ctxKey is the type used for dslogger's own context keys, preventing collisions
// with string keys from other packages.
type ctxKey string

const (
	// RequestIDKey is the context key used by WithContext to extract a request ID
	RequestIDKey ctxKey = "request_id"
	// TraceIDKey is the context key used by WithContext to extract a trace ID
	TraceIDKey ctxKey = "trace_id"
	// SpanIDKey is the context key used by WithContext when extracting an OpenTelemetry span
	SpanIDKey ctxKey = "span_id"
)

// dsloggerCallerSkip is the number of stack frames between the user's call and the underlying
// zap SugaredLogger call site.
// Path: user -> Logger.Info -> logMessage -> logStructured -> SugaredLogger.Infow = skip of 3.
const dsloggerCallerSkip = 3

// Logger is a configurable logger safe for concurrent use, supporting console and file output.
// The underlying zap loggers are stored behind an atomic.Pointer so that SetLogLevel and
// other option updates do not race with concurrent log calls. A mutex serializes infrequent
// rebuilds to avoid conflicting mutations.
type Logger struct {
	config           *Config
	consoleLogger    atomic.Pointer[zap.SugaredLogger]
	fileLogger       atomic.Pointer[zap.SugaredLogger]
	lumberjackLogger *lumberjack.Logger
	level            zap.AtomicLevel
	serviceName      string
	customFields     []zap.Field // tracks fields for the Fields() getter
	mu               sync.Mutex
}

// Config returns the current logger configuration.
// The returned value is owned by the logger and must not be mutated.
func (l *Logger) Config() *Config {
	return l.config
}

// ConsoleLogger returns the underlying console logger instance.
// Advanced use only, calls bypass service-name decoration and custom-field formatting.
func (l *Logger) ConsoleLogger() *zap.SugaredLogger {
	return l.consoleLogger.Load()
}

// FileLogger returns the underlying file logger instance.
// Advanced use only, calls bypass service-name decoration and custom-field formatting
// on the text-file path.
func (l *Logger) FileLogger() *zap.SugaredLogger {
	return l.fileLogger.Load()
}

// LumberjackLogger returns the lumberjack.Logger managing rotation, or nil if file logging is disabled.
func (l *Logger) LumberjackLogger() *lumberjack.Logger {
	return l.lumberjackLogger
}

// ServiceName returns the name of the service associated with the logger.
func (l *Logger) ServiceName() string {
	return l.serviceName
}

// Level retrieves the current logging level (atomic, no lock).
func (l *Logger) Level() zapcore.Level {
	return l.level.Level()
}

// Fields retrieves a copy of the custom zap fields attached to this logger.
func (l *Logger) Fields() []zap.Field {
	return slices.Clone(l.customFields)
}

// SetLogLevel updates the log level atomically.
// Both the console and file cores share a single zap.AtomicLevel,
// so there is no core rebuild and no race.
func (l *Logger) SetLogLevel(level string) error {
	parsed, err := parseLogLevel(level)
	if err != nil {
		return err
	}
	l.level.SetLevel(parsed)
	return nil
}

// WithContext returns a new logger that attaches common fields extracted
// from the provided context:
//   - dslogger.RequestIDKey and dslogger.TraceIDKey (typed context keys)
//   - OpenTelemetry trace.SpanContext (trace_id, span_id) if the span is valid
//
// String-keyed context values are ignored to avoid cross-package collisions.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	kv := make([]any, 0, 8)

	// Typed dslogger keys
	if requestID, _ := ctx.Value(RequestIDKey).(string); requestID != "" {
		kv = append(kv, string(RequestIDKey), requestID)
	}
	if traceID, _ := ctx.Value(TraceIDKey).(string); traceID != "" {
		kv = append(kv, string(TraceIDKey), traceID)
	}

	// OpenTelemetry span context
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		if sc.HasTraceID() {
			kv = append(kv, string(TraceIDKey), sc.TraceID().String())
		}
		if sc.HasSpanID() {
			kv = append(kv, string(SpanIDKey), sc.SpanID().String())
		}
	}

	if len(kv) == 0 {
		return l
	}
	return l.WithFields(kv...)
}

// WithFields returns a new logger with the specified structured fields attached.
// It tolerates odd field counts and non-string keys by inserting a placeholder
// instead of panicking, so a single bad call site cannot crash the process.
// The returned logger has its own copy of the fields slice (no aliasing).
func (l *Logger) WithFields(fields ...any) *Logger {
	fields = normalizeFields(fields)

	zapFields := make([]zap.Field, 0, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", fields[i])
		}
		zapFields = append(zapFields, zap.Any(key, fields[i+1]))
	}

	newFields := make([]zap.Field, 0, len(l.customFields)+len(zapFields))
	newFields = append(newFields, l.customFields...)
	newFields = append(newFields, zapFields...)

	newLogger := &Logger{
		config:           l.config,
		lumberjackLogger: l.lumberjackLogger,
		level:            l.level,
		serviceName:      l.serviceName,
		customFields:     newFields,
	}
	if c := l.consoleLogger.Load(); c != nil {
		newLogger.consoleLogger.Store(c.Desugar().With(zapFields...).Sugar())
	}
	if f := l.fileLogger.Load(); f != nil {
		newLogger.fileLogger.Store(f.Desugar().With(zapFields...).Sugar())
	}
	return newLogger
}

// WithService creates a new logger that includes a "service" field in its log entries.
// For the console and text-file paths the service name is rendered using ServiceNameDecorators
// via the custom encoder. For JSON file output it is added as a structured "service" field.
// Additional zap.Options can be provided to customize the underlying logger.
func (l *Logger) WithService(serviceName string, options ...zap.Option) *Logger {
	newLogger := &Logger{
		config:           l.config,
		lumberjackLogger: l.lumberjackLogger,
		level:            l.level,
		serviceName:      serviceName,
		customFields:     slices.Clone(l.customFields),
	}

	zapOpts := []zap.Option{zap.AddCaller(), zap.AddCallerSkip(dsloggerCallerSkip)}
	zapOpts = append(zapOpts, options...)

	// Console: rebuild with the service name baked into the encoder
	var consEncoder zapcore.Encoder
	if l.config.ConsoleFormat == LogFormatJSON {
		consEncoder = zapcore.NewJSONEncoder(l.config.ConsoleConfig)
	} else {
		consEncoder = newDSConsoleEncoder(l.config, l.config.ConsoleConfig, serviceName)
	}

	consWriter := zapcore.Lock(zapcore.AddSync(l.config.consoleOut()))
	consCore := zapcore.NewCore(consEncoder, consWriter, l.level)
	consSugar := zap.New(consCore, zapOpts...).Sugar()

	// For JSON console, add service as a structured field
	if l.config.ConsoleFormat == LogFormatJSON && serviceName != "" {
		consSugar = consSugar.Desugar().With(zap.String("service", serviceName)).Sugar()
	}
	if len(l.customFields) > 0 {
		consSugar = consSugar.Desugar().With(l.customFields...).Sugar()
	}
	newLogger.consoleLogger.Store(consSugar)

	// File
	if f := l.fileLogger.Load(); f != nil && l.lumberjackLogger != nil {
		if l.config.LogFileFormat == LogFormatJSON {
			base := f.Desugar().WithOptions(options...)
			newLogger.fileLogger.Store(base.With(zap.String("service", serviceName)).Sugar())
		} else {
			fileEncoder := newDSConsoleEncoder(l.config, l.config.FileConfig, serviceName)
			fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(l.lumberjackLogger), l.level)
			fileSugar := zap.New(fileCore, zapOpts...).Sugar()
			if len(l.customFields) > 0 {
				fileSugar = fileSugar.Desugar().With(l.customFields...).Sugar()
			}
			newLogger.fileLogger.Store(fileSugar)
		}
	}

	return newLogger
}

// Info logs an informational message along with optional structured fields.
func (l *Logger) Info(msg string, fields ...any) {
	l.logMessage(zapcore.InfoLevel, msg, fields...)
}

// Debug logs a debug-level message along with optional structured fields.
func (l *Logger) Debug(msg string, fields ...any) {
	l.logMessage(zapcore.DebugLevel, msg, fields...)
}

// Warn logs a warning message along with optional structured fields.
func (l *Logger) Warn(msg string, fields ...any) {
	l.logMessage(zapcore.WarnLevel, msg, fields...)
}

// Error logs an error message along with optional structured fields.
func (l *Logger) Error(msg string, fields ...any) {
	l.logMessage(zapcore.ErrorLevel, msg, fields...)
}

// Fatal logs a message at error level with a [FATAL] tag, flushes buffered output,
// then calls os.Exit(1).
// Deferred functions are NOT run.
func (l *Logger) Fatal(msg string, fields ...any) {
	l.logMessage(zapcore.ErrorLevel, "[FATAL] "+msg, fields...)
	_ = l.Sync()
	osExit(1)
}

// Panic logs a message at error level with a [PANIC] tag, then panics with the message.
// The panic value is the message string.
// Fields are logged but not included in the panic value.
func (l *Logger) Panic(msg string, fields ...any) {
	l.logMessage(zapcore.ErrorLevel, "[PANIC] "+msg, fields...)
	panic(msg)
}

// Sync flushes any buffered log entries to the underlying writers.
// Callers should defer logger.Sync() at program exit to avoid losing recent log lines.
// Errors returned by Sync on non-file writers (stdout/stderr on most platforms) are filtered.
func (l *Logger) Sync() error {
	var errs []error

	if c := l.consoleLogger.Load(); c != nil {
		if err := c.Sync(); err != nil && !isIgnorableSyncError(err) {
			errs = append(errs, err)
		}
	}
	if f := l.fileLogger.Load(); f != nil {
		if err := f.Sync(); err != nil && !isIgnorableSyncError(err) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Close flushes outstanding writes and releases the file handle held by lumberjack (if any).
// After Close the logger should not be used.
func (l *Logger) Close() error {
	syncErr := l.Sync()

	if l.lumberjackLogger != nil {
		if err := l.lumberjackLogger.Close(); err != nil {
			return errors.Join(syncErr, err)
		}
	}
	return syncErr
}

// logMessage is the hot path. It gates on the current level BEFORE any formatting,
// so calls at a disabled level are effectively free (one atomic load).
// All formatting is handled by the underlying encoder, the console/text path uses
// dsConsoleEncoder, and the JSON file path uses zap's stock JSON encoder.
func (l *Logger) logMessage(lvl zapcore.Level, msg string, fields ...any) {
	if !l.level.Enabled(lvl) {
		return
	}
	fields = normalizeFields(fields)

	if cons := l.consoleLogger.Load(); cons != nil {
		logStructured(cons, lvl, msg, fields...)
	}

	if f := l.fileLogger.Load(); f != nil {
		logStructured(f, lvl, msg, fields...)
	}
}

func logStructured(s *zap.SugaredLogger, lvl zapcore.Level, msg string, kv ...any) {
	switch lvl {
	case zapcore.DebugLevel:
		s.Debugw(msg, kv...)
	case zapcore.InfoLevel:
		s.Infow(msg, kv...)
	case zapcore.WarnLevel:
		s.Warnw(msg, kv...)
	case zapcore.ErrorLevel:
		s.Errorw(msg, kv...)
	}
}

// normalizeFields ensures an even number of args, appending a placeholder value
// if the caller passed an odd count. This is more permissive than panicking and
// matches zap's SugaredLogger DPanic behavior without the panic.
func normalizeFields(fields []any) []any {
	if len(fields)%2 == 0 {
		return fields
	}
	return append(fields, "<missing>")
}

// isIgnorableSyncError reports whether err is a benign sync error from stdout/stderr.
// On most Unix systems, os.Stdout.Sync returns EINVAL or ENOTTY, these are not real
// errors and should not be surfaced to callers.
func isIgnorableSyncError(err error) bool {
	if err == nil {
		return true
	}

	msg := err.Error()
	return strings.Contains(msg, "/dev/stdout") ||
		strings.Contains(msg, "/dev/stderr") ||
		strings.Contains(msg, "invalid argument") ||
		strings.Contains(msg, "inappropriate ioctl for device")
}
