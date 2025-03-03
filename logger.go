package dslogger

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Option represents a functional option for configuring a Logger.
type Option func(*Logger) error

// Logger is a configurable logger that supports output to console and file.
// It uses zap for logging and lumberjack for file rotation.
type Logger struct {
	config           *Config
	consoleLogger    *zap.SugaredLogger
	fileLogger       *zap.SugaredLogger
	lumberjackLogger *lumberjack.Logger
	serviceName      string
	level            int32
	customFields     []zap.Field
	logConsoleFuncs  map[string]func(...any)
	logTextFuncs     map[string]func(...any)
	logJSONFuncs     map[string]func(string, ...any)
	mu               sync.Mutex
}

// Config returns the current logger configuration.
func (l *Logger) Config() *Config {
	return l.config
}

// ConsoleLogger returns the underlying console logger instance.
func (l *Logger) ConsoleLogger() *zap.SugaredLogger {
	return l.consoleLogger
}

// FileLogger returns the underlying file logger instance.
func (l *Logger) FileLogger() *zap.SugaredLogger {
	return l.fileLogger
}

func (l *Logger) LumberjackLogger() *lumberjack.Logger {
	return l.lumberjackLogger
}

// ServiceName returns the name of the service associated with the logger.
func (l *Logger) ServiceName() string {
	return l.serviceName
}

// Level retrieves the current logging level (atomic).
func (l *Logger) Level() zapcore.Level {
	return zapcore.Level(atomic.LoadInt32(&l.level))
}

// Fields retrieves the custom zap fields.
func (l *Logger) Fields() []zap.Field {
	return l.customFields
}

// SetLogLevel updates the log level for both console and file loggers.
// It returns an error if updating any of the loggers fails.
func (l *Logger) SetLogLevel(level string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	zapLevel := parseLogLevel(level)
	if atomic.LoadInt32(&l.level) == int32(zapLevel) {
		return nil
	}
	atomic.StoreInt32(&l.level, int32(zapLevel))

	newConsole, err := newConsoleLogger(l.config, zapLevel)
	if err != nil {
		return fmt.Errorf("failed to update console logger: %w", err)
	}
	l.consoleLogger = newConsole

	if l.fileLogger != nil {
		newFile, err := newFileLogger(l.config, zapLevel, l.lumberjackLogger)
		if err != nil {
			return fmt.Errorf("failed to update file logger: %w", err)
		}
		l.fileLogger = newFile
	}
	initializeLogFuncMaps(l)
	return nil
}

// ClearFields clears all custom zap fields.
func (l *Logger) ClearFields() {
	l.mu.Lock()
	defer l.mu.Unlock()
	clear(l.customFields)
}

// WithContext returns a new logger instance that attaches common fields (request_id, trace_id)
// extracted from the provided context.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	requestID, _ := ctx.Value("request_id").(string)
	traceID, _ := ctx.Value("trace_id").(string)
	if requestID != "" || traceID != "" {
		return l.WithFields("request_id", requestID, "trace_id", traceID)
	}
	return l
}

// WithFields returns a new logger instance with the specified structured fields attached.
func (l *Logger) WithFields(fields ...any) *Logger {
	if len(fields)%2 != 0 {
		panic(fmt.Sprintf("WithFields requires an even number of arguments (key-value pairs), got %d", len(fields)))
	}

	zapFields := []zap.Field{}
	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			panic(fmt.Sprintf("WithFields: field key must be a string, got %T", fields[i]))
		}
		value := fields[i+1]
		zapFields = append(zapFields, zap.Any(key, value))
	}

	newLogger := &Logger{
		config:           l.config,
		consoleLogger:    l.consoleLogger,
		fileLogger:       l.fileLogger,
		lumberjackLogger: l.lumberjackLogger,
		serviceName:      l.serviceName,
		customFields:     l.customFields,
	}

	if len(zapFields) > 0 {
		newLogger.customFields = append(newLogger.customFields, zapFields...)
	}
	atomic.StoreInt32(&newLogger.level, atomic.LoadInt32(&l.level))
	initializeLogFuncMaps(newLogger)
	return newLogger
}

// WithService creates a new logger instance that includes a "service" field in its log entries.
// Additional zap.Options can be provided to customize the underlying logger.
func (l *Logger) WithService(serviceName string, options ...zap.Option) *Logger {
	newConsole := l.consoleLogger.Desugar().WithOptions(options...).Sugar()

	var newFile *zap.SugaredLogger
	if l.fileLogger != nil {
		baseLogger := l.fileLogger.Desugar().WithOptions(options...)
		if l.config.LogFileFormat == LogFormatJSON {
			newFile = baseLogger.With(zap.String("service", serviceName)).Sugar()
		} else {
			newFile = baseLogger.Sugar()
		}
	}
	newLogger := &Logger{
		config:           l.config,
		consoleLogger:    newConsole,
		fileLogger:       newFile,
		lumberjackLogger: l.lumberjackLogger,
		serviceName:      serviceName,
		customFields:     l.customFields,
	}
	atomic.StoreInt32(&newLogger.level, atomic.LoadInt32(&l.level))
	initializeLogFuncMaps(newLogger)
	return newLogger
}

// Info logs an informational message along with optional structured fields.
func (l *Logger) Info(msg string, fields ...any) {
	l.logMessage(zapcore.InfoLevel.String(), msg, fields...)
}

// Debug logs a debug-level message along with optional structured fields.
func (l *Logger) Debug(msg string, fields ...any) {
	l.logMessage(zapcore.DebugLevel.String(), msg, fields...)
}

// Warn logs a warning message along with optional structured fields.
func (l *Logger) Warn(msg string, fields ...any) {
	l.logMessage(zapcore.WarnLevel.String(), msg, fields...)
}

// Error logs an error message along with optional structured fields.
func (l *Logger) Error(msg string, fields ...any) {
	l.logMessage(zapcore.ErrorLevel.String(), msg, fields...)
}

// logMessage is an internal helper that logs a message at the given level to both console and file outputs.
func (l *Logger) logMessage(level, msg string, fields ...any) {
	if !isLogFuncMapsInitialized(l) {
		initializeLogFuncMaps(l)
	}

	formattedMsg := l.formatConsoleMessage(msg, fields...)

	// Log to console
	if logFunc, ok := l.logConsoleFuncs[level]; ok {
		logFunc(formattedMsg)
	}

	// Log to file
	if l.fileLogger != nil {
		// For JSON file format, use the raw message so that Zap's encoder handles fields properly
		if l.config.LogFileFormat == LogFormatJSON {
			formattedMsg = msg
		}
		l.logToFile(level, formattedMsg, fields...)
	}
}

// logToFile logs a message to the file output using either text or JSON format.
func (l *Logger) logToFile(level, msg string, fields ...any) {
	if l.config.LogFileFormat == LogFormatText {
		if logFunc, ok := l.logTextFuncs[level]; ok && logFunc != nil {
			logFunc(msg)
		}
	} else {
		if logFunc, ok := l.logJSONFuncs[level]; ok && logFunc != nil {
			logFunc(msg, fields...)
		}
	}
}
